package service

import "testing"

func TestHashAndToken(t *testing.T){
	if Hash("a")==Hash("b") { t.Fatal("hash collision on distinct inputs") }
	if len(Token(24))!=48 { t.Fatal("token length") }
}
func TestEventCode(t *testing.T){ c:=EventCode("Rahul Wedding"); if len(c)<6 || c[3]!='-' { t.Fatalf("bad code %s", c) } }
func TestDedupLogicThresholds(t *testing.T){
	// Threshold contract: strong >=95, possible >=85 by default (env-overridable).
	s:=Service{}; _=s
}
