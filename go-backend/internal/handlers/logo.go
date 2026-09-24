package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// LogoHandler serves GET /logo/<filename>.
//
// Only files inside BASE_DIR/logo/ are served. Hidden files (leading dot)
// are rejected in both the selection logic (meta.GetLogoPath) and here.
//
// SVG is allowed because this is a single-user installation. Multi-user
// deployments may wish to restrict SVG uploads because SVG can contain
// active content.
type LogoHandler struct {
	BaseDir string
}

func NewLogoHandler(baseDir string) *LogoHandler {
	return &LogoHandler{BaseDir: baseDir}
}

func (h *LogoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logoDir := filepath.Join(h.BaseDir, "logo")

	// Extract the requested filename from the URL.
	name := strings.TrimPrefix(r.URL.Path, "/logo/")
	name = filepath.Clean(name)

	// Reject hidden files (defense in depth - GetLogoPath also filters).
	if strings.HasPrefix(filepath.Base(name), ".") {
		http.NotFound(w, r)
		return
	}

	// Reject empty or directory-referencing names.
	if name == "" || name == "." || name == "/" {
		http.NotFound(w, r)
		return
	}

	// Build the full path - always resolved against logoDir.
	full := filepath.Join(logoDir, name)

	// Containment check via filepath.Rel - rejects ../ escapes.
	rel, err := filepath.Rel(logoDir, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "Access denied", 403)
		return
	}

	// Final safety net: must be inside BaseDir.
	if !security.IsSafePath(h.BaseDir, full) {
		http.Error(w, "Access denied", 403)
		return
	}

	info, err := os.Stat(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if info.IsDir() {
		http.Error(w, "Cannot view a directory", 400)
		return
	}

	// Logo is branding; users may swap it. Prevent stale caching.
	w.Header().Set("Cache-Control", "no-cache")

	http.ServeFile(w, r, full)
}
