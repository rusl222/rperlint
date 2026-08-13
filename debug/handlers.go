package debug

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

type stateResponse struct {
	Connected bool `json:"connected"`
}

type setRequest struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Dost    string `json:"dost"`
	Timeout int    `json:"timeout"`
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "" {
		http.ServeFile(w, r, filepath.Join(s.wwwDir, "index.html"))
		return
	}

	filePath := filepath.Join(s.wwwDir, filepath.FromSlash(path))
	http.ServeFile(w, r, filePath)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	resp := stateResponse{
		Connected: s.proxy.Connected(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {

	list := s.proxy.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req setRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, ok := s.proxy.Type(req.Name)

	if !ok {
		http.Error(w, "unknown reper", http.StatusNotFound)
		return
	}

	var (
		value any
		err   error
	)

	switch t {

	case TypeFloat:
		value, err = strconv.ParseFloat(req.Value, 64)

	case TypeInt:
		value, err = strconv.Atoi(req.Value)

	case TypeBool:
		value, err = strconv.ParseBool(req.Value)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.proxy.Set(req.Name, value)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
