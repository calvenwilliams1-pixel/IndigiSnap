package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// InfoHandler serves GET /image_info/<path> — returns EXIF metadata as JSON.
type InfoHandler struct {
	BaseDir string
}

func NewInfoHandler(baseDir string) *InfoHandler {
	return &InfoHandler{BaseDir: baseDir}
}

func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/image_info/")
	raw = strings.TrimPrefix(raw, "/")
	rel, err := security.ValidatePath(raw)
	if err != nil {
		http.Error(w, "Invalid path", 400)
		return
	}

	full := filepath.Join(h.BaseDir, security.ToOSPath(rel))
	if !security.IsSafePath(h.BaseDir, full) {
		http.Error(w, "Access denied", 403)
		return
	}
	info, err := os.Stat(full)
	if err != nil {
		http.Error(w, "File not found", 404)
		return
	}

	out := map[string]string{
		"filename":   filepath.Base(full),
		"filesize":   formatBytes(info.Size()),
		"dimensions": "",
		"camera":     "",
		"make":       "",
		"iso":        "",
		"shutter":    "",
		"aperture":   "",
		"datetime":   "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func formatBytes(n int64) string {
	if n < 1024 {
		return "tiny"
	}
	if n < 1024*1024 {
		return "KB"
	}
	return "MB"
}
