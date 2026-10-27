package jobs

import "testing"

func TestRedisOptUpstashTLS(t *testing.T){
	o:=RedisOpt("rediss://default:secret@eu1-xxxx.upstash.io:6379")
	if o.Addr!="eu1-xxxx.upstash.io:6379" { t.Fatalf("addr %s",o.Addr) }
	if o.Username!="default" || o.Password!="secret" { t.Fatalf("credentials not carried through") }
	if o.TLSConfig==nil { t.Fatal("rediss:// must enable TLS for Upstash") }
}
func TestRedisOptOfflineLocal(t *testing.T){
	o:=RedisOpt("redis://localhost:6379/0")
	if o.Addr!="localhost:6379" || o.TLSConfig!=nil { t.Fatalf("local offline redis must be plaintext: %+v",o) }
}
