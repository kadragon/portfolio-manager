package main

import "testing"

func TestDefaultAddrBindsContainerInterface(t *testing.T) {
	t.Setenv("PORTFOLIO_ADDR", "")
	if got := defaultAddr(); got != "0.0.0.0:8000" {
		t.Fatalf("defaultAddr() = %q, want 0.0.0.0:8000", got)
	}
}

func TestDefaultAddrUsesEnvOverride(t *testing.T) {
	t.Setenv("PORTFOLIO_ADDR", "127.0.0.1:9000")
	if got := defaultAddr(); got != "127.0.0.1:9000" {
		t.Fatalf("defaultAddr() = %q, want env override", got)
	}
}
