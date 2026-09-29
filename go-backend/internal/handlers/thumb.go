package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/paths"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// ThumbHandler serves GET /thumb/<videoRelPath> from ThumbDir.
//
// The URL path is the relative path to the video (e.g. "Tokyo day 2/clip.mp4").
// The handler maps it to {ThumbDir}/<relPath>.jpg and serves that file.
// No EXIF rotation is applied — thumbnails are generated pre-oriented by
// ffmpeg from already-processed frames.
type ThumbHandler struct{}

func NewThumbHandler() *ThumbHandler {
	return &ThumbHandler{}
}

func (h *ThumbHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/thumb")
	raw = strings.TrimPrefix(raw, "/")

	rel, err := security.ValidatePath(raw)
	if err != nil {
		http.Error(w, "Invalid path: "+err.Error(), 400)
		return
	}

	thumbPath := paths.ThumbFile(rel)
	if thumbPath == "" {
		http.Error(w, "Invalid thumbnail path", 400)
		return
	}

	// Containment check: thumbPath must be inside ThumbDir.
	if !security.IsSafePath(paths.ThumbDir(), thumbPath) {
		http.Error(w, "Access denied", 403)
		return
	}

	info, err := os.Stat(thumbPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if info.IsDir() {
		http.Error(w, "Cannot view a directory", 400)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, thumbPath)
	_ = log.Prefix // silence unused import if log ends up unused
}
