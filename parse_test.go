package main

import (
	"reflect"
	"testing"
)

func TestParsePorts(t *testing.T) {
	got := parsePorts("80,443, 80, 1-3, 70000, a, 3-1")
	want := []string{"80", "443", "1", "2", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePorts mismatch: got=%v want=%v", got, want)
	}
}

func TestParseTargetsRange(t *testing.T) {
	got := parseTargets("192.168.1.10-12")
	want := []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseTargets mismatch: got=%v want=%v", got, want)
	}
}

func TestParseTTLAndGuessOS(t *testing.T) {
	if v := parseTTL("Reply from 1.1.1.1: bytes=32 time=2ms TTL=64"); v != 64 {
		t.Fatalf("unexpected ttl: %d", v)
	}
	if v := parseTTL("no ttl"); v != 0 {
		t.Fatalf("unexpected ttl for empty: %d", v)
	}
	if os := guessOS(64); os != "Linux/IoT" {
		t.Fatalf("unexpected os: %s", os)
	}
	if os := guessOS(128); os != "Windows" {
		t.Fatalf("unexpected os: %s", os)
	}
	if os := guessOS(200); os != "Unknown" {
		t.Fatalf("unexpected os: %s", os)
	}
}
