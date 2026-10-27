package config

import "testing"

func TestDatabaseURLNeonWins(t *testing.T){
	t.Setenv("NEON_DATABASE_URL","postgres://u:p@ep-x-pooler.aws.neon.tech/db?sslmode=require")
	t.Setenv("DATABASE_URL","postgres://facefinder:facefinder@localhost:5433/facefinder?sslmode=disable")
	if got:=databaseURL(); got!="postgres://u:p@ep-x-pooler.aws.neon.tech/db?sslmode=require" { t.Fatalf("neon must win, got %s",got) }
}
func TestDatabaseURLOfflineFallback(t *testing.T){
	t.Setenv("NEON_DATABASE_URL","")
	t.Setenv("DATABASE_URL","postgres://facefinder:facefinder@localhost:5433/facefinder?sslmode=disable")
	if got:=databaseURL(); got!="postgres://facefinder:facefinder@localhost:5433/facefinder?sslmode=disable" { t.Fatalf("offline fallback, got %s",got) }
}
func TestRedisURLUpstashWins(t *testing.T){
	t.Setenv("UPSTASH_REDIS_URL","rediss://default:x@y.upstash.io:6379")
	t.Setenv("REDIS_URL","redis://localhost:6379/0")
	if got:=redisURL(); got!="rediss://default:x@y.upstash.io:6379" { t.Fatalf("upstash must win, got %s",got) }
}
