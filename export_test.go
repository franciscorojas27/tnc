package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportJSONAndTXT(t *testing.T) {
	globalResults = []HostResult{
		{
			Target: "host1",
			IP:     "10.0.0.1",
			Status: "Successful",
			OS:     "Windows",
			Ports:  []PortResult{{Port: "80", Protocol: "TCP", Service: "nginx"}},
		},
	}

	d := t.TempDir()
	jsonFile := filepath.Join(d, "out.json")
	txtFile := filepath.Join(d, "out.txt")

	if err := exportResults(jsonFile, "json"); err != nil {
		t.Fatalf("json export failed: %v", err)
	}
	if err := exportResults(txtFile, "txt"); err != nil {
		t.Fatalf("txt export failed: %v", err)
	}

	jb, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("read json failed: %v", err)
	}
	if !strings.Contains(string(jb), "10.0.0.1") {
		t.Fatalf("json content mismatch: %s", string(jb))
	}

	tb, err := os.ReadFile(txtFile)
	if err != nil {
		t.Fatalf("read txt failed: %v", err)
	}
	if !strings.Contains(string(tb), "Port 80 [TCP] nginx") {
		t.Fatalf("txt content mismatch: %s", string(tb))
	}
}
