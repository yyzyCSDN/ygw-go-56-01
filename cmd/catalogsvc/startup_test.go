package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRealStartup boots the actual HTTP server on a real loopback listener
// and probes it over TCP, exercising the service entry point end to end.
func TestRealStartup(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server := BuildServer(filepath.Join("..", "..", "web"))
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.http.Serve(listener)
	}()
	defer func() {
		_ = server.Shutdown()
		select {
		case <-serveErr:
		case <-time.After(2 * time.Second):
		}
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 3 * time.Second}

	health, err := client.Get(base + "/api/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	healthBody, _ := io.ReadAll(health.Body)
	_ = health.Body.Close()
	if health.StatusCode != http.StatusOK || !bytes.Contains(healthBody, []byte("ok")) {
		t.Fatalf("health probe failed: %d %s", health.StatusCode, healthBody)
	}

	payload := `{"id":"t1","name":"orders","schema":"ods","owner":"team","fields":[{"name":"id","type":"BIGINT","primary":true}]}`
	register, err := client.Post(base+"/api/tables", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	_ = register.Body.Close()
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", register.StatusCode)
	}

	list, err := client.Get(base + "/api/tables")
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	listBody, _ := io.ReadAll(list.Body)
	_ = list.Body.Close()
	var tables []map[string]any
	if err := json.Unmarshal(listBody, &tables); err != nil || len(tables) != 1 || tables[0]["ID"] != "t1" {
		t.Fatalf("list probe failed: %s", listBody)
	}

	browse, err := client.Get(base + "/")
	if err != nil {
		t.Fatalf("browse request failed: %v", err)
	}
	browseBody, _ := io.ReadAll(browse.Body)
	_ = browse.Body.Close()
	if browse.StatusCode != http.StatusOK || !bytes.Contains(browseBody, []byte("CatalogSvc")) {
		t.Fatalf("browse probe failed: %d", browse.StatusCode)
	}
}
