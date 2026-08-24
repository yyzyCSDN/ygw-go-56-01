package main

import (
	"net/http"
	"os"
	"path/filepath"
)

// handleBrowse serves the catalog browsing page at the root path.
func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	html, err := browseHTML(s.webDir)
	if err != nil {
		http.Error(w, "browse page unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(html)
}

// browseHTML loads the browse page from the configured web directory.
func browseHTML(webDir string) ([]byte, error) {
	path := filepath.Join(webDir, "browse.html")
	return os.ReadFile(path)
}
