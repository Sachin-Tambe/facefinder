// Asynq jobs (§30): retries with backoff, max attempts, idempotent handlers.
package jobs

import (
	"context"; "encoding/json"; "time"
	"github.com/hibiken/asynq"; "github.com/redis/go-redis/v9"
	"facefinder/internal/service"
)
const (
	PhotoProcess="photo.process" // variants + IndexFaces (idempotent: faces upsert on aws_face_id, status overwrite)
	SearchSelfie="search.process_selfie"
	SearchCleanup="search.cleanup_selfie"
	RetentionSweep="retention.sweep"
)

// RedisOpt parses redis:// (offline local) or rediss:// (Upstash, TLS) URLs,
// carrying username/password/DB/TLS through to Asynq. Upstash requires rediss://.
func RedisOpt(redisURL string) asynq.RedisClientOpt {
	o, err := redis.ParseURL(redisURL)
	if err != nil { return asynq.RedisClientOpt{Addr:"localhost:6379"} }
	opt := asynq.RedisClientOpt{Addr:o.Addr, Username:o.Username, Password:o.Password, DB:o.DB, PoolSize:10,
		DialTimeout:10*time.Second, ReadTimeout:10*time.Second, WriteTimeout:10*time.Second}
	if o.TLSConfig != nil { opt.TLSConfig = o.TLSConfig } // rediss:// (Upstash) — go-redis sets this from the scheme
	return opt
}
// Ping verifies connectivity at startup so a wrong Upstash URL fails loudly in logs, not silently at enqueue time.
func Ping(redisURL string) error {
	o, err := redis.ParseURL(redisURL); if err != nil { return err }
	c := redis.NewClient(o); defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
	return c.Ping(ctx).Err()
}
func Client(redisURL string) *asynq.Client { return asynq.NewClient(RedisOpt(redisURL)) }
func Enqueue(c *asynq.Client) func(string,map[string]string) error {
	return func(kind string, payload map[string]string) error {
		name:=kind; switch kind { case "photo.process": name=PhotoProcess; case "search.cleanup_selfie": name=SearchCleanup }
		b,_:=json.Marshal(payload); t:=asynq.NewTask(name,b)
		_,err:=c.Enqueue(t, asynq.MaxRetry(5), asynq.Timeout(10*time.Minute), asynq.Retention(24*time.Hour))
		return err
	}
}
func Mux(svc *service.Service) *asynq.ServeMux {
	m:=asynq.NewServeMux()
	m.HandleFunc(PhotoProcess, func(ctx context.Context,t *asynq.Task) error { var p map[string]string; _=json.Unmarshal(t.Payload(),&p); return svc.ProcessPhoto(ctx,p["photo_id"]) })
	m.HandleFunc(SearchCleanup, func(ctx context.Context,t *asynq.Task) error { var p map[string]string; _=json.Unmarshal(t.Payload(),&p); return svc.CleanupSelfie(ctx,p["search_id"]) })
	m.HandleFunc(RetentionSweep, func(ctx context.Context,_ *asynq.Task) error { return svc.SweepRetention(ctx) })
	return m
}
func Server(redisURL string, svc *service.Service) *asynq.Server {
	return asynq.NewServer(RedisOpt(redisURL), asynq.Config{Concurrency:10, Queues:map[string]int{"default":1}, RetryDelayFunc: asynq.DefaultRetryDelayFunc})
}
