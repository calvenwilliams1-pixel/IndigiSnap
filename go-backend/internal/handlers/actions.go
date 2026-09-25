package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/meta"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/video"
)

// ActionHandler handles POST operations: create/rename/delete folders,
// delete files, toggle favorites, set sort.
type ActionHandler struct {
	BaseDir string
}

func NewActionHandler(baseDir string) *ActionHandler {
	return &ActionHandler{BaseDir: baseDir}
}

// CreateFolder handles POST /create_folder/<path>
func (h *ActionHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/create_folder/")
	raw = strings.TrimPrefix(raw, "/")
	folder, _ := security.ValidatePath(raw)

	parentPath := h.BaseDir
	if folder != "" && folder != "ROOT_DIR" {
		parentPath = filepath.Join(h.BaseDir, security.ToOSPath(folder))
	}
	if !security.IsSafePath(h.BaseDir, parentPath) {
		http.Error(w, "Access denied", 403)
		return
	}

	newName := security.SanitizeFilename(r.FormValue("new_name"))
	if newName == "" || newName == "untitled" {
		http.Error(w, "Invalid folder name", 400)
		return
	}
	if security.IsHiddenFolder(newName) {
		http.Error(w, "Reserved folder name", 403)
		return
	}

	newPath := filepath.Join(parentPath, newName)
	if !security.IsSafePath(h.BaseDir, newPath) {
		http.Error(w, "Access denied", 403)
		return
	}
	if err := os.MkdirAll(newPath, 0755); err != nil {
		http.Error(w, "Cannot create folder: "+err.Error(), 500)
		return
	}
	meta.SaveMeta(newPath, meta.DefaultMeta())

	// Redirect back to where we were
	back := "/browse"
	if folder != "" && folder != "ROOT_DIR" {
		back = "/browse/" + url.QueryEscape(folder)
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// RenameFolder handles POST /rename_folder/<path>
func (h *ActionHandler) RenameFolder(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/rename_folder/")
	raw = strings.TrimPrefix(raw, "/")
	folder, _ := security.ValidatePath(raw)

	oldPath := filepath.Join(h.BaseDir, security.ToOSPath(folder))
	if !security.IsSafePath(h.BaseDir, oldPath) {
		http.Error(w, "Access denied", 403)
		return
	}

	newName := security.SanitizeFilename(r.FormValue("new_name"))
	if newName == "" || security.IsHiddenFolder(newName) {
		http.Error(w, "Invalid new name", 400)
		return
	}

	parent := filepath.Dir(oldPath)
	newPath := filepath.Join(parent, newName)
	if !security.IsSafePath(h.BaseDir, newPath) {
		http.Error(w, "Access denied", 403)
		return
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		http.Error(w, "Rename failed: "+err.Error(), 500)
		return
	}

	parentRel := strings.TrimPrefix(filepath.Dir(folder), ".")
	parentRel = strings.TrimPrefix(parentRel, "/")
	back := "/browse"
	if parentRel != "" {
		back = "/browse/" + url.QueryEscape(parentRel)
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// DeleteFolder handles POST /delete_folder/<path>
func (h *ActionHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/delete_folder/")
	raw = strings.TrimPrefix(raw, "/")
	folder, _ := security.ValidatePath(raw)

	target := filepath.Join(h.BaseDir, security.ToOSPath(folder))
	if !security.IsSafePath(h.BaseDir, target) || target == h.BaseDir {
		http.Error(w, "Access denied", 403)
		return
	}
	if err := os.RemoveAll(target); err != nil {
		http.Error(w, "Delete failed: "+err.Error(), 500)
		return
	}

	parentRel := filepath.Dir(folder)
	if parentRel == "." || parentRel == "/" {
		parentRel = ""
	}
	back := "/browse"
	if parentRel != "" {
		back = "/browse/" + url.QueryEscape(parentRel)
	}
	w.WriteHeader(204)
	_ = back
}

// DeletePicture handles POST /delete_picture/<path>
func (h *ActionHandler) DeletePicture(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/delete_picture/")
	raw = strings.TrimPrefix(raw, "/")
	rel, _ := security.ValidatePath(raw)

	target := filepath.Join(h.BaseDir, security.ToOSPath(rel))
	if !security.IsSafePath(h.BaseDir, target) {
		http.Error(w, "Access denied", 403)
		return
	}
	if strings.Contains(target, security.LogoFolder) {
		http.Error(w, "Cannot delete logo", 403)
		return
	}

	// Remove associated thumbnails (legacy and new .thumbs/ naming)
	deleteVideoThumbnails(target)

	if err := os.Remove(target); err != nil {
		http.Error(w, "Delete failed: "+err.Error(), 500)
		return
	}
	w.WriteHeader(204)
}

// ToggleFavorite handles POST /toggle_favorite/<path>
func (h *ActionHandler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/toggle_favorite/")
	raw = strings.TrimPrefix(raw, "/")
	rel, _ := security.ValidatePath(raw)

	full := filepath.Join(h.BaseDir, security.ToOSPath(rel))
	if !security.IsSafePath(h.BaseDir, full) {
		http.Error(w, "Access denied", 403)
		return
	}
	if _, err := os.Stat(full); err != nil {
		http.Error(w, "File not found", 404)
		return
	}

	isFav, err := meta.ToggleFavorite(h.BaseDir, rel)
	if err != nil {
		http.Error(w, "Toggle failed: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if isFav {
		w.Write([]byte(`{"success":true,"favorited":true}`))
	} else {
		w.Write([]byte(`{"success":true,"favorited":false}`))
	}
}

// Upload handles POST /upload — accepts multipart form data with
// one or more files and a "folder" field indicating the destination.
func (h *ActionHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "Cannot parse upload: "+err.Error(), 400)
		return
	}

	folder := r.FormValue("folder")
	folder, _ = security.ValidatePath(folder)

	destDir := h.BaseDir
	if folder != "" {
		destDir = filepath.Join(h.BaseDir, security.ToOSPath(folder))
	}
	if !security.IsSafePath(h.BaseDir, destDir) {
		http.Error(w, "Access denied", 403)
		return
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		http.Error(w, "Cannot create destination: "+err.Error(), 500)
		return
	}

	files := r.MultipartForm.File["photo"]
	if len(files) == 0 {
		files = r.MultipartForm.File["file"]
	}
	if len(files) == 0 {
		http.Error(w, "No files uploaded", 400)
		return
	}

	prefix := meta.GenerateSearchablePrefix(folder)
	now := time.Now().Format("20060102_150405")

	for i, fh := range files {
		name := security.SanitizeFilename(fh.Filename)
		if name == "untitled" || name == "" {
			// no usable filename — generate one
			ext := filepath.Ext(fh.Filename)
			if ext == "" {
				ext = ".jpg"
			}
			name = fmt.Sprintf("upload_%s_%d%s", now, i+1, ext)
		}
		if !security.AllowedFile(name) {
			continue
		}

		// Unique-ify: prepend prefix + timestamp
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		finalName := fmt.Sprintf("%s%s_%s%s", prefix, now, base, ext)

		dst := filepath.Join(destDir, finalName)
		if !security.IsSafePath(h.BaseDir, dst) {
			continue
		}

		src, err := fh.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(dst)
		if err != nil {
			src.Close()
			continue
		}
		io.Copy(out, src)
		out.Close()
		src.Close()

		if security.IsVideoFile(finalName) {
			video.GenerateThumbnailBackground(dst)
		}
	}

	back := "/browse"
	if folder != "" {
		back = "/browse/" + url.QueryEscape(folder)
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// SetSort handles POST /set_sort/<folder>
func (h *ActionHandler) SetSort(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/set_sort/")
	raw = strings.TrimPrefix(raw, "/")
	folder, _ := security.ValidatePath(raw)

	target := h.BaseDir
	if folder != "" && folder != "ROOT_DIR" {
		target = filepath.Join(h.BaseDir, security.ToOSPath(folder))
	}
	if !security.IsSafePath(h.BaseDir, target) {
		http.Error(w, "Access denied", 403)
		return
	}
	_ = os.MkdirAll(target, 0755)

	sortValue := r.FormValue("sort")
	if sortValue != "Newest" && sortValue != "Oldest" && sortValue != "A-Z" {
		sortValue = "Newest"
	}
	m := meta.GetMeta(target)
	m.Sort = sortValue
	_ = meta.SaveMeta(target, m)

	back := "/browse"
	if folder != "" && folder != "ROOT_DIR" {
		back = "/browse/" + url.QueryEscape(folder)
	}
	log.Printf("SetSort %q -> %s", folder, sortValue)
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// deleteVideoThumbnails removes any thumbnail variants associated with
// a file. Handles both the legacy "_thumb.jpg" naming (same folder) and
// the new ".thumbs/<basename>.jpg" naming (hidden subfolder).
//
// Silently ignores missing files. Safe to call on any media path.
func deleteVideoThumbnails(fullPath string) {
	base := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	baseName := filepath.Base(base)

	// Legacy: same folder, _thumb.jpg suffix
	_ = os.Remove(base + "_thumb.jpg")

	// New: hidden .thumbs/ subfolder
	thumbsDir := filepath.Join(filepath.Dir(fullPath), ".thumbs")
	_ = os.Remove(filepath.Join(thumbsDir, baseName+".jpg"))
}

// RotatePicture handles POST /rotate_picture/<path>?degrees=90|180|270.
//
// Per decision in DECISIONS_OPEN.md (Rotate + EXIF Interaction):
//   1. Read existing EXIF orientation.
//   2. Decode pixels.
//   3. Apply orientation correction (so image is visually upright).
//   4. Apply user's requested rotation.
//   5. Re-encode as JPEG (imaging.Save strips metadata).
//   6. Write via temp + atomic rename.
//   7. Delete any stale thumbnails (both namings).
//
// After rotation, pixel orientation is the source of truth.
func (h *ActionHandler) RotatePicture(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/rotate_picture/")
	raw = strings.TrimPrefix(raw, "/")
	rel, _ := security.ValidatePath(raw)

	degreesStr := r.URL.Query().Get("degrees")
	degrees, err := strconv.Atoi(degreesStr)
	if err != nil || (degrees != 90 && degrees != 180 && degrees != 270) {
		http.Error(w, "degrees must be 90, 180, or 270", 400)
		return
	}

	target := filepath.Join(h.BaseDir, security.ToOSPath(rel))
	if !security.IsSafePath(h.BaseDir, target) {
		http.Error(w, "Access denied", 403)
		return
	}
	if strings.Contains(target, security.LogoFolder) {
		http.Error(w, "Cannot rotate logo", 403)
		return
	}

	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found", 404)
		return
	}

	isJpeg := strings.HasSuffix(strings.ToLower(target), ".jpg") || strings.HasSuffix(strings.ToLower(target), ".jpeg")

	var img image.Image
	if isJpeg {
		orientation := readExifOrientation(target)
		src, err := imaging.Open(target, imaging.AutoOrientation(false))
		if err != nil {
			http.Error(w, "Cannot decode image: "+err.Error(), 500)
			return
		}
		img = applyExifRotation(src, orientation)
	} else {
		src, err := imaging.Open(target)
		if err != nil {
			http.Error(w, "Cannot decode image: "+err.Error(), 500)
			return
		}
		img = src
	}

	switch degrees {
	case 90:
		img = imaging.Rotate90(img)
	case 180:
		img = imaging.Rotate180(img)
	case 270:
		img = imaging.Rotate270(img)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		http.Error(w, "Cannot encode rotated image: "+err.Error(), 500)
		return
	}

	tmpPath := target + ".rotatetmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), info.Mode()); err != nil {
		_ = os.Remove(tmpPath)
		http.Error(w, "Cannot write temp file: "+err.Error(), 500)
		return
	}
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		http.Error(w, "Cannot finalize rotate: "+err.Error(), 500)
		return
	}

	deleteVideoThumbnails(target)

	log.Printf("RotatePicture: %s by %d degrees", target, degrees)
	w.WriteHeader(204)
}

// readExifOrientation returns the EXIF orientation tag (1-8), defaulting to 1.
func readExifOrientation(path string) int {
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

// applyExifRotation applies orientation correction to img (same mapping as /view).
func applyExifRotation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 3:
		return imaging.Rotate180(img)
	case 6:
		return imaging.Rotate270(img)
	case 8:
		return imaging.Rotate90(img)
	default:
		return img
	}
}

