package middleware

import (
	"context"; "crypto/subtle"; "net/http"; "strings"; "sync"; "time"
	"github.com/golang-jwt/jwt/v5"
)
type ctxKey string
const UserKey ctxKey="user"; const OrgKey ctxKey="org"; const ReqKey ctxKey="req"
func UserID(r *http.Request) string { v,_:=r.Context().Value(UserKey).(string); return v }
func UserEmail(r *http.Request) string { v,_:=r.Context().Value(ctxKey("email")).(string); return v }
func UserName(r *http.Request) string { v,_:=r.Context().Value(ctxKey("name")).(string); return v }
func OrgID(r *http.Request) string { v,_:=r.Context().Value(OrgKey).(string); return v }

func RequestID(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
	id:=r.Header.Get("X-Request-Id"); if id=="" {id=time.Now().Format("20060102150405")+r.RemoteAddr}
	w.Header().Set("X-Request-Id",id); next.ServeHTTP(w,r.WithContext(context.WithValue(r.Context(),ReqKey,id)))
})}
func CORS(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
	w.Header().Set("Access-Control-Allow-Origin","*"); w.Header().Set("Access-Control-Allow-Headers","Authorization,Content-Type,X-Org-Id,X-WhatsApp-Secret"); w.Header().Set("Access-Control-Allow-Methods","GET,POST,PATCH,DELETE,PUT,OPTIONS")
	if r.Method=="OPTIONS" {w.WriteHeader(204); return}; next.ServeHTTP(w,r)
})}
// Auth validates the Go-issued JWT (Better Auth / Neon Auth can mint the same claims — swap verifier here only).
func Auth(secret string) func(http.Handler) http.Handler { return func(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
	h:=strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer "); if h=="" {http.Error(w,`{"error":"unauthorized"}`,401); return}
	tok,err:=jwt.Parse(h, func(t *jwt.Token)(any,error){return []byte(secret),nil}); if err!=nil||!tok.Valid {http.Error(w,`{"error":"unauthorized"}`,401); return}
	claims,_:=tok.Claims.(jwt.MapClaims); sub,_:=claims["sub"].(string); if sub=="" {http.Error(w,`{"error":"unauthorized"}`,401); return}
	ctx:=context.WithValue(r.Context(),UserKey,sub)
	if e,_:=claims["email"].(string); e!="" { ctx=context.WithValue(ctx,ctxKey("email"),e) }
	if n,_:=claims["name"].(string); n!="" { ctx=context.WithValue(ctx,ctxKey("name"),n) }
	if org:=r.Header.Get("X-Org-Id"); org!="" {ctx=context.WithValue(ctx,OrgKey,org)}
	next.ServeHTTP(w,r.WithContext(ctx))
})}}
// Simple per-IP token-bucket-ish fixed window limiter (use Redis limiter at scale).
type Limiter struct{ mu sync.Mutex; hits map[string][]time.Time; Max int; Window time.Duration }
func NewLimiter(max int, window time.Duration) *Limiter { return &Limiter{hits:map[string][]time.Time{},Max:max,Window:window} }
func (l *Limiter) Middleware(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
	l.mu.Lock(); now:=time.Now(); var kept []time.Time; for _,t:=range l.hits[r.RemoteAddr] {if now.Sub(t)<l.Window {kept=append(kept,t)}}
	if len(kept)>=l.Max {l.mu.Unlock(); http.Error(w,`{"error":"rate limited"}`,429); return}
	l.hits[r.RemoteAddr]=append(kept,now); l.mu.Unlock(); next.ServeHTTP(w,r)
})}
func SecretEqual(a,b string) bool { return subtle.ConstantTimeCompare([]byte(a),[]byte(b))==1 }
