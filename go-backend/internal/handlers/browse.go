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
	"github.com/calvenwilliams1-pixel/indigisnap/internal/ui"
)

// BrowseHandler serves GET /browse and GET /browse/<path>.
type BrowseHandler struct {
	BaseDir string
}

func NewBrowseHandler(baseDir string) *BrowseHandler {
	return &BrowseHandler{BaseDir: baseDir}
}

func (h *BrowseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/browse")
	rawPath = strings.TrimPrefix(rawPath, "/")

	folder, err := security.ValidatePath(rawPath)
	if err != nil {
		http.Error(w, "Invalid path: "+err.Error(), 400)
		return
	}

	fullPath := filepath.Join(h.BaseDir, security.ToOSPath(folder))
	if !security.IsSafePath(h.BaseDir, fullPath) {
		http.Error(w, "Access denied", 403)
		return
	}

	// Create the folder if it doesn't exist yet (matches Flask behavior)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		log.Printf("MkdirAll failed for %s: %v", fullPath, err)
		http.Error(w, "Cannot create folder: "+err.Error(), 500)
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		log.Printf("ReadDir failed for %s: %v", fullPath, err)
		http.Error(w, "Cannot read folder", 500)
		return
	}

	folders := []ui.FolderItem{}
	media := []ui.MediaItem{}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if security.IsHiddenFolder(name) {
			continue
		}

		if entry.IsDir() {
			rel := name
			if folder != "" {
				rel = folder + "/" + name
			}
			folders = append(folders, ui.FolderItem{
				Name:     name,
				URL:      "/browse/" + url.QueryEscape(rel),
				Previews: []string{},
			})
			continue
		}

		if !security.AllowedFile(name) {
			continue
		}

		rel := name
		if folder != "" {
			rel = folder + "/" + name
		}

		isVid := security.IsVideoFile(name)
		mediaType := "image"
		if isVid {
			mediaType = "video"
		}

		media = append(media, ui.MediaItem{
			Name:       name,
			RelPath:    url.QueryEscape(rel),
			IsVideo:    isVid,
			Type:       mediaType,
			IsFavorite: false,
		})
	}

	log.Printf("Browse %q: %d folders, %d media", folder, len(folders), len(media))

	data := ui.TemplateData{
		Folder:      folder,
		Items:       folders,
		Images:      media,
		Meta:        ui.FolderMeta{Sort: "Newest"},
		Recents:     []ui.Recent{},
		Page:        1,
		TotalPages:  1,
		LogoURL:     "",
		SortBy:      "newest",
		FilterType:  "all",
		Breadcrumbs: toUIBreadcrumbs(meta.GetBreadcrumbs(folder)),
	}

	html, err := ui.Render(data)
	if err != nil {
		log.Printf("Template render error: %v", err)
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func toUIBreadcrumbs(in []meta.Breadcrumb) []ui.Breadcrumb {
	out := make([]ui.Breadcrumb, 0, len(in))
	for _, b := range in {
		out = append(out, ui.Breadcrumb{Label: b.Label, Path: b.URL})
	}
	return out
}
