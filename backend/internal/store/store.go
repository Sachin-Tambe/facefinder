package store

import (
	"context"; "errors"
	"github.com/jackc/pgx/v5"; "github.com/jackc/pgx/v5/pgxpool"
	"facefinder/internal/model"
)

type Store struct{ DB *pgxpool.Pool }
func New(db *pgxpool.Pool) *Store { return &Store{db} }
var ErrNotFound = errors.New("not found")

func (s *Store) CreateUser(ctx context.Context, email, hash, name string) (string, error) {
	var id string; err := s.DB.QueryRow(ctx, `INSERT INTO users(email,password_hash,name) VALUES($1,$2,$3) RETURNING id`, email, hash, name).Scan(&id); return id, err
}
// EnsureUser maps an external auth identity (Neon Auth / Better Auth) to a local users row
// on first sight, keyed by email. Local-auth users already exist and are returned as-is.
func (s *Store) EnsureUser(ctx context.Context, email, name string) (string, error) {
	if email == "" { return "", ErrNotFound }
	id, _, _, err := s.UserByEmail(ctx, email); if err == nil { return id, nil }
	return s.CreateUser(ctx, email, "", name) // empty password_hash: login happens in Neon Auth, never here
}
func (s *Store) UserByEmail(ctx context.Context, email string) (id, hash, name string, err error) {
	err = s.DB.QueryRow(ctx, `SELECT id,password_hash,name FROM users WHERE email=$1`, email).Scan(&id, &hash, &name)
	if err == pgx.ErrNoRows { err = ErrNotFound }; return
}
func (s *Store) CreateOrg(ctx context.Context, userID, name, slug string) (model.Org, error) {
	var o model.Org
	err := s.DB.QueryRow(ctx, `INSERT INTO organizations(name,slug) VALUES($1,$2) RETURNING id,name,slug,plan,created_at`, name, slug).Scan(&o.ID,&o.Name,&o.Slug,&o.Plan,&o.CreatedAt)
	if err != nil { return o, err }
	_, err = s.DB.Exec(ctx, `INSERT INTO organization_members(organization_id,user_id,role) VALUES($1,$2,'organization_owner')`, o.ID, userID)
	o.Role = "organization_owner"; return o, err
}
func (s *Store) OrgsForUser(ctx context.Context, userID string) ([]model.Org, error) {
	rows, e := s.DB.Query(ctx, `SELECT o.id,o.name,o.slug,o.plan,o.created_at,m.role FROM organizations o JOIN organization_members m ON m.organization_id=o.id WHERE m.user_id=$1 ORDER BY o.created_at`, userID)
	if e != nil { return nil, e }; defer rows.Close()
	var out []model.Org
	for rows.Next() { var o model.Org; if e:=rows.Scan(&o.ID,&o.Name,&o.Slug,&o.Plan,&o.CreatedAt,&o.Role); e!=nil {return nil,e}; out=append(out,o) }
	return out, rows.Err()
}
// Role returns "" when the user is not a member — the org-isolation gate for every query below.
func (s *Store) Role(ctx context.Context, orgID, userID string) (string, error) {
	var r string; err := s.DB.QueryRow(ctx, `SELECT role FROM organization_members WHERE organization_id=$1 AND user_id=$2`, orgID, userID).Scan(&r)
	if err == pgx.ErrNoRows { return "", ErrNotFound }; return r, err
}
func (s *Store) Audit(ctx context.Context, orgID, eventID, userID, action string, details any) {
	_, _ = s.DB.Exec(ctx, `INSERT INTO audit_logs(organization_id,event_id,user_id,actor,action,details) VALUES(NULLIF($1,'')::uuid,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,'user',$4,$5)`, orgID, eventID, userID, action, details)
}

