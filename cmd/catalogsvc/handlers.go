package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"catalogsvc/internal/ingest"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
	"catalogsvc/internal/walk"
)

type tableRequest struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Schema string        `json:"schema"`
	Owner  string        `json:"owner"`
	Fields []model.Field `json:"fields"`
}

type edgeRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
}

// routes builds the HTTP mux for the catalog service.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleBrowse)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/tables", s.handleTables)
	mux.HandleFunc("/api/tables/", s.handleTable)
	mux.HandleFunc("/api/snapshots", s.handleSnapshots)
	mux.HandleFunc("/api/lineage/nodes", s.handleNodes)
	mux.HandleFunc("/api/lineage/edges", s.handleEdges)
	mux.HandleFunc("/api/lineage/upstream/", s.handleUpstream)
	mux.HandleFunc("/api/lineage/downstream/", s.handleDownstream)
	mux.HandleFunc("/api/ingest/begin", s.handleIngestBegin)
	mux.HandleFunc("/api/ingest/apply", s.handleIngestApply)
	mux.HandleFunc("/api/ingest/commit", s.handleIngestCommit)
	mux.HandleFunc("/api/ingest/rollback", s.handleIngestRollback)
	return mux
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	builder := notify.NewSnapshotBuilder(s.catalog)
	tableID := r.URL.Query().Get("table")
	if tableID != "" {
		version, err := s.meta.Current(tableID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		snapshot, err := builder.Build(tableID, version.Number)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
		return
	}
	writeJSON(w, http.StatusOK, builder.BuildAll())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleTables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.query.ListTables())
	case http.MethodPost:
		var req tableRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		table := &model.Table{
			ID:       req.ID,
			Name:     req.Name,
			Schema:   req.Schema,
			Owner:    req.Owner,
			Fields:   req.Fields,
			Revision: 1,
		}
		if err := s.catalog.Register(table); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		s.lineage.AddNode(req.ID)
		writeJSON(w, http.StatusCreated, table)
	default:
		writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

func (s *Server) handleTable(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tables/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, errors.New("bad table id"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		table, err := s.query.GetTable(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		fields, ferr := s.query.SortedFields(id)
		if ferr != nil {
			fields = table.Fields
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"table":  table,
			"fields": fields,
		})
	case http.MethodPut:
		var req tableRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		schema := model.Schema{Fields: req.Fields}
		if err := s.meta.ApplySchema(id, schema); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": "applied"})
	default:
		writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	var req edgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.lineage.AddNode(req.From)
	writeJSON(w, http.StatusOK, map[string]string{"node": req.From})
}

func (s *Server) handleEdges(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req edgeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.lineage.AddEdge(req.From, req.To, req.Reason); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		_ = s.notify.Push(context.Background(), model.ChangeEvent{
			TableID: req.From,
			Type:    model.EventLineageChanged,
		})
		writeJSON(w, http.StatusOK, map[string]string{"edge": req.From + "->" + req.To})
	case http.MethodDelete:
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		s.lineage.DeleteEdge(from, to)
		_ = s.notify.Push(context.Background(), model.ChangeEvent{
			TableID: from,
			Type:    model.EventLineageChanged,
		})
		writeJSON(w, http.StatusOK, map[string]string{"deleted": from + "->" + to})
	default:
		writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

func (s *Server) handleUpstream(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/lineage/upstream/")
	ids := walk.Upstream(s.lineage, id)
	writeJSON(w, http.StatusOK, map[string]any{
		"root":  id,
		"tables": ids,
	})
}

func (s *Server) handleDownstream(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/lineage/downstream/")
	ids := walk.Downstream(s.lineage, id)
	writeJSON(w, http.StatusOK, map[string]any{
		"root":   id,
		"tables": ids,
	})
}

func (s *Server) handleIngestBegin(w http.ResponseWriter, r *http.Request) {
	if err := s.ingest.Begin(); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "begun"})
}

func (s *Server) handleIngestApply(w http.ResponseWriter, r *http.Request) {
	var segment ingest.Segment
	if err := json.NewDecoder(r.Body).Decode(&segment); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.ingest.Apply(segment); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "applied"})
}

func (s *Server) handleIngestCommit(w http.ResponseWriter, r *http.Request) {
	if err := s.ingest.Commit(); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "committed"})
}

func (s *Server) handleIngestRollback(w http.ResponseWriter, r *http.Request) {
	if err := s.ingest.Rollback(); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rolled-back"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
