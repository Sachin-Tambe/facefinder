package main

import ("context";"log";"os";"path/filepath";"sort"
	"facefinder/internal/config"; "facefinder/internal/database")
func main(){
	cfg:=config.Load()
	pool,err:=database.NewPool(context.Background(),cfg.DatabaseURL); if err!=nil{log.Fatalf("db unreachable (%s, neon=%v): %v",cfg.DatabaseDisplay(),cfg.DatabaseIsNeon(),err)}; defer pool.Close()
	log.Printf("database connected: %s (neon=%v)",cfg.DatabaseDisplay(),cfg.DatabaseIsNeon())
	if _,err:=pool.Exec(context.Background(),`CREATE TABLE IF NOT EXISTS schema_migrations(name TEXT PRIMARY KEY)`); err!=nil{log.Fatal(err)}
	files,_:=filepath.Glob("migrations/*.sql"); sort.Strings(files)
	for _,f:=range files { var done bool; _=pool.QueryRow(context.Background(),`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)`,f).Scan(&done); if done{continue}
		b,err:=os.ReadFile(f); if err!=nil{log.Fatal(err)}
		if _,err:=pool.Exec(context.Background(),string(b)); err!=nil{log.Fatalf("%s: %v",f,err)}
		if _,err:=pool.Exec(context.Background(),`INSERT INTO schema_migrations(name) VALUES($1)`,f); err!=nil{log.Fatal(err)}
		log.Println("applied",f) }
	log.Println("migrations up to date")
}
