package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerRoutes(t *testing.T) {
	webDir := filepath.Join("..", "..", "web")
	server := BuildServer(webDir)
	handler := server.routes()

	req := httptest.NewRequest(http.MethodPost, "/api/tables", bytes.NewBufferString(`{
		"id": "t1", "name": "orders", "schema": "ods", "owner": "team",
		"fields": [{"name": "id", "type": "BIGINT", "primary": true}]
	}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tables", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var tables []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &tables); err != nil {
		t.Fatalf("bad list json: %v", err)
	}
	if len(tables) != 1 || tables[0]["ID"] != "t1" {
		t.Fatalf("unexpected table list: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("health failed: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "CatalogSvc") {
		t.Fatalf("browse page failed: %d", rec.Code)
	}
}

func TestBrowseHTML(t *testing.T) {
	html, err := browseHTML(filepath.Join("..", "..", "web"))
	if err != nil {
		t.Fatalf("browse html missing: %v", err)
	}
	if !bytes.Contains(html, []byte("数据目录")) {
		t.Fatalf("browse html content missing")
	}
}
