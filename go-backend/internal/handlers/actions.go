package handlers

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/meta"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
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

	// Remove associated thumbnail if video
	base := strings.TrimSuffix(target, filepath.Ext(target))
	thumb := base + "_thumb.jpg"
	_ = os.Remove(thumb)

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
