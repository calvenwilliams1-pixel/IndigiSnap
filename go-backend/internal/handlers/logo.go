package handlers

import (
	"bytes"
	"io"
	"log"
	"mime"
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

const maxLogoSize = 5 << 20 // 5 MB

// allowedLogoExts is the set of image extensions accepted for a logo upload.
var allowedLogoExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".svg":  true,
}

// SetLogo handles POST /set_logo.
//
// Accepts a multipart file upload from the "logo" form field. Validates
// MIME type, extension, and size. Deletes any existing logo files in
// BASE_DIR/logo/ and saves the new one as logo.<ext>.
//
// Redirects to /browse on success. Returns 400/413/500 on failure.
//
// SVG is allowed for personal use. See note on LogoHandler about SVG
// active content risk for multi-user deployments.
func (h *LogoHandler) SetLogo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxLogoSize); err != nil {
		http.Error(w, "Cannot parse upload: "+err.Error(), 400)
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		http.Error(w, "No logo file provided", 400)
		return
	}
	defer file.Close()

	// Size cap (header.Size is available before reading).
	if header.Size > maxLogoSize {
		http.Error(w, "Logo too large (max 5 MB)", 413)
		return
	}

	// Read first 512 bytes to detect MIME.
	buf := make([]byte, 512)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		http.Error(w, "Cannot read upload", 500)
		return
	}
	detectedMime := http.DetectContentType(buf[:n])

	// SVG isn't always detected by DetectContentType. Accept it if the
	// declared extension is .svg and the bytes look like XML/SVG.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedLogoExts[ext] {
		http.Error(w, "Extension not allowed: "+ext, 400)
		return
	}

	isImageMime := strings.HasPrefix(detectedMime, "image/")
	isSvgByExt := ext == ".svg" && (bytes.Contains(buf[:n], []byte("<svg")) || bytes.Contains(buf[:n], []byte("<?xml")))
	if !isImageMime && !isSvgByExt {
		http.Error(w, "Not an image file (detected: "+detectedMime+")", 400)
		return
	}

	// Reset file pointer to start for full copy.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Cannot reset upload stream", 500)
		return
	}

	// Prepare logo dir.
	logoDir := filepath.Join(h.BaseDir, "logo")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		http.Error(w, "Cannot create logo dir: "+err.Error(), 500)
		return
	}

	// Write new file to a temp name first.
	tempPath := filepath.Join(logoDir, ".uploading" + ext)
	out, err := os.Create(tempPath)
	if err != nil {
		http.Error(w, "Cannot create temp file: "+err.Error(), 500)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		os.Remove(tempPath)
		http.Error(w, "Cannot write upload: "+err.Error(), 500)
		return
	}
	out.Close()

	// Delete existing logo files (any extension) before renaming.
	entries, _ := os.ReadDir(logoDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(name), "logo.") {
			continue
		}
		_ = os.Remove(filepath.Join(logoDir, name))
	}

	// Atomic rename to final name.
	finalPath := filepath.Join(logoDir, "logo"+ext)
	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		http.Error(w, "Cannot finalize logo: "+err.Error(), 500)
		return
	}

	log.Printf("SetLogo: saved %s (%d bytes, %s)", finalPath, header.Size, detectedMime)

	http.Redirect(w, r, "/browse", http.StatusSeeOther)
}
