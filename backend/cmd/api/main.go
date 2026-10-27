package main

import (
	"context"; "log/slog"; "net/http"; "os"; "path/filepath"
	"facefinder/internal/config"; "facefinder/internal/database"; "facefinder/internal/face"; "facefinder/internal/httpapi"; "facefinder/internal/jobs"; "facefinder/internal/service"; "facefinder/internal/storage"; "facefinder/internal/store"; "facefinder/internal/whatsapp"
)
func main(){
	cfg:=config.Load(); ctx:=context.Background()
	pool,err:=database.NewPool(ctx,cfg.DatabaseURL); if err!=nil{slog.Error("db unreachable - check NEON_DATABASE_URL / DATABASE_URL","db",cfg.DatabaseDisplay(),"neon",cfg.DatabaseIsNeon(),"err",err);os.Exit(1)}; defer pool.Close()
	slog.Info("database connected","db",cfg.DatabaseDisplay(),"neon",cfg.DatabaseIsNeon())
	st:=store.New(pool)
	var fp face.Provider = face.Fake{}; if cfg.AWSEnabled(){ if p,err:=face.NewAWS(ctx,cfg); err==nil{fp=p} else {slog.Error("aws init, using fake","err",err)} }
	var sp storage.Provider = storage.Local{Root:".local-r2", APIPublic:"http://localhost:8080"}
	if cfg.R2Enabled(){ if p,err:=storage.NewR2(ctx,cfg); err==nil{sp=p} else {slog.Error("r2 init, using local","err",err)} }
	_ = filepath.Join
	var wa whatsapp.Provider = whatsapp.Log{}; if cfg.WAEnabled(){ wa=whatsapp.Gateway{Base:cfg.WAGatewayURL,Key:cfg.WAGatewayKey} }
	svc:=&service.Service{Cfg:cfg,Store:st,Face:fp,Storage:sp,WA:wa}
	if client:=jobs.Client(cfg.RedisURL); client!=nil { svc.Enqueue=jobs.Enqueue(client) }
	if err:=jobs.Ping(cfg.RedisURL); err!=nil { slog.Error("redis unreachable - jobs will fail until fixed","redis",cfg.RedisDisplay(),"upstash",cfg.RedisIsUpstash(),"err",err) } else { slog.Info("redis connected","redis",cfg.RedisDisplay(),"upstash",cfg.RedisIsUpstash()) }
	srv:=&httpapi.Server{Cfg:cfg,Svc:svc,Store:st}
	slog.Info("api listening","addr",cfg.Addr,"aws",cfg.AWSEnabled(),"r2",cfg.R2Enabled(),"whatsapp",cfg.WAEnabled())
	if err:=http.ListenAndServe(cfg.Addr, srv.Handler()); err!=nil{slog.Error("serve","err",err);os.Exit(1)}
}
