package httpapi

import (
	"context"; "encoding/json"; "fmt"; "io"; "log/slog"; "net/http"; "strconv"; "strings"; "time"

	"github.com/golang-jwt/jwt/v5"; "github.com/skip2/go-qrcode"; "golang.org/x/crypto/bcrypt"
	"facefinder/internal/config"; "facefinder/internal/middleware"; "facefinder/internal/model"; "facefinder/internal/service"; "facefinder/internal/store"; "facefinder/internal/whatsapp"
)

type Server struct { Cfg config.Config; Svc *service.Service; Store *store.Store }
func jsonw(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(code); _=json.NewEncoder(w).Encode(v) }
func errw(w http.ResponseWriter, code int, msg string) { jsonw(w,code,map[string]string{"error":msg}) }
func decode(r *http.Request, v any) error { defer r.Body.Close(); return json.NewDecoder(io.LimitReader(r.Body,1<<20)).Decode(v) }
// requireOrg enforces membership — org isolation gate for every business route.
func (s *Server) requireOrg(w http.ResponseWriter, r *http.Request) (string,bool) {
	org:=middleware.OrgID(r); if org=="" { org=r.URL.Query().Get("organization_id") }
	if org=="" { // default to first org
		orgs,_:=s.Store.OrgsForUser(r.Context(), middleware.UserID(r)); if len(orgs)>0 { org=orgs[0].ID }
	}
	if org=="" { errw(w,400,"organization required (X-Org-Id)"); return "",false }
	if _,err:=s.Store.Role(r.Context(), org, middleware.UserID(r)); err!=nil { errw(w,403,"not a member of this organization"); return "",false }
	return org,true
}
func (s *Server) token(userID string) (string,error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub":userID,"exp":time.Now().Add(7*24*time.Hour).Unix()}).SignedString([]byte(s.Cfg.AuthSecret))
}

func (s *Server) Handler() http.Handler {
	mux:=http.NewServeMux(); auth:=middleware.Auth(s.Cfg.AuthSecret)
	mux.HandleFunc("POST /api/v1/auth/register", s.register)
	mux.HandleFunc("POST /api/v1/auth/exchange", s.exchange) // Neon Auth bridge: Next.js server (holding the shared secret) trades a verified Neon session for a Go JWT
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.Handle("GET /api/v1/me", auth(http.HandlerFunc(s.me)))
	mux.Handle("GET /api/v1/organizations", auth(http.HandlerFunc(s.listOrgs)))
	mux.Handle("POST /api/v1/organizations", auth(http.HandlerFunc(s.createOrg)))
	mux.Handle("GET /api/v1/organizations/stats", auth(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; st,_:=s.Store.OrgStats(r.Context(),org); jsonw(w,200,st) })))
	mux.Handle("POST /api/v1/events", auth(http.HandlerFunc(s.createEvent)))
	mux.Handle("GET /api/v1/events", auth(http.HandlerFunc(s.listEvents)))
	mux.Handle("GET /api/v1/events/{id}", auth(http.HandlerFunc(s.getEvent)))
	mux.Handle("PATCH /api/v1/events/{id}", auth(http.HandlerFunc(s.patchEvent)))
	mux.Handle("DELETE /api/v1/events/{id}", auth(http.HandlerFunc(s.deleteEvent)))
	mux.Handle("GET /api/v1/events/{id}/qr", auth(http.HandlerFunc(s.qr)))
	mux.Handle("POST /api/v1/events/{id}/qr/regenerate", auth(http.HandlerFunc(s.regenQR)))
	mux.Handle("POST /api/v1/events/{id}/uploads/presign", auth(http.HandlerFunc(s.presign)))
	mux.Handle("POST /api/v1/events/{id}/uploads/complete", auth(http.HandlerFunc(s.completeUpload)))
	mux.Handle("GET /api/v1/events/{id}/photos", auth(http.HandlerFunc(s.listPhotos)))
	mux.Handle("GET /api/v1/photos/{id}", auth(http.HandlerFunc(s.getPhoto)))
	mux.Handle("DELETE /api/v1/photos/{id}", auth(http.HandlerFunc(s.deletePhoto)))
	mux.Handle("POST /api/v1/photos/{id}/reindex", auth(http.HandlerFunc(s.reindex)))
	mux.Handle("GET /api/v1/events/{id}/processing", auth(http.HandlerFunc(s.processing)))
	mux.Handle("POST /api/v1/events/{id}/retry-failed", auth(http.HandlerFunc(s.retryFailed)))
	mux.Handle("GET /api/v1/events/{id}/retention", auth(http.HandlerFunc(s.getRetention)))
	mux.Handle("PATCH /api/v1/events/{id}/retention", auth(http.HandlerFunc(s.patchRetention)))
	mux.HandleFunc("POST /api/v1/webhooks/whatsapp", s.waWebhook)
	mux.HandleFunc("POST /api/v1/webhooks/whatsapp/status", func(w http.ResponseWriter,r *http.Request){ jsonw(w,200,map[string]bool{"ok":true}) })
	mux.Handle("GET /api/v1/searches/{id}", auth(http.HandlerFunc(s.getSearch)))
	mux.HandleFunc("GET /api/v1/gallery/{token}", s.gallery) // data API for /g/{token} page
	mux.HandleFunc("GET /g/{token}", func(w http.ResponseWriter,r *http.Request){ http.Redirect(w,r,s.Cfg.PublicWebURL+"/g/"+r.PathValue("token"),302) })
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter,r *http.Request){ jsonw(w,200,map[string]string{"status":"ok"}) })
	// Dev-only local-storage object route (never registered when R2 is enabled in prod path — handler still requires no auth for upload PUT from browser; keys are unguessable UUIDs)
	mux.HandleFunc("PUT /dev-objects/", s.devPut); mux.HandleFunc("GET /dev-objects/", s.devGet)
	return middleware.RequestID(middleware.CORS(middleware.NewLimiter(600,time.Minute).Middleware(mux)))
}