const eventCols = `id,organization_id,name,description,event_code,rekognition_collection_id,status,starts_at,ends_at,expires_at,face_data_expires_at,photos_expire_at,gallery_ttl_hours,whatsapp_session_ttl_hours,selfie_retention_minutes,auto_delete_enabled,downloads_enabled,created_at,updated_at`
func scanEvent(r pgx.Row) (model.Event, error) {
	var e model.Event
	err := r.Scan(&e.ID,&e.OrganizationID,&e.Name,&e.Description,&e.EventCode,&e.RekognitionCollectionID,&e.Status,&e.StartsAt,&e.EndsAt,&e.ExpiresAt,&e.FaceDataExpiresAt,&e.PhotosExpireAt,&e.GalleryTTLHours,&e.WhatsAppSessionTTLHours,&e.SelfieRetentionMinutes,&e.AutoDeleteEnabled,&e.DownloadsEnabled,&e.CreatedAt,&e.UpdatedAt)
	if err == pgx.ErrNoRows { err = ErrNotFound }; return e, err
}
func (s *Store) CreateEvent(ctx context.Context, e model.Event, tokenHash string) (model.Event, error) {
	return scanEvent(s.DB.QueryRow(ctx, `INSERT INTO events(organization_id,name,description,event_code,event_token_hash,rekognition_collection_id,starts_at,ends_at,face_data_expires_at,photos_expire_at,gallery_ttl_hours,whatsapp_session_ttl_hours,selfie_retention_minutes,auto_delete_enabled,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING `+eventCols,
		e.OrganizationID,e.Name,e.Description,e.EventCode,tokenHash,e.RekognitionCollectionID,e.StartsAt,e.EndsAt,e.FaceDataExpiresAt,e.PhotosExpireAt,e.GalleryTTLHours,e.WhatsAppSessionTTLHours,e.SelfieRetentionMinutes,e.AutoDeleteEnabled, nil))
}
func (s *Store) EventByID(ctx context.Context, orgID, id string) (model.Event, error) { return scanEvent(s.DB.QueryRow(ctx, `SELECT `+eventCols+` FROM events WHERE id=$1 AND organization_id=$2`, id, orgID)) }
func (s *Store) EventByIDAnyOrg(ctx context.Context, id string) (model.Event, error) { return scanEvent(s.DB.QueryRow(ctx, `SELECT `+eventCols+` FROM events WHERE id=$1`, id)) }
func (s *Store) EventByTokenHash(ctx context.Context, hash string) (model.Event, error) { return scanEvent(s.DB.QueryRow(ctx, `SELECT `+eventCols+` FROM events WHERE event_token_hash=$1`, hash)) }
func (s *Store) ListEvents(ctx context.Context, orgID string) ([]model.Event, error) {
	rows, e := s.DB.Query(ctx, `SELECT `+eventCols+` FROM events WHERE organization_id=$1 ORDER BY created_at DESC`, orgID)
	if e != nil { return nil, e }; defer rows.Close()
	var out []model.Event
	for rows.Next() { ev, err := scanEvent(rows); if err!=nil {return nil,err}; out=append(out,ev) }
	return out, rows.Err()
}
func (s *Store) UpdateEventToken(ctx context.Context, orgID, id, hash string) error { _, e := s.DB.Exec(ctx, `UPDATE events SET event_token_hash=$3,updated_at=now() WHERE id=$1 AND organization_id=$2`, id, orgID, hash); return e }
func (s *Store) PatchEvent(ctx context.Context, orgID, id, name, desc, status string) (model.Event, error) {
	return scanEvent(s.DB.QueryRow(ctx, `UPDATE events SET name=COALESCE(NULLIF($3,''),name),description=$4,status=COALESCE(NULLIF($5,''),status),updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING `+eventCols, id, orgID, name, desc, status))
}
func (s *Store) DeleteEventRow(ctx context.Context, orgID, id string) error { _, e := s.DB.Exec(ctx, `DELETE FROM events WHERE id=$1 AND organization_id=$2`, id, orgID); return e }
func (s *Store) UpdateRetention(ctx context.Context, orgID, id string, galleryH, sessionH, selfieMin int, autoDel bool) (model.Event, error) {
	return scanEvent(s.DB.QueryRow(ctx, `UPDATE events SET gallery_ttl_hours=$3,whatsapp_session_ttl_hours=$4,selfie_retention_minutes=$5,auto_delete_enabled=$6,updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING `+eventCols, id, orgID, galleryH, sessionH, selfieMin, autoDel))
}

