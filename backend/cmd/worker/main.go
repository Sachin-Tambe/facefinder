package main

import (
	"context"; "log/slog"; "os"; "time"
	"github.com/hibiken/asynq"
	"facefinder/internal/config"; "facefinder/internal/database"; "facefinder/internal/face"; "facefinder/internal/jobs"; "facefinder/internal/service"; "facefinder/internal/storage"; "facefinder/internal/store"; "facefinder/internal/whatsapp"
)
func main(){
	cfg:=config.Load(); ctx:=context.Background()
	pool,err:=database.NewPool(ctx,cfg.DatabaseURL); if err!=nil{slog.Error("db unreachable - check NEON_DATABASE_URL / DATABASE_URL","db",cfg.DatabaseDisplay(),"neon",cfg.DatabaseIsNeon(),"err",err);os.Exit(1)}; defer pool.Close()
	slog.Info("database connected","db",cfg.DatabaseDisplay(),"neon",cfg.DatabaseIsNeon())
	st:=store.New(pool)
	var fp face.Provider = face.Fake{}; if cfg.AWSEnabled(){ if p,err:=face.NewAWS(ctx,cfg); err==nil{fp=p} }
	var sp storage.Provider = storage.Local{Root:".local-r2",APIPublic:"http://localhost:8080"}; if cfg.R2Enabled(){ if p,err:=storage.NewR2(ctx,cfg); err==nil{sp=p} }
	var wa whatsapp.Provider = whatsapp.Log{}; if cfg.WAEnabled(){ wa=whatsapp.Gateway{Base:cfg.WAGatewayURL,Key:cfg.WAGatewayKey} }
	svc:=&service.Service{Cfg:cfg,Store:st,Face:fp,Storage:sp,WA:wa}
	svc.Enqueue=jobs.Enqueue(jobs.Client(cfg.RedisURL))
	if err:=jobs.Ping(cfg.RedisURL); err!=nil { slog.Error("redis unreachable - check UPSTASH_REDIS_URL / REDIS_URL","redis",cfg.RedisDisplay(),"upstash",cfg.RedisIsUpstash(),"err",err) } else { slog.Info("redis connected","redis",cfg.RedisDisplay(),"upstash",cfg.RedisIsUpstash()) }
	// hourly retention sweep (§31-34), also runnable on demand via asynq task
	go func(){ t:=time.NewTicker(time.Hour); defer t.Stop(); for range t.C { if err:=svc.SweepRetention(context.Background()); err!=nil{slog.Error("retention","err",err)} } }()
	slog.Info("worker starting")
	if err:=jobs.Server(cfg.RedisURL,svc).Run(jobs.Mux(svc)); err!=nil && err!=asynq.ErrServerClosed {slog.Error("worker","err",err);os.Exit(1)}
}