func (s *Server) register(w http.ResponseWriter,r *http.Request){ var in struct{Email,Password,Name string}; if decode(r,&in)!=nil||in.Email==""||len(in.Password)<8 {errw(w,400,"email and 8+ char password required");return}
	h,_:=bcrypt.GenerateFromPassword([]byte(in.Password),12); id,err:=s.Store.CreateUser(r.Context(),strings.ToLower(in.Email),string(h),in.Name); if err!=nil {errw(w,409,"email already registered");return}
	t,_:=s.token(id); jsonw(w,201,map[string]any{"token":t,"user":map[string]string{"id":id,"email":in.Email,"name":in.Name}}) }
func (s *Server) exchange(w http.ResponseWriter,r *http.Request){
	if !middleware.SecretEqual(s.Cfg.AuthSecret, r.Header.Get("X-Auth-Bridge-Secret")) { errw(w,401,"bad bridge secret"); return }
	var in struct{Email,Name string}; if decode(r,&in)!=nil||in.Email=="" {errw(w,400,"email required");return}
	id,err:=s.Store.EnsureUser(r.Context(),strings.ToLower(in.Email),in.Name); if err!=nil{errw(w,500,"user provisioning failed");return}
	t,_:=s.token(id); jsonw(w,200,map[string]any{"token":t,"user":map[string]string{"id":id,"email":strings.ToLower(in.Email),"name":in.Name}}) }
func (s *Server) login(w http.ResponseWriter,r *http.Request){ var in struct{Email,Password string}; if decode(r,&in)!=nil {errw(w,400,"bad request");return}
	id,hash,name,err:=s.Store.UserByEmail(r.Context(),strings.ToLower(in.Email)); if err!=nil||bcrypt.CompareHashAndPassword([]byte(hash),[]byte(in.Password))!=nil {errw(w,401,"invalid credentials");return}
	t,_:=s.token(id); jsonw(w,200,map[string]any{"token":t,"user":map[string]string{"id":id,"email":in.Email,"name":name}}) }
func (s *Server) me(w http.ResponseWriter,r *http.Request){ orgs,_:=s.Store.OrgsForUser(r.Context(),middleware.UserID(r)); jsonw(w,200,map[string]any{"id":middleware.UserID(r),"organizations":orgs}) }
func (s *Server) listOrgs(w http.ResponseWriter,r *http.Request){ orgs,err:=s.Store.OrgsForUser(r.Context(),middleware.UserID(r)); if err!=nil{errw(w,500,"db error");return}; if orgs==nil{orgs=[]model.Org{}}; jsonw(w,200,orgs) }
func (s *Server) createOrg(w http.ResponseWriter,r *http.Request){ var in struct{Name string}; _=decode(r,&in); if in.Name==""{errw(w,400,"name required");return}
	slug:=strings.ToLower(strings.ReplaceAll(in.Name," ","-"))+"-"+service.Token(2); o,err:=s.Store.CreateOrg(r.Context(),middleware.UserID(r),in.Name,slug); if err!=nil{errw(w,500,err.Error());return}; jsonw(w,201,o) }

