package service

import (
	"context"; "crypto/rand"; "crypto/sha256"; "encoding/hex"; "fmt"; "regexp"; "sort"; "strings"; "time"

	"facefinder/internal/config"; "facefinder/internal/face"; "facefinder/internal/model"; "facefinder/internal/storage"; "facefinder/internal/store"; "facefinder/internal/whatsapp"
)

type Service struct {
	Cfg config.Config; Store *store.Store; Face face.Provider; Storage storage.Provider; WA whatsapp.Provider
	Enqueue func(kind string, payload map[string]string) error // set by jobs package; nil = run best-effort inline disabled
}
func Hash(s string) string { h:=sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func Token(n int) string { b:=make([]byte,n); _,_=rand.Read(b); return hex.EncodeToString(b) }
var codeRe = regexp.MustCompile(`[^A-Z0-9]`)
func EventCode(name string) string {
	p := strings.ToUpper(name); if len(p)>3 {p=p[:3]}; p = codeRe.ReplaceAllString(p,""); if p=="" {p="EVT"}
	return p+"-"+strings.ToUpper(Token(2))
}
func (s *Service) WhatsAppLink(token string) string {
	num := s.Cfg.WABusinessNumber
	return fmt.Sprintf("https://wa.me/%s?text=%s", num, "START%20"+token)
}
func (s *Service) CreateEvent(ctx context.Context, orgID, userID string, in model.Event) (model.Event, error) {
	token := Token(24)
	in.OrganizationID = orgID; in.EventCode = EventCode(in.Name)
	in.RekognitionCollectionID = "evt_" + strings.ReplaceAll(Token(8), "-", "")
	if in.GalleryTTLHours==0 {in.GalleryTTLHours=s.Cfg.GalleryTTLHours}
	if in.WhatsAppSessionTTLHours==0 {in.WhatsAppSessionTTLHours=s.Cfg.SessionTTLHours}
	if in.SelfieRetentionMinutes==0 {in.SelfieRetentionMinutes=s.Cfg.SelfieRetentionMin}
	now := time.Now()
	if in.FaceDataExpiresAt==nil { t:=now.AddDate(0,0,s.Cfg.FaceRetentionDays); in.FaceDataExpiresAt=&t }
	if in.PhotosExpireAt==nil { t:=now.AddDate(0,0,s.Cfg.PhotoRetentionDays); in.PhotosExpireAt=&t }
	if err := s.Face.CreateCollection(ctx, in.RekognitionCollectionID); err != nil { return in, fmt.Errorf("create rekognition collection: %w", err) }
	ev, err := s.Store.CreateEvent(ctx, in, Hash(token))
	if err != nil { _ = s.Face.DeleteCollection(ctx, in.RekognitionCollectionID); return ev, err }
	ev.EventToken = token; ev.WhatsAppLink = s.WhatsAppLink(token)
	s.Store.Audit(ctx, orgID, ev.ID, userID, "event.created", map[string]any{"code":ev.EventCode})
	return ev, nil
}
func (s *Service) DeleteEvent(ctx context.Context, orgID, userID, id string) error {
	ev, err := s.Store.EventByID(ctx, orgID, id); if err != nil { return err }
	_ = s.Face.DeleteCollection(ctx, ev.RekognitionCollectionID)
	_ = s.Storage.DeletePrefix(ctx, "events/"+ev.ID+"/")
	_ = s.Storage.DeletePrefix(ctx, "search-selfies/"+ev.ID+"/")
	s.Store.Audit(ctx, orgID, id, userID, "event.deleted", nil)
	return s.Store.DeleteEventRow(ctx, orgID, id)
}
func (s *Service) RegenerateToken(ctx context.Context, orgID, userID, id string) (string, string, error) {
	if _, err := s.Store.EventByID(ctx, orgID, id); err != nil { return "", "", err }
	t := Token(24); return t, s.WhatsAppLink(t), s.Store.UpdateEventToken(ctx, orgID, id, Hash(t))
}

// ---- Photo pipeline (called by worker) ----
func (s *Service) ProcessPhoto(ctx context.Context, photoID string) error {
	p, err := s.Store.PhotoByID(ctx, photoID); if err != nil { return err }
	ev, err := s.Store.EventByIDAnyOrg(ctx, p.EventID); if err != nil { return err }
	_ = s.Store.SetPhotoStatus(ctx, photoID, "processing", 0, "")
	raw, err := s.Storage.Get(ctx, p.OriginalKey); if err != nil { _ = s.Store.SetPhotoStatus(ctx,photoID,"failed",0,err.Error()); return err }
	web, thumb, analysis, w, h, err := MakeVariants(raw)
	if err != nil { _ = s.Store.SetPhotoStatus(ctx,photoID,"failed",0,"corrupt or unsupported image"); return err }
	webK := fmt.Sprintf("events/%s/web/%s.jpg", p.EventID, p.ID); thumbK := fmt.Sprintf("events/%s/thumb/%s.jpg", p.EventID, p.ID); anaK := fmt.Sprintf("events/%s/analysis/%s.jpg", p.EventID, p.ID)
	if err := s.Storage.Put(ctx, webK, web, "image/jpeg"); err != nil { return err }
	if err := s.Storage.Put(ctx, thumbK, thumb, "image/jpeg"); err != nil { return err }
	if err := s.Storage.Put(ctx, anaK, analysis, "image/jpeg"); err != nil { return err }
	_ = s.Store.SetPhotoVariants(ctx, photoID, webK, thumbK, anaK, w, h)
	faces, err := s.Face.IndexPhoto(ctx, ev.RekognitionCollectionID, analysis, p.ID)
	if err != nil { _ = s.Store.SetPhotoStatus(ctx,photoID,"failed",0,err.Error()); return err }
	if len(faces)==0 { return s.Store.SetPhotoStatus(ctx, photoID, "no_face", 0, "") }
	for _, f := range faces { if err := s.Store.InsertFace(ctx, p.EventID, p.ID, model.Face{AWSFaceID:f.FaceID, AWSImageID:f.ImageID, BBoxJSON:f.BBoxJSON, Confidence:f.Confidence}); err != nil { return err } }
	return s.Store.SetPhotoStatus(ctx, photoID, "ready", len(faces), "")
}

// ---- Guest search (§17–23) ----
type SearchOutcome struct { SearchID, GalleryURL string; Results []model.SearchResult; Message string }
func (s *Service) ProcessSelfie(ctx context.Context, ev model.Event, phone string, img []byte) (SearchOutcome, error) {
	searchID, err := s.Store.CreateSearch(ctx, ev.ID, phone, ""); if err != nil { return SearchOutcome{}, err }
	out := SearchOutcome{SearchID: searchID}
	key := fmt.Sprintf("search-selfies/%s/%s.jpg", ev.ID, searchID)
	_ = s.Storage.Put(ctx, key, img, "image/jpeg")
	det, err := s.Face.DetectFaces(ctx, img)
	if err != nil { _ = s.Store.FinishSearch(ctx, searchID, "failed", 0, err.Error()); return out, err }
	if det.Count==0 { _ = s.Store.FinishSearch(ctx, searchID, "no_face", 0, ""); out.Message="No clear face was detected. Please send a clear front-facing selfie."; return out, nil }
	if det.Count>1 { _ = s.Store.FinishSearch(ctx, searchID, "multiple_faces", det.Count, ""); out.Message="Please send a selfie containing only one person."; return out, nil }
	matches, err := s.Face.SearchFace(ctx, ev.RekognitionCollectionID, img, s.Cfg.SearchThreshold, 200)
	if err != nil { _ = s.Store.FinishSearch(ctx, searchID, "failed", 1, err.Error()); return out, err }
	best := map[string]model.SearchResult{}
	if len(matches)>0 {
		ids := make([]string, len(matches)); for i,m := range matches { ids[i]=m.FaceID }
		face2photo, err := s.Store.PhotosForFaceIDs(ctx, ev.ID, ids); if err != nil { return out, err }
		for _, m := range matches {
			pid, ok := face2photo[m.FaceID]; if !ok { continue }
			typ := "possible"; if m.Similarity >= s.Cfg.ThresholdStrong { typ="strong" } else if m.Similarity < s.Cfg.ThresholdPossible { continue }
			if cur, ok := best[pid]; !ok || m.Similarity > cur.Similarity { best[pid] = model.SearchResult{PhotoID:pid, BestFaceID:m.FaceID, Similarity:m.Similarity, Type:typ} }
		}
	}
	results := make([]model.SearchResult,0,len(best)); for _,r := range best { results=append(results,r) }
	sort.Slice(results, func(i,j int) bool { return results[i].Similarity > results[j].Similarity })
	for i := range results { results[i].Rank=i+1; _ = s.Store.InsertResult(ctx, searchID, results[i]) }
	out.Results = results
	if len(results)==0 { _ = s.Store.FinishSearch(ctx, searchID, "no_matches", 1, ""); out.Message="We couldn't find matching photos yet. Please try another clearer selfie."; return out, nil }
	_ = s.Store.FinishSearch(ctx, searchID, "completed", 1, "")
	gt := Token(32)
	if err := s.Store.CreateGallery(ctx, searchID, ev.ID, phone, Hash(gt), ev.GalleryTTLHours); err != nil { return out, err }
	out.GalleryURL = s.Cfg.PublicWebURL + "/g/" + gt
	// WhatsApp delivery: up to 5 previews (§22)
	previews := results; if len(previews)>5 { previews=previews[:5] }
	for _, r := range previews {
		p, err := s.Store.PhotoByID(ctx, r.PhotoID); if err!=nil || p.WebKey==nil { continue }
		url, err := s.Storage.PresignDownload(ctx, *p.WebKey, 15*time.Minute); if err!=nil { continue }
		_ = s.WA.SendImage(ctx, phone, url, ev.Name)
	}
	_ = s.WA.SendInteractive(ctx, phone, fmt.Sprintf("We found %d photos of you in %s.", len(results), ev.Name), map[string]string{"view_all": out.GalleryURL, "search_again":"SEARCH_AGAIN", "change_event":"CHANGE_EVENT"})
	if s.Enqueue != nil { _ = s.Enqueue("search.cleanup_selfie", map[string]string{"search_id":searchID}) }
	return out, nil
}

// ---- Retention (§31–34) ----
func (s *Service) SweepRetention(ctx context.Context) error {
	if evs, err := s.Store.ExpiredFaceEvents(ctx); err==nil { for _, ev := range evs {
		if err := s.Face.DeleteCollection(ctx, ev.RekognitionCollectionID); err!=nil { continue } // retry next sweep; never delete R2 here
		_ = s.Store.MarkFaceCollectionDeleted(ctx, ev.ID); s.Store.Audit(ctx, ev.OrganizationID, ev.ID, "", "retention.face_collection_deleted", nil)
	}}
	if evs, err := s.Store.ExpiredPhotoEvents(ctx); err==nil { for _, ev := range evs {
		_ = s.Storage.DeletePrefix(ctx, "events/"+ev.ID+"/"); _ = s.Store.MarkPhotosDeleted(ctx, ev.ID)
		s.Store.Audit(ctx, ev.OrganizationID, ev.ID, "", "retention.photos_deleted", nil)
	}}
	if keys, err := s.Store.ExpiredSelfies(ctx); err==nil && len(keys)>0 {
		for _, k := range keys { _ = s.Storage.Delete(ctx, k) }; _ = s.Store.MarkSelfiesDeleted(ctx, keys)
	}
	return nil
}
func (s *Service) CleanupSelfie(ctx context.Context, searchID string) error {
	// Immediate post-search cleanup keeps the selfie only for the configured window:
	// deletion is performed by SweepRetention based on events.selfie_retention_minutes.
	// This job exists to guarantee a scheduled check per search (job type kept for observability).
	_ = searchID; return s.SweepRetention(ctx)
}
