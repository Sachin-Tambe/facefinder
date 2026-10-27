package config

import ("net/url";"os";"strconv";"strings")

type Config struct {
	DatabaseURL, AuthSecret, Addr, PublicWebURL, RedisURL string
	AWSRegion, AWSKey, AWSSecret string
	R2Account, R2Key, R2Secret, R2Bucket, R2Endpoint string
	WAGatewayURL, WAGatewayKey, WAWebhookSecret, WABusinessNumber string
	SessionTTLHours, SelfieRetentionMin, FaceRetentionDays, PhotoRetentionDays, GalleryTTLHours int
	ThresholdStrong, ThresholdPossible, SearchThreshold float64
}
func env(k,d string) string { if v:=os.Getenv(k); v!="" {return v}; return d }
func envi(k string,d int) int { v,_:=strconv.Atoi(env(k,"")); if v==0 {return d}; return v }
func envf(k string,d float64) float64 { v,e:=strconv.ParseFloat(env(k,""),64); if e!=nil {return d}; return v }
// loadDotEnv fills unset env vars from the first .env found (cwd, then project root
// relative to backend/). Real environment variables always win.
func loadDotEnv(){
	for _, path := range []string{".env","../.env","../../.env"} {
		b, err := os.ReadFile(path); if err != nil { continue }
		for _, line := range strings.Split(string(b),"\n") {
			line = strings.TrimSpace(line); if line=="" || strings.HasPrefix(line,"#") { continue }
			k, v, ok := strings.Cut(line,"="); if !ok { continue }
			k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v),`"'`)
			if os.Getenv(k)=="" { _ = os.Setenv(k,v) }
		}
		return
	}
}
func Load() Config { loadDotEnv(); return Config{
	DatabaseURL: databaseURL(),
	AuthSecret: env("BETTER_AUTH_SECRET","dev-secret-change-me-32-bytes!!"), Addr: env("API_ADDR",":8080"),
	PublicWebURL: env("PUBLIC_WEB_URL","http://localhost:3000"), RedisURL: redisURL(),
	AWSRegion: env("AWS_REGION","ap-south-1"), AWSKey: os.Getenv("AWS_ACCESS_KEY_ID"), AWSSecret: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	R2Account: os.Getenv("R2_ACCOUNT_ID"), R2Key: os.Getenv("R2_ACCESS_KEY_ID"), R2Secret: os.Getenv("R2_SECRET_ACCESS_KEY"),
	R2Bucket: env("R2_BUCKET","facefinder-private"), R2Endpoint: os.Getenv("R2_ENDPOINT"),
	WAGatewayURL: os.Getenv("WHATSAPP_GATEWAY_BASE_URL"), WAGatewayKey: os.Getenv("WHATSAPP_GATEWAY_API_KEY"),
	WAWebhookSecret: env("WHATSAPP_WEBHOOK_SECRET",""), WABusinessNumber: env("WHATSAPP_BUSINESS_NUMBER",""),
	SessionTTLHours: envi("WHATSAPP_SESSION_TTL_HOURS",24), SelfieRetentionMin: envi("SELFIE_RETENTION_MINUTES",60),
	FaceRetentionDays: envi("DEFAULT_FACE_RETENTION_DAYS",30), PhotoRetentionDays: envi("DEFAULT_PHOTO_RETENTION_DAYS",180),
	GalleryTTLHours: envi("DEFAULT_GALLERY_TTL_HOURS",24),
	ThresholdStrong: envf("FACE_MATCH_THRESHOLD_STRONG",95), ThresholdPossible: envf("FACE_MATCH_THRESHOLD_POSSIBLE",85),
	SearchThreshold: envf("SEARCH_FACE_THRESHOLD",80),
}}
func (c Config) AWSEnabled() bool { return c.AWSKey!="" }
func (c Config) R2Enabled() bool { return c.R2Key!="" && (c.R2Endpoint!="" || c.R2Account!="") }
func (c Config) WAEnabled() bool { return c.WAGatewayURL!="" }

// redisURL: Upstash wins when set (UPSTASH_REDIS_URL, rediss://...), else REDIS_URL
// (offline local Docker Redis), else the local default.
func redisURL() string {
	if v := os.Getenv("UPSTASH_REDIS_URL"); v != "" { return v }
	return env("REDIS_URL", "redis://localhost:6379/0")
}
// RedisDisplay is the log-safe form: scheme+host+db, never credentials.
func (c Config) RedisDisplay() string {
	u, err := url.Parse(c.RedisURL); if err != nil { return "unparseable" }
	return u.Scheme + "://" + u.Host + u.Path
}
func (c Config) RedisIsUpstash() bool { return os.Getenv("UPSTASH_REDIS_URL") != "" }

// databaseURL: Neon wins when set (NEON_DATABASE_URL), else DATABASE_URL
// (offline local Docker Postgres), else the local default.
func databaseURL() string {
	if v := os.Getenv("NEON_DATABASE_URL"); v != "" { return v }
	return env("DATABASE_URL", "postgres://facefinder:facefinder@localhost:5433/facefinder?sslmode=disable")
}
// DatabaseDisplay is the log-safe form: scheme+host+db name, never credentials.
func (c Config) DatabaseDisplay() string {
	u, err := url.Parse(c.DatabaseURL); if err != nil { return "unparseable" }
	return u.Scheme + "://" + u.Host + u.Path
}
func (c Config) DatabaseIsNeon() bool { return os.Getenv("NEON_DATABASE_URL") != "" }