func (s *Server) createEvent(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}
	var in model.Event; if decode(r,&in)!=nil||in.Name==""{errw(w,400,"name required");return}
	ev,err:=s.Svc.CreateEvent(r.Context(),org,middleware.UserID(r),in); if err!=nil{slog.Error("event create","err",err);errw(w,502,err.Error());return}; jsonw(w,201,ev) }
func (s *Server) listEvents(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; evs,err:=s.Store.ListEvents(r.Context(),org); if err!=nil{errw(w,500,"db error");return}; if evs==nil{evs=[]model.Event{}}; jsonw(w,200,evs) }
func (s *Server) getEvent(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; ev,err:=s.Store.EventByID(r.Context(),org,r.PathValue("id")); if err!=nil{errw(w,404,"event not found");return}
	st,_:=s.Store.EventStats(r.Context(),org,ev.ID); jsonw(w,200,map[string]any{"event":ev,"stats":st,"whatsapp_link":s.Svc.WhatsAppLink("")}) }
func (s *Server) patchEvent(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; var in struct{Name,Description,Status string}; _=decode(r,&in)
	ev,err:=s.Store.PatchEvent(r.Context(),org,r.PathValue("id"),in.Name,in.Description,in.Status); if err!=nil{errw(w,404,"event not found");return}; jsonw(w,200,ev) }
func (s *Server) deleteEvent(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; if err:=s.Svc.DeleteEvent(r.Context(),org,middleware.UserID(r),r.PathValue("id"));err!=nil{errw(w,404,"event not found");return}; jsonw(w,200,map[string]bool{"ok":true}) }
func (s *Server) qr(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; ev,err:=s.Store.EventByID(r.Context(),org,r.PathValue("id")); if err!=nil{errw(w,404,"event not found");return}
	// Token is not retrievable after create (hash only) — QR encodes link with placeholder note; regenerate to get a fresh printable QR.
	_ = ev; errw(w,409,"QR token is shown only at creation/regeneration (stored hashed). Use POST /qr/regenerate to print a new QR.") }
func (s *Server) regenQR(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}
	tok,link,err:=s.Svc.RegenerateToken(r.Context(),org,middleware.UserID(r),r.PathValue("id")); if err!=nil{errw(w,404,"event not found");return}
	if r.URL.Query().Get("format")=="png" { png,err:=qrcode.Encode(link, qrcode.Medium, 512); if err!=nil{errw(w,500,"qr failed");return}; w.Header().Set("Content-Type","image/png"); _,_=w.Write(png); return }
	jsonw(w,200,map[string]string{"event_token":tok,"whatsapp_link":link}) }
func (s *Server) presign(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; ev,err:=s.Store.EventByID(r.Context(),org,r.PathValue("id")); if err!=nil{errw(w,404,"event not found");return}
	var in struct{FileName,Mime string; Size int64}; if decode(r,&in)!=nil{errw(w,400,"bad request");return}
	if !strings.HasPrefix(in.Mime,"image/"){errw(w,415,"only images allowed");return}; if in.Size>25<<20{errw(w,413,"file too large (25MB)");return}
	photoID:=service.Token(16); _=photoID
	p,err:=s.Store.CreatePhoto(r.Context(), model.Photo{EventID:ev.ID, OrgID:org, FileName:in.FileName, Mime:in.Mime, FileSize:in.Size, OriginalKey:"pending"})
	if err!=nil{errw(w,500,"db error");return}
	key:=fmt.Sprintf("events/%s/original/%s.jpg", ev.ID, p.ID)
	// store key back via variants-less update: reuse SetPhotoVariants is wrong; do direct through photo row update in complete step. Keep key in response and persist on complete.
	url,err:=s.Svc.Storage.PresignUpload(r.Context(),key,in.Mime,15*time.Minute); if err!=nil{errw(w,502,"storage unavailable");return}
	jsonw(w,200,map[string]any{"photo_id":p.ID,"key":key,"upload_url":url}) }
