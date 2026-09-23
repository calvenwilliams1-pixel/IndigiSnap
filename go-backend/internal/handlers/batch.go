package handlers

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// BatchHandler handles POST /batch_delete, /batch_move, /batch_rename,
// POST /export_zip, and /export_folder_zip.
type BatchHandler struct {
	BaseDir string
}

func NewBatchHandler(baseDir string) *BatchHandler {
	return &BatchHandler{BaseDir: baseDir}
}

func parseItems(r *http.Request) (string, []string) {
	_ = r.ParseForm()
	folder := r.FormValue("folder")
	folder, _ = security.ValidatePath(folder)
	var items []string
	raw := r.FormValue("folders")
	if raw == "" {
		return folder, items
	}
	_ = json.Unmarshal([]byte(raw), &items)
	return folder, items
}

func (h *BatchHandler) resolveBase(folder string) (string, bool) {
	base := h.BaseDir
	if folder != "" {
		base = filepath.Join(h.BaseDir, security.ToOSPath(folder))
	}
	if !security.IsSafePath(h.BaseDir, base) {
		return "", false
	}
	return base, true
}

// BatchDelete — POST /batch_delete
func (h *BatchHandler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	folder, items := parseItems(r)
	base, ok := h.resolveBase(folder)
	if !ok {
		http.Error(w, "Access denied", 403)
		return
	}
	for _, name := range items {
		name = security.SanitizeFilename(name)
		target := filepath.Join(base, name)
		if !security.IsSafePath(h.BaseDir, target) {
			continue
		}
		if security.IsHiddenFolder(name) {
			continue
		}
		if info, err := os.Stat(target); err == nil {
			if info.IsDir() {
				os.RemoveAll(target)
			} else {
				// remove thumb
				b := strings.TrimSuffix(target, filepath.Ext(target))
				os.Remove(b + "_thumb.jpg")
				os.Remove(target)
			}
		}
	}
	w.WriteHeader(204)
}

// BatchMove — POST /batch_move
func (h *BatchHandler) BatchMove(w http.ResponseWriter, r *http.Request) {
	folder, items := parseItems(r)
	base, ok := h.resolveBase(folder)
	if !ok {
		http.Error(w, "Access denied", 403)
		return
	}
	target := r.FormValue("target")
	target, _ = security.ValidatePath(target)
	dest := h.BaseDir
	if target != "" {
		dest = filepath.Join(h.BaseDir, security.ToOSPath(target))
	}
	if !security.IsSafePath(h.BaseDir, dest) {
		http.Error(w, "Access denied", 403)
		return
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		http.Error(w, "Cannot create destination", 500)
		return
	}
	for _, name := range items {
		name = security.SanitizeFilename(name)
		src := filepath.Join(base, name)
		dst := filepath.Join(dest, name)
		if !security.IsSafePath(h.BaseDir, src) || !security.IsSafePath(h.BaseDir, dst) {
			continue
		}
		if security.IsHiddenFolder(name) {
			continue
		}
		if _, err := os.Stat(src); err == nil {
			os.Rename(src, dst)
		}
	}
	w.WriteHeader(204)
}

// BatchRename — POST /batch_rename
// Pattern uses {n} for numbering, {name} for original base name.
func (h *BatchHandler) BatchRename(w http.ResponseWriter, r *http.Request) {
	folder, items := parseItems(r)
	base, ok := h.resolveBase(folder)
	if !ok {
		http.Error(w, "Access denied", 403)
		return
	}
	pattern := r.FormValue("pattern")
	if pattern == "" {
		http.Error(w, "Pattern required", 400)
		return
	}
	for i, name := range items {
		name = security.SanitizeFilename(name)
		src := filepath.Join(base, name)
		if !security.IsSafePath(h.BaseDir, src) {
			continue
		}
		if security.IsHiddenFolder(name) {
			continue
		}
		ext := filepath.Ext(name)
		origBase := strings.TrimSuffix(name, ext)
		newName := strings.ReplaceAll(pattern, "{n}", itoa(i+1))
		newName = strings.ReplaceAll(newName, "{name}", origBase)
		if ext != "" && !strings.HasSuffix(newName, ext) {
			newName += ext
		}
		newName = security.SanitizeFilename(newName)
		dst := filepath.Join(base, newName)
		if !security.IsSafePath(h.BaseDir, dst) {
			continue
		}
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		os.Rename(src, dst)
	}
	w.WriteHeader(204)
}

// ExportZip — POST /export_zip (selected items as ZIP)
func (h *BatchHandler) ExportZip(w http.ResponseWriter, r *http.Request) {
	folder, items := parseItems(r)
	base, ok := h.resolveBase(folder)
	if !ok {
		http.Error(w, "Access denied", 403)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="IndigiSnap_Export.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, name := range items {
		name = security.SanitizeFilename(name)
		full := filepath.Join(base, name)
		if !security.IsSafePath(h.BaseDir, full) || security.IsHiddenFolder(name) {
			continue
		}
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.IsDir() {
			filepath.Walk(full, func(p string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(base, p)
				w, err := zw.Create(filepath.ToSlash(rel))
				if err != nil {
					return nil
				}
				f, err := os.Open(p)
				if err != nil {
					return nil
				}
				defer f.Close()
				io.Copy(w, f)
				return nil
			})
		} else {
			f, err := os.Open(full)
			if err != nil {
				continue
			}
			w, err := zw.Create(name)
			if err != nil {
				f.Close()
				continue
			}
			io.Copy(w, f)
			f.Close()
		}
	}
}

// ExportFolderZip — POST /export_folder_zip/<path>
func (h *BatchHandler) ExportFolderZip(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/export_folder_zip/")
	raw = strings.TrimPrefix(raw, "/")
	folder, _ := security.ValidatePath(raw)
	if folder == "" {
		http.Error(w, "Folder required", 400)
		return
	}
	full := filepath.Join(h.BaseDir, security.ToOSPath(folder))
	if !security.IsSafePath(h.BaseDir, full) {
		http.Error(w, "Access denied", 403)
		return
	}
	if _, err := os.Stat(full); err != nil {
		http.Error(w, "Folder not found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="IndigiSnap_Folder.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	filepath.Walk(full, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(full, p)
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		io.Copy(w, f)
		return nil
	})
}
