package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// ViewHandler serves GET /view/<path> — raw image/video bytes.
type ViewHandler struct {
	BaseDir string
}

func NewViewHandler(baseDir string) *ViewHandler {
	return &ViewHandler{BaseDir: baseDir}
}

func (h *ViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/view")
	rawPath = strings.TrimPrefix(rawPath, "/")

	rel, err := security.ValidatePath(rawPath)
	if err != nil {
		http.Error(w, "Invalid path: "+err.Error(), 400)
		return
	}

	fullPath := filepath.Join(h.BaseDir, security.ToOSPath(rel))
	if !security.IsSafePath(h.BaseDir, fullPath) {
		http.Error(w, "Access denied", 403)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		log.Printf("View: file not found: %s", fullPath)
		http.Error(w, "File not found", 404)
		return
	}
	if info.IsDir() {
		http.Error(w, "Cannot view a directory", 400)
		return
	}

	// Cache headers
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// http.ServeFile handles Range requests, content-type detection, etc.
	http.ServeFile(w, r, fullPath)
}