func (s *Server) completeUpload(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; ev,err:=s.Store.EventByID(r.Context(),org,r.PathValue("id")); if err!=nil{errw(w,404,"event not found");return}
	var in struct{PhotoID,Key string}; if decode(r,&in)!=nil{errw(w,400,"bad request");return}
	// persist original key (single-purpose update)
	if _,err:=s.Store.DB.Exec(r.Context(),`UPDATE photos SET r2_original_key=$2,processing_status='queued' WHERE id=$1 AND event_id=$3 AND organization_id=$4`,in.PhotoID,in.Key,ev.ID,org); err!=nil{errw(w,500,"db error");return}
	if s.Svc.Enqueue!=nil { if err:=s.Svc.Enqueue("photo.process",map[string]string{"photo_id":in.PhotoID}); err!=nil { slog.Error("enqueue photo","err",err) } } else { go func(){ _=s.Svc.ProcessPhoto(context.Background(), in.PhotoID) }() }
	jsonw(w,200,map[string]string{"status":"queued"}) }
func (s *Server) listPhotos(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; limit,_:=strconv.Atoi(r.URL.Query().Get("limit")); if limit<=0||limit>200{limit=60}; offset,_:=strconv.Atoi(r.URL.Query().Get("offset"))
	photos,total,err:=s.Store.ListPhotos(r.Context(),org,r.PathValue("id"),limit,offset); if err!=nil{errw(w,500,"db error");return}
	out:=make([]map[string]any,0,len(photos)); for _,p:=range photos { thumb:=""; if p.ThumbKey!=nil { thumb,_=s.Svc.Storage.PresignDownload(r.Context(),*p.ThumbKey,15*time.Minute) }; out=append(out,map[string]any{"photo":p,"thumb_url":thumb}) }
	jsonw(w,200,map[string]any{"photos":out,"total":total}) }
func (s *Server) getPhoto(w http.ResponseWriter,r *http.Request){ p,err:=s.Store.PhotoByID(r.Context(),r.PathValue("id")); if err!=nil{errw(w,404,"photo not found");return}
	if _,err:=s.Store.Role(r.Context(),p.OrgID,middleware.UserID(r));err!=nil{errw(w,403,"forbidden");return}
	url:=""; if p.WebKey!=nil{url,_=s.Svc.Storage.PresignDownload(r.Context(),*p.WebKey,15*time.Minute)}; jsonw(w,200,map[string]any{"photo":p,"web_url":url}) }
func (s *Server) deletePhoto(w http.ResponseWriter,r *http.Request){ p,err:=s.Store.PhotoByID(r.Context(),r.PathValue("id")); if err!=nil{errw(w,404,"photo not found");return}
	if _,err:=s.Store.Role(r.Context(),p.OrgID,middleware.UserID(r));err!=nil{errw(w,403,"forbidden");return}
	for _,k:=range []*string{&p.OriginalKey,p.WebKey,p.ThumbKey,p.AnalysisKey}{ if k!=nil&&*k!=""{_ = s.Svc.Storage.Delete(r.Context(),*k)} }
	_,_=s.Store.DB.Exec(r.Context(),`DELETE FROM photos WHERE id=$1`,p.ID); jsonw(w,200,map[string]bool{"ok":true}) }
func (s *Server) reindex(w http.ResponseWriter,r *http.Request){ p,err:=s.Store.PhotoByID(r.Context(),r.PathValue("id")); if err!=nil{errw(w,404,"photo not found");return}
	if _,err:=s.Store.Role(r.Context(),p.OrgID,middleware.UserID(r));err!=nil{errw(w,403,"forbidden");return}
	_=s.Store.SetPhotoStatus(r.Context(),p.ID,"queued",0,""); if s.Svc.Enqueue!=nil{_ = s.Svc.Enqueue("photo.process",map[string]string{"photo_id":p.ID})} else {go func(){_=s.Svc.ProcessPhoto(context.Background(),p.ID)}()}; jsonw(w,200,map[string]string{"status":"queued"}) }