// Photos
const photoCols = `id,event_id,organization_id,r2_original_key,r2_web_key,r2_thumb_key,r2_analysis_key,file_name,mime_type,file_size,width,height,processing_status,face_count,error,uploaded_at,processed_at`
func scanPhoto(r pgx.Row) (model.Photo, error) {
	var p model.Photo
	err := r.Scan(&p.ID,&p.EventID,&p.OrgID,&p.OriginalKey,&p.WebKey,&p.ThumbKey,&p.AnalysisKey,&p.FileName,&p.Mime,&p.FileSize,&p.Width,&p.Height,&p.Status,&p.FaceCount,&p.Error,&p.UploadedAt,&p.ProcessedAt)
	if err == pgx.ErrNoRows { err = ErrNotFound }; return p, err
}
func (s *Store) CreatePhoto(ctx context.Context, p model.Photo) (model.Photo, error) {
	return scanPhoto(s.DB.QueryRow(ctx, `INSERT INTO photos(event_id,organization_id,r2_original_key,file_name,mime_type,file_size,processing_status) VALUES($1,$2,$3,$4,$5,$6,'queued') RETURNING `+photoCols, p.EventID,p.OrgID,p.OriginalKey,p.FileName,p.Mime,p.FileSize))
}
func (s *Store) PhotoByID(ctx context.Context, id string) (model.Photo, error) { return scanPhoto(s.DB.QueryRow(ctx, `SELECT `+photoCols+` FROM photos WHERE id=$1`, id)) }
func (s *Store) ListPhotos(ctx context.Context, orgID, eventID string, limit, offset int) ([]model.Photo, int, error) {
	var total int; _ = s.DB.QueryRow(ctx, `SELECT count(*) FROM photos WHERE event_id=$1 AND organization_id=$2`, eventID, orgID).Scan(&total)
	rows, e := s.DB.Query(ctx, `SELECT `+photoCols+` FROM photos WHERE event_id=$1 AND organization_id=$2 ORDER BY uploaded_at DESC LIMIT $3 OFFSET $4`, eventID, orgID, limit, offset)
	if e != nil { return nil, 0, e }; defer rows.Close()
	var out []model.Photo
	for rows.Next() { p, err := scanPhoto(rows); if err!=nil {return nil,0,err}; out=append(out,p) }
	return out, total, rows.Err()
}
func (s *Store) SetPhotoStatus(ctx context.Context, id, status string, faceCount int, errText string) error {
	_, e := s.DB.Exec(ctx, `UPDATE photos SET processing_status=$2,face_count=$3,error=NULLIF($4,''),processed_at=CASE WHEN $2 IN ('ready','no_face','failed') THEN now() ELSE processed_at END,updated_at=now() WHERE id=$1`, id, status, faceCount, errText); return e
}
func (s *Store) SetPhotoVariants(ctx context.Context, id, web, thumb, analysis string, w, h int) error {
	_, e := s.DB.Exec(ctx, `UPDATE photos SET r2_web_key=$2,r2_thumb_key=$3,r2_analysis_key=$4,width=$5,height=$6,updated_at=now() WHERE id=$1`, id, web, thumb, analysis, w, h); return e
}
func (s *Store) InsertFace(ctx context.Context, eventID, photoID string, f model.Face) error {
	_, e := s.DB.Exec(ctx, `INSERT INTO photo_faces(event_id,photo_id,aws_face_id,aws_image_id,bounding_box_json,detection_confidence) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(event_id,aws_face_id) DO NOTHING`, eventID, photoID, f.AWSFaceID, f.AWSImageID, f.BBoxJSON, f.Confidence); return e
}
func (s *Store) EventStats(ctx context.Context, orgID, eventID string) (model.Stats, error) {
	var st model.Stats
	err := s.DB.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE processing_status='ready'),count(*) FILTER(WHERE processing_status='failed'),count(*) FILTER(WHERE processing_status='no_face'),count(*) FILTER(WHERE processing_status IN ('queued','processing','uploaded')) FROM photos WHERE event_id=$1 AND organization_id=$2`, eventID, orgID).Scan(&st.TotalPhotos,&st.Ready,&st.Failed,&st.NoFace,&st.Processing)
	if err != nil { return st, err }
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM photo_faces WHERE event_id=$1`, eventID).Scan(&st.IndexedFaces)
	_ = s.DB.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE status='completed'),count(*) FILTER(WHERE status='no_matches') FROM search_requests WHERE event_id=$1`, eventID).Scan(&st.Searches,&st.SuccessfulSearches,&st.ZeroSearches)
	return st, nil
}
func (s *Store) OrgStats(ctx context.Context, orgID string) (map[string]int, error) {
	out := map[string]int{}
	var totalEvents, activeEvents, totalPhotos, indexedFaces, guestSearches int
	_ = s.DB.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE status='active') FROM events WHERE organization_id=$1`, orgID).Scan(&totalEvents, &activeEvents)
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM photos WHERE organization_id=$1`, orgID).Scan(&totalPhotos)
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM photo_faces pf JOIN events e ON e.id=pf.event_id WHERE e.organization_id=$1`, orgID).Scan(&indexedFaces)
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM search_requests sr JOIN events e ON e.id=sr.event_id WHERE e.organization_id=$1`, orgID).Scan(&guestSearches)
	out["total_events"], out["active_events"], out["total_photos"], out["indexed_faces"], out["guest_searches"] = totalEvents, activeEvents, totalPhotos, indexedFaces, guestSearches
	return out, nil
}

// WhatsApp sessions / searches / gallery
func (s *Store) UpsertSession(ctx context.Context, phone, gateway, eventID string, ttlHours int) error {
	_, e := s.DB.Exec(ctx, `INSERT INTO whatsapp_sessions(phone_e164,gateway_number,event_id,expires_at) VALUES($1,$2,$3,now()+make_interval(hours=>$4)) ON CONFLICT(phone_e164,gateway_number) DO UPDATE SET event_id=EXCLUDED.event_id,last_activity_at=now(),expires_at=EXCLUDED.expires_at,status='active'`, phone, gateway, eventID, ttlHours); return e
}
func (s *Store) ActiveSession(ctx context.Context, phone, gateway string) (model.Event, error) {
	// gateway may be unknown on some webhooks: fall back to phone-only lookup
	if gateway != "" {
		ev, err := scanEvent(s.DB.QueryRow(ctx, `SELECT `+prefixedEventCols("e")+` FROM whatsapp_sessions s JOIN events e ON e.id=s.event_id WHERE s.phone_e164=$1 AND s.gateway_number=$2 AND s.status='active' AND s.expires_at>now()`, phone, gateway))
		if err == nil { return ev, nil }
	}
	return scanEvent(s.DB.QueryRow(ctx, `SELECT `+prefixedEventCols("e")+` FROM whatsapp_sessions s JOIN events e ON e.id=s.event_id WHERE s.phone_e164=$1 AND s.status='active' AND s.expires_at>now() ORDER BY s.last_activity_at DESC LIMIT 1`, phone))
}
func prefixedEventCols(a string) string {
	cols := []string{"id","organization_id","name","description","event_code","rekognition_collection_id","status","starts_at","ends_at","expires_at","face_data_expires_at","photos_expire_at","gallery_ttl_hours","whatsapp_session_ttl_hours","selfie_retention_minutes","auto_delete_enabled","downloads_enabled","created_at","updated_at"}
	out := ""; for i,c := range cols { if i>0 {out+=","}; out += a+"."+c }; return out
}
func (s *Store) CreateSearch(ctx context.Context, eventID, phone, selfieKey string) (string, error) {
	var id string; err := s.DB.QueryRow(ctx, `INSERT INTO search_requests(event_id,phone_e164,selfie_r2_key,status) VALUES($1,$2,$3,'processing') RETURNING id`, eventID, phone, selfieKey).Scan(&id); return id, err
}
func (s *Store) FinishSearch(ctx context.Context, id, status string, faceCount int, errText string) error {
	_, e := s.DB.Exec(ctx, `UPDATE search_requests SET status=$2,detected_face_count=$3,error=NULLIF($4,''),completed_at=now() WHERE id=$1`, id, status, faceCount, errText); return e
}
func (s *Store) SearchByID(ctx context.Context, id string) (model.Search, error) {
	var sr model.Search
	err := s.DB.QueryRow(ctx, `SELECT id,event_id,phone_e164,status,detected_face_count,created_at,completed_at FROM search_requests WHERE id=$1`, id).Scan(&sr.ID,&sr.EventID,&sr.Phone,&sr.Status,&sr.FaceCount,&sr.CreatedAt,&sr.CompletedAt)
	if err == pgx.ErrNoRows { err = ErrNotFound }; return sr, err
}
func (s *Store) InsertResult(ctx context.Context, searchID string, r model.SearchResult) error {
	_, e := s.DB.Exec(ctx, `INSERT INTO search_results(search_id,photo_id,best_face_id,similarity,result_type,rank) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(search_id,photo_id) DO NOTHING`, searchID, r.PhotoID, r.BestFaceID, r.Similarity, r.Type, r.Rank); return e
}
func (s *Store) ResultsForSearch(ctx context.Context, searchID string) ([]model.SearchResult, error) {
	rows, e := s.DB.Query(ctx, `SELECT photo_id,best_face_id,similarity,result_type,rank FROM search_results WHERE search_id=$1 ORDER BY rank`, searchID)
	if e != nil { return nil, e }; defer rows.Close()
	var out []model.SearchResult
	for rows.Next() { var r model.SearchResult; if e:=rows.Scan(&r.PhotoID,&r.BestFaceID,&r.Similarity,&r.Type,&r.Rank); e!=nil {return nil,e}; out=append(out,r) }
	return out, rows.Err()
}
// PhotosForFaces maps AWS FaceId -> photo keeping best similarity (dedup §20, sort §20).
func (s *Store) PhotosForFaceIDs(ctx context.Context, eventID string, faceIDs []string) (map[string]string, error) {
	rows, e := s.DB.Query(ctx, `SELECT aws_face_id,photo_id::text FROM photo_faces WHERE event_id=$1 AND aws_face_id=ANY($2)`, eventID, faceIDs)
	if e != nil { return nil, e }; defer rows.Close()
	out := map[string]string{}
	for rows.Next() { var f,p string; if e:=rows.Scan(&f,&p); e!=nil {return nil,e}; out[f]=p }
	return out, rows.Err()
}
func (s *Store) CreateGallery(ctx context.Context, searchID, eventID, phone, tokenHash string, ttlHours int) error {
	_, e := s.DB.Exec(ctx, `INSERT INTO gallery_tokens(search_id,event_id,phone_e164,token_hash,expires_at) VALUES($1,$2,$3,$4,now()+make_interval(hours=>$5))`, searchID, eventID, phone, tokenHash, ttlHours); return e
}
func (s *Store) GalleryByHash(ctx context.Context, hash string) (searchID, eventID string, err error) {
	err = s.DB.QueryRow(ctx, `SELECT search_id::text,event_id::text FROM gallery_tokens WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now()`, hash).Scan(&searchID,&eventID)
	if err == pgx.ErrNoRows { err = ErrNotFound }; return
}
func (s *Store) TouchGallery(ctx context.Context, hash string) { _, _ = s.DB.Exec(ctx, `UPDATE gallery_tokens SET last_accessed_at=now() WHERE token_hash=$1`, hash) }

// Retention sweeps
func (s *Store) ExpiredFaceEvents(ctx context.Context) ([]model.Event, error) { return s.eventsWhere(ctx, `face_data_expires_at<=now() AND face_collection_deleted_at IS NULL AND auto_delete_enabled`) }
func (s *Store) ExpiredPhotoEvents(ctx context.Context) ([]model.Event, error) { return s.eventsWhere(ctx, `photos_expire_at<=now() AND photos_deleted_at IS NULL AND auto_delete_enabled`) }
func (s *Store) eventsWhere(ctx context.Context, where string) ([]model.Event, error) {
	rows, e := s.DB.Query(ctx, `SELECT `+eventCols+` FROM events WHERE `+where+` LIMIT 100`)
	if e != nil { return nil, e }; defer rows.Close()
	var out []model.Event
	for rows.Next() { ev, err := scanEvent(rows); if err!=nil {return nil,err}; out=append(out,ev) }
	return out, rows.Err()
}
func (s *Store) MarkFaceCollectionDeleted(ctx context.Context, id string) error { _, e := s.DB.Exec(ctx, `UPDATE events SET face_collection_deleted_at=now(),status='expired',updated_at=now() WHERE id=$1`, id); if e==nil { _, _ = s.DB.Exec(ctx, `DELETE FROM photo_faces WHERE event_id=$1`, id) }; return e }
func (s *Store) MarkPhotosDeleted(ctx context.Context, id string) error { _, e := s.DB.Exec(ctx, `UPDATE events SET photos_deleted_at=now(),updated_at=now() WHERE id=$1`, id); return e }
func (s *Store) ExpiredSelfies(ctx context.Context) ([]string, error) {
	rows, e := s.DB.Query(ctx, `SELECT selfie_r2_key FROM search_requests WHERE selfie_r2_key IS NOT NULL AND deleted_at IS NULL AND created_at < now() - make_interval(mins => (SELECT selfie_retention_minutes FROM events WHERE id=event_id)) LIMIT 500`)
	if e != nil { return nil, e }; defer rows.Close()
	var out []string
	for rows.Next() { var k string; if e:=rows.Scan(&k); e!=nil {return nil,e}; out=append(out,k) }
	return out, rows.Err()
}
func (s *Store) MarkSelfiesDeleted(ctx context.Context, keys []string) error { _, e := s.DB.Exec(ctx, `UPDATE search_requests SET selfie_r2_key=NULL,deleted_at=now() WHERE selfie_r2_key=ANY($1)`, keys); return e }
func (s *Store) SelfieKey(ctx context.Context, searchID string) (string, error) { var k *string; e := s.DB.QueryRow(ctx, `SELECT selfie_r2_key FROM search_requests WHERE id=$1`, searchID).Scan(&k); if k==nil {return "",e}; return *k, e }
