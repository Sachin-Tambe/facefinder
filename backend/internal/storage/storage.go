// Package storage: StorageProvider abstraction (§40). Cloudflare R2 via S3-compatible API.
package storage

import ("context";"fmt";"os";"path/filepath";"strings";"time")

type Provider interface {
	PresignUpload(ctx context.Context, key, mime string, ttl time.Duration) (string, error)
	PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error)
	Put(ctx context.Context, key string, data []byte, mime string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}

// LocalProvider: dev fallback writing under .local-r2/ . Presigned URLs point at the API's dev object route.
type Local struct{ Root, APIPublic string }
func (l Local) path(k string) string { return filepath.Join(l.Root, filepath.Clean("/"+k)) }
func (l Local) PresignUpload(_ context.Context, k, _ string, _ time.Duration) (string,error) { return fmt.Sprintf("%s/dev-objects/%s", l.APIPublic, k), nil }
func (l Local) PresignDownload(_ context.Context, k string, _ time.Duration) (string,error) { return fmt.Sprintf("%s/dev-objects/%s", l.APIPublic, k), nil }
func (l Local) Put(_ context.Context, k string, d []byte, _ string) error { p:=l.path(k); if err:=os.MkdirAll(filepath.Dir(p),0o755); err!=nil {return err}; return os.WriteFile(p,d,0o644) }
func (l Local) Get(_ context.Context, k string) ([]byte,error) { return os.ReadFile(l.path(k)) }
func (l Local) Delete(_ context.Context, k string) error { err:=os.Remove(l.path(k)); if os.IsNotExist(err) {return nil}; return err }
func (l Local) DeletePrefix(_ context.Context, prefix string) error {
	return filepath.Walk(l.path(prefix), func(p string, info os.FileInfo, err error) error { if err==nil && !info.IsDir() { return os.Remove(p) }; return nil })
}
var _ Provider = Local{}
var _ = strings.TrimSpace