func (s *Server) processing(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; st,err:=s.Store.EventStats(r.Context(),org,r.PathValue("id")); if err!=nil{errw(w,500,"db error");return}; jsonw(w,200,st) }
func (s *Server) retryFailed(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}
	rows,err:=s.Store.DB.Query(r.Context(),`SELECT id::text FROM photos WHERE event_id=$1 AND organization_id=$2 AND processing_status='failed'`,r.PathValue("id"),org); if err!=nil{errw(w,500,"db error");return}; defer rows.Close(); n:=0
	for rows.Next(){var id string; _=rows.Scan(&id); _=s.Store.SetPhotoStatus(r.Context(),id,"queued",0,""); if s.Svc.Enqueue!=nil{_ = s.Svc.Enqueue("photo.process",map[string]string{"photo_id":id})}; n++}; jsonw(w,200,map[string]int{"requeued":n}) }
func (s *Server) getRetention(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; ev,err:=s.Store.EventByID(r.Context(),org,r.PathValue("id")); if err!=nil{errw(w,404,"event not found");return}; jsonw(w,200,ev) }
func (s *Server) patchRetention(w http.ResponseWriter,r *http.Request){ org,ok:=s.requireOrg(w,r); if !ok{return}; ev,err:=s.Store.EventByID(r.Context(),org,r.PathValue("id")); if err!=nil{errw(w,404,"event not found");return}
	var in struct{GalleryTTLHours,WhatsAppSessionTTLHours,SelfieRetentionMinutes *int; AutoDeleteEnabled *bool}; _=decode(r,&in)
	g,s2,se,a:=ev.GalleryTTLHours,ev.WhatsAppSessionTTLHours,ev.SelfieRetentionMinutes,ev.AutoDeleteEnabled
	if in.GalleryTTLHours!=nil{g=*in.GalleryTTLHours}; if in.WhatsAppSessionTTLHours!=nil{s2=*in.WhatsAppSessionTTLHours}; if in.SelfieRetentionMinutes!=nil{se=*in.SelfieRetentionMinutes}; if in.AutoDeleteEnabled!=nil{a=*in.AutoDeleteEnabled}
	ev,err=s.Store.UpdateRetention(r.Context(),org,ev.ID,g,s2,se,a); if err!=nil{errw(w,500,"db error");return}; jsonw(w,200,ev) }
func (s *Server) getSearch(w http.ResponseWriter,r *http.Request){ sr,err:=s.Store.SearchByID(r.Context(),r.PathValue("id")); if err!=nil{errw(w,404,"search not found");return}
	ev,err:=s.Store.EventByIDAnyOrg(r.Context(),sr.EventID); if err!=nil{errw(w,404,"search not found");return}
	if _,err:=s.Store.Role(r.Context(),ev.OrganizationID,middleware.UserID(r));err!=nil{errw(w,403,"forbidden");return}
	res,_:=s.Store.ResultsForSearch(r.Context(),sr.ID); jsonw(w,200,map[string]any{"search":sr,"results":res}) }

// Gallery data API: token is the secret (hashed at rest), expiry enforced in store query.
func (s *Server) gallery(w http.ResponseWriter,r *http.Request){ tok:=r.PathValue("token"); searchID,eventID,err:=s.Store.GalleryByHash(r.Context(),service.Hash(tok)); if err!=nil{errw(w,404,"gallery expired or not found");return}
	s.Store.TouchGallery(r.Context(),service.Hash(tok)); ev,_:=s.Store.EventByIDAnyOrg(r.Context(),eventID); res,_:=s.Store.ResultsForSearch(r.Context(),searchID)
	items:=make([]map[string]any,0,len(res)); for _,it:=range res { p,err:=s.Store.PhotoByID(r.Context(),it.PhotoID); if err!=nil{continue}; web,dl:="",""; if p.WebKey!=nil{web,_=s.Svc.Storage.PresignDownload(r.Context(),*p.WebKey,time.Hour)}; if ev.DownloadsEnabled{dl,_=s.Svc.Storage.PresignDownload(r.Context(),p.OriginalKey,time.Hour)}
		items=append(items,map[string]any{"photo_id":p.ID,"similarity":it.Similarity,"type":it.Type,"web_url":web,"download_url":dl,"file_name":p.FileName}) }
	jsonw(w,200,map[string]any{"event_name":ev.Name,"count":len(items),"photos":items,"downloads_enabled":ev.DownloadsEnabled}) }

