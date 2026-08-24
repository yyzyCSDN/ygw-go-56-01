package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/ingest"
	"catalogsvc/internal/lineage"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/notify"
	"catalogsvc/internal/query"
)

// Server wires all catalog components behind one HTTP front.
type Server struct {
	webDir  string
	catalog *catalog.Catalog
	meta    *meta.Meta
	query   *query.Service
	lineage *lineage.Graph
	ingest  *ingest.Importer
	notify  *notify.Notifier
	http    *http.Server
}

// BuildServer constructs the component graph and HTTP handlers.
func BuildServer(webDir string) *Server {
	cat := catalog.New()
	notifier := notify.New()
	versioner := meta.New(cat, notifier)
	cache := catalog.NewCache()
	svc := query.New(versioner, cat, cache)
	graph := lineage.New()
	importer := ingest.New(cat, versioner, notifier)

	srv := &Server{
		webDir:  webDir,
		catalog: cat,
		meta:    versioner,
		query:   svc,
		lineage: graph,
		ingest:  importer,
		notify:  notifier,
	}
	srv.http = &http.Server{
		Handler:      srv.routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return srv
}

// Start serves HTTP until the listener fails or Shutdown is called.
func (s *Server) Start(addr string) error {
	log.Printf("catalog service listening on %s", addr)
	return s.http.ListenAndServe()
}

// Shutdown stops the HTTP server gracefully.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.http.Shutdown(ctx)
}
