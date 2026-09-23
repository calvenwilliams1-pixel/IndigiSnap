package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// BrowserHandler serves GET /folder_browser?path=<path>
// Returns a JSON list of subfolder names for the given path.
type BrowserHandler struct {
	BaseDir string
}

func NewBrowserHandler(baseDir string) *BrowserHandler {
	return &BrowserHandler{BaseDir: baseDir}
}

func (h *BrowserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("path")
	folder, _ := security.ValidatePath(raw)

	full := h.BaseDir
	if folder != "" {
		full = filepath.Join(h.BaseDir, security.ToOSPath(folder))
	}
	if !security.IsSafePath(h.BaseDir, full) {
		writeJSONError(w, "Access denied", 403)
		return
	}

	entries, err := os.ReadDir(full)
	if err != nil {
		writeJSONError(w, "Cannot read folder: "+err.Error(), 500)
		return
	}

	names := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || security.IsHiddenFolder(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"current_path": folder,
		"folders":      names,
	})
}

func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