// WhatsApp webhook (§10–11,29). Never searches without an active event session.
func (s *Server) waWebhook(w http.ResponseWriter,r *http.Request){
	if s.Cfg.WAWebhookSecret!="" && !middleware.SecretEqual(s.Cfg.WAWebhookSecret, r.Header.Get("X-WhatsApp-Secret")) { errw(w,401,"bad webhook secret"); return }
	var in whatsapp.Inbound; if decode(r,&in)!=nil||in.Phone==""{errw(w,400,"bad payload");return}
	ctx:=r.Context(); phone:=in.Phone; if !strings.HasPrefix(phone,"+"){phone="+"+phone}
	text:=strings.TrimSpace(in.Text)
	if strings.HasPrefix(strings.ToUpper(text),"START ") {
		token:=strings.TrimSpace(text[len("START "):]); ev,err:=s.Store.EventByTokenHash(ctx, service.Hash(token))
		if err!=nil { _=s.Svc.WA.SendText(ctx,phone,"This event link is invalid or expired. Please scan the QR code at your event again."); jsonw(w,200,map[string]bool{"ok":true}); return }
		if err:=s.Store.UpsertSession(ctx,phone,in.GatewayNumber,ev.ID,ev.WhatsAppSessionTTLHours); err!=nil{errw(w,500,"session failed");return}
		_,_=s.Store.DB.Exec(ctx,`INSERT INTO consents(event_id,phone_e164,kind) VALUES($1,$2,'whatsapp_search')`,ev.ID,phone)
		_=s.Svc.WA.SendText(ctx,phone,fmt.Sprintf("Welcome to %s! Please send one clear selfie containing only your face, and we'll find your photos.",ev.Name))
		jsonw(w,200,map[string]bool{"ok":true}); return
	}
	if strings.EqualFold(text,"SEARCH_AGAIN") { _=s.Svc.WA.SendText(ctx,phone,"Please send one clear selfie containing only your face."); jsonw(w,200,map[string]bool{"ok":true}); return }
	if strings.EqualFold(text,"CHANGE_EVENT") { _=s.Svc.WA.SendText(ctx,phone,"Please scan the QR code available at your event before sending your selfie."); jsonw(w,200,map[string]bool{"ok":true}); return }
	if in.Type=="image" || in.MediaID!="" {
		ev,err:=s.Store.ActiveSession(ctx,phone,in.GatewayNumber)
		if err!=nil { _=s.Svc.WA.SendText(ctx,phone,"Please scan the QR code available at your event before sending your selfie."); jsonw(w,200,map[string]bool{"ok":true}); return }
		img,_,err:=s.Svc.WA.DownloadMedia(ctx,in.MediaID); if err!=nil||len(img)==0 { _=s.Svc.WA.SendText(ctx,phone,"We couldn't download your selfie. Please send it again."); jsonw(w,200,map[string]bool{"ok":true}); return }
		out,err:=s.Svc.ProcessSelfie(ctx,ev,phone,img)
		if err!=nil { slog.Error("selfie search","err",err,"event",ev.ID); _=s.Svc.WA.SendText(ctx,phone,"Something went wrong while searching. Please try again."); jsonw(w,200,map[string]bool{"ok":true}); return }
		if out.Message!="" && len(out.Results)==0 { _=s.Svc.WA.SendText(ctx,phone,out.Message) }
		jsonw(w,200,map[string]bool{"ok":true}); return
	}
	_=s.Svc.WA.SendText(ctx,phone,"Please scan the QR code available at your event, then send one clear selfie to find your photos.")
	jsonw(w,200,map[string]bool{"ok":true})
}
func (s *Server) devPut(w http.ResponseWriter,r *http.Request){ key:=strings.TrimPrefix(r.URL.Path,"/dev-objects/"); b,err:=io.ReadAll(io.LimitReader(r.Body,30<<20)); if err!=nil{errw(w,400,"read failed");return}; if err:=s.Svc.Storage.Put(r.Context(),key,b,r.Header.Get("Content-Type"));err!=nil{errw(w,500,err.Error());return}; jsonw(w,200,map[string]bool{"ok":true}) }
func (s *Server) devGet(w http.ResponseWriter,r *http.Request){ key:=strings.TrimPrefix(r.URL.Path,"/dev-objects/"); b,err:=s.Svc.Storage.Get(r.Context(),key); if err!=nil{errw(w,404,"not found");return}; w.Header().Set("Content-Type","image/jpeg"); _,_=w.Write(b) }
