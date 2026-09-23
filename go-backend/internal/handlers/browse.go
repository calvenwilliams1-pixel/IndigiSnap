package handlers

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/meta"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/ui"
)

type BrowseHandler struct {
	BaseDir string
}

func NewBrowseHandler(baseDir string) *BrowseHandler {
	return &BrowseHandler{BaseDir: baseDir}
}

type fileRecord struct {
	Name       string
	RelPath    string
	IsVideo    bool
	Size       int64
	ModTime    int64
	CreateTime int64
	IsFavorite bool
}

func (h *BrowseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/browse")
	rawPath = strings.TrimPrefix(rawPath, "/")

	// Virtual Favorites folder
	if rawPath == "favorites" {
		h.serveFavorites(w, r)
		return
	}

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

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		log.Printf("MkdirAll failed: %v", err)
		http.Error(w, "Cannot create folder", 500)
		return
	}

	sortBy := r.URL.Query().Get("sort_by")
	if sortBy == "" {
		sortBy = "newest"
	}
	filterType := r.URL.Query().Get("filter_type")
	if filterType == "" {
		filterType = "all"
	}

	folderMeta := meta.GetMeta(fullPath)
	favorites := meta.LoadFavorites(h.BaseDir)
	favSet := make(map[string]bool, len(favorites))
	for _, f := range favorites {
		favSet[f] = true
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, "Cannot read folder", 500)
		return
	}

	folders := []ui.FolderItem{}
	media := []fileRecord{}

	// Add Favorites virtual folder at root
	if folder == "" {
		folders = append(folders, ui.FolderItem{
			Name:     "⭐ Favorites",
			URL:      "/browse/favorites",
			Previews: []string{},
		})
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if security.IsHiddenFolder(name) {
			continue
		}

		fullEntryPath := filepath.Join(fullPath, name)

		if entry.IsDir() {
			rel := name
			if folder != "" {
				rel = folder + "/" + name
			}

			// Build previews (first 4 images inside)
			previews := []string{}
			subEntries, err := os.ReadDir(fullEntryPath)
			if err == nil {
				for _, se := range subEntries {
					if len(previews) >= 4 {
						break
					}
					sn := se.Name()
					if strings.HasPrefix(sn, ".") {
						continue
					}
					if se.IsDir() {
						continue
					}
					if !security.AllowedFile(sn) {
						continue
					}
					if security.IsVideoFile(sn) {
						continue
					}
					previewRel := rel + "/" + sn
					previews = append(previews, url.QueryEscape(previewRel))
				}
			}

			folders = append(folders, ui.FolderItem{
				Name:     name,
				URL:      "/browse/" + url.QueryEscape(rel),
				Previews: previews,
			})
			continue
		}

		if !security.AllowedFile(name) {
			continue
		}

		info, err := entry.Info()
		var size, modTime, ctime int64
		if err == nil {
			size = info.Size()
			modTime = info.ModTime().Unix()
			ctime = info.ModTime().Unix()
		}

		rel := name
		if folder != "" {
			rel = folder + "/" + name
		}

		media = append(media, fileRecord{
			Name:       name,
			RelPath:    rel,
			IsVideo:    security.IsVideoFile(name),
			Size:       size,
			ModTime:    modTime,
			CreateTime: ctime,
			IsFavorite: favSet[rel],
		})
	}

	// Filter
	switch filterType {
	case "images":
		filtered := media[:0]
		for _, m := range media {
			if !m.IsVideo {
				filtered = append(filtered, m)
			}
		}
		media = filtered
	case "videos":
		filtered := media[:0]
		for _, m := range media {
			if m.IsVideo {
				filtered = append(filtered, m)
			}
		}
		media = filtered
	case "favorites":
		filtered := media[:0]
		for _, m := range media {
			if m.IsFavorite {
				filtered = append(filtered, m)
			}
		}
		media = filtered
	}

	// Sort folders (A-Z, but keep Favorites at top)
	sort.SliceStable(folders, func(i, j int) bool {
		if strings.HasPrefix(folders[i].Name, "⭐") {
			return true
		}
		if strings.HasPrefix(folders[j].Name, "⭐") {
			return false
		}
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})

	// Sort media
	switch sortBy {
	case "name":
		sort.Slice(media, func(i, j int) bool {
			return strings.ToLower(media[i].Name) < strings.ToLower(media[j].Name)
		})
	case "size":
		sort.Slice(media, func(i, j int) bool {
			return media[i].Size > media[j].Size
		})
	case "oldest":
		sort.Slice(media, func(i, j int) bool {
			return media[i].CreateTime < media[j].CreateTime
		})
	default:
		sort.Slice(media, func(i, j int) bool {
			return media[i].CreateTime > media[j].CreateTime
		})
	}

	uiMedia := make([]ui.MediaItem, 0, len(media))
	for _, m := range media {
		mediaType := "image"
		if m.IsVideo {
			mediaType = "video"
		}
		uiMedia = append(uiMedia, ui.MediaItem{
			Name:       m.Name,
			RelPath:    url.QueryEscape(m.RelPath),
			IsVideo:    m.IsVideo,
			Type:       mediaType,
			Size:       m.Size,
			IsFavorite: m.IsFavorite,
		})
	}

	log.Printf("Browse %q: %d folders, %d media (sort=%s filter=%s)",
		folder, len(folders), len(uiMedia), sortBy, filterType)

	data := ui.TemplateData{
		Folder:      folder,
		Items:       folders,
		Images:      uiMedia,
		Meta:        ui.FolderMeta{Sort: folderMeta.Sort},
		Recents:     []ui.Recent{},
		Page:        1,
		TotalPages:  1,
		LogoURL:     "",
		SortBy:      sortBy,
		FilterType:  filterType,
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

// serveFavorites renders the virtual Favorites folder.
func (h *BrowseHandler) serveFavorites(w http.ResponseWriter, r *http.Request) {
	favorites := meta.CleanupFavorites(h.BaseDir)
	media := []ui.MediaItem{}

	for _, rel := range favorites {
		full := filepath.Join(h.BaseDir, filepath.FromSlash(rel))
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		name := info.Name()
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
			Size:       info.Size(),
			IsFavorite: true,
		})
	}

	data := ui.TemplateData{
		Folder:      "favorites",
		Items:       []ui.FolderItem{},
		Images:      media,
		Meta:        ui.FolderMeta{Sort: "Newest"},
		Recents:     []ui.Recent{},
		Page:        1,
		TotalPages:  1,
		LogoURL:     "",
		SortBy:      "newest",
		FilterType:  "all",
		Breadcrumbs: toUIBreadcrumbs(meta.GetBreadcrumbs("favorites")),
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
