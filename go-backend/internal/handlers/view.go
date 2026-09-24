package handlers

import (
	"bytes"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// ViewHandler serves GET /view/<path> — raw image or video bytes.
//
// For JPEGs with EXIF orientation != 1, the image is decoded, rotated, and
// re-encoded before serving so it displays upright in browsers and WebViews.
// The original file on disk is never modified.
type ViewHandler struct {
	BaseDir string
}

func NewViewHandler(baseDir string) *ViewHandler {
	return &ViewHandler{BaseDir: baseDir}
}

func (h *ViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/view")
	raw = strings.TrimPrefix(raw, "/")

	rel, err := security.ValidatePath(raw)
	if err != nil {
		http.Error(w, "Invalid path: "+err.Error(), 400)
		return
	}

	full := filepath.Join(h.BaseDir, security.ToOSPath(rel))
	if !security.IsSafePath(h.BaseDir, full) {
		http.Error(w, "Access denied", 403)
		return
	}

	info, err := os.Stat(full)
	if err != nil {
		log.Printf("view: file not found: %s", full)
		http.Error(w, "File not found", 404)
		return
	}
	if info.IsDir() {
		http.Error(w, "Cannot view a directory", 400)
		return
	}

	// Cache headers apply to both rotated and unmodified responses.
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// Only JPEGs have EXIF orientation. Everything else serves unmodified.
	if !isJpeg(full) {
		http.ServeFile(w, r, full)
		return
	}

	orientation := readOrientation(full)
	if orientation == 1 {
		// Normal orientation: serve original bytes, no re-encode.
		http.ServeFile(w, r, full)
		return
	}

	if err := serveRotated(w, full, orientation); err != nil {
		// Rotation failed for any reason — fall back to serving the original.
		log.Printf("view: rotation failed for %s (orientation=%d): %v", full, orientation, err)
		http.ServeFile(w, r, full)
		return
	}
}

// readOrientation returns the EXIF orientation tag (1–8).
// Returns 1 (normal) if no EXIF, decode failure, or out-of-range value.
func readOrientation(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 1
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return 1
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	n, err := tag.Int(0)
	if err != nil || n < 1 || n > 8 {
		return 1
	}
	return n
}

// serveRotated decodes the JPEG, rotates based on orientation, and serves
// the re-encoded result. Only orientations 3, 6, and 8 are handled; other
// values (including mirror variants 2, 4, 5, 7) are treated as 1 by the caller.
func serveRotated(w http.ResponseWriter, path string, orientation int) error {
	src, err := imaging.Open(path, imaging.AutoOrientation(false))
	if err != nil {
		return err
	}

	// Go's imaging.AutoOrientation would rotate based on EXIF, but we're
	// handling it manually so behavior is explicit and testable.
	//
	// Camera orientation 6 means "rotated 90 CW" — the raw pixels are stored
	// sideways, so we rotate 270 to undo it. Orientation 8 is the opposite.
	var rotated = src
	switch orientation {
	case 3:
		rotated = imaging.Rotate180(src)
	case 6:
		rotated = imaging.Rotate270(src)
	case 8:
		rotated = imaging.Rotate90(src)
	default:
		// Caller only invokes us for 3, 6, 8. Defensive: return src unchanged.
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rotated, &jpeg.Options{Quality: 100}); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", intToStr(buf.Len()))
	_, err = w.Write(buf.Bytes())
	return err
}

// intToStr is a small int-to-string helper to avoid importing strconv.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
