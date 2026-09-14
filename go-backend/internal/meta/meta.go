package meta

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// FolderMeta mirrors the Python .indigisnap_meta.json schema.
type FolderMeta struct {
	Thumb        *string  `json:"thumb"`
	ActivePrefix string   `json:"active_prefix"`
	History      []string `json:"history"`
	Sort         string   `json:"sort"`
	Theme        string   `json:"theme"`
	Previews     []string `json:"previews"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	HideFromApp  bool     `json:"hide_from_app"`
}

// Breadcrumb is a (label, url) pair for the UI.
type Breadcrumb struct {
	Label string
	URL   string
}

const metaFilename = ".indigisnap_meta.json"
const favoritesFilename = ".indigisnap_favorites.json"

// DefaultMeta returns a fresh metadata struct with sensible defaults.
func DefaultMeta() FolderMeta {
	now := time.Now().Format(time.RFC3339)
	return FolderMeta{
		Thumb:        nil,
		ActivePrefix: "",
		History:      []string{},
		Sort:         "Newest",
		Theme:        "psychedelic",
		Previews:     []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
		HideFromApp:  false,
	}
}

// GetMeta reads folder metadata, filling in defaults for missing fields.
func GetMeta(folderPath string) FolderMeta {
	path := filepath.Join(folderPath, metaFilename)
	defaults := DefaultMeta()

	data, err := os.ReadFile(path)
	if err != nil {
		return defaults
	}

	var m FolderMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return defaults
	}

	if m.History == nil {
		m.History = defaults.History
	}
	if m.Previews == nil {
		m.Previews = defaults.Previews
	}
	if m.Sort == "" {
		m.Sort = defaults.Sort
	}
	if m.Theme == "" {
		m.Theme = defaults.Theme
	}
	if m.CreatedAt == "" {
		m.CreatedAt = defaults.CreatedAt
	}
	if m.UpdatedAt == "" {
		m.UpdatedAt = defaults.UpdatedAt
	}
	return m
}

// SaveMeta writes folder metadata to disk.
func SaveMeta(folderPath string, m FolderMeta) error {
	m.UpdatedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(folderPath, metaFilename), data, 0644)
}

// LoadFavorites reads the global favorites list from BASE_DIR.
func LoadFavorites(baseDir string) []string {
	path := filepath.Join(baseDir, favoritesFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{}
	}
	var favs []string
	if err := json.Unmarshal(data, &favs); err != nil {
		return []string{}
	}
	if favs == nil {
		return []string{}
	}
	return favs
}

// SaveFavorites writes the favorites list to disk.
func SaveFavorites(baseDir string, favs []string) error {
	data, err := json.MarshalIndent(favs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(baseDir, favoritesFilename), data, 0644)
}

// ToggleFavorite flips the favorite state of a relative path and returns
// the new state.
func ToggleFavorite(baseDir, relPath string) (bool, error) {
	favs := LoadFavorites(baseDir)
	found := false
	newFavs := make([]string, 0, len(favs)+1)
	for _, f := range favs {
		if f == relPath {
			found = true
			continue
		}
		newFavs = append(newFavs, f)
	}
	if !found {
		newFavs = append(newFavs, relPath)
	}
	if err := SaveFavorites(baseDir, newFavs); err != nil {
		return false, err
	}
	return !found, nil
}

// CleanupFavorites removes favorites that no longer exist on disk.
func CleanupFavorites(baseDir string) []string {
	favs := LoadFavorites(baseDir)
	kept := make([]string, 0, len(favs))
	for _, rel := range favs {
		full := filepath.Join(baseDir, filepath.FromSlash(rel))
		if _, err := os.Stat(full); err == nil {
			kept = append(kept, rel)
		}
	}
	if len(kept) != len(favs) {
		_ = SaveFavorites(baseDir, kept)
	}
	return kept
}

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9_]`)
var vowels = regexp.MustCompile(`[aeiouAEIOU]`)

// GenerateSearchablePrefix builds a short uppercase prefix from a folder route,
// used for uploaded filenames.
func GenerateSearchablePrefix(folderRoute string) string {
	if folderRoute == "" {
		return "root"
	}
	decoded, err := url.QueryUnescape(folderRoute)
	if err != nil {
		decoded = folderRoute
	}
	parts := []string{}
	for _, p := range strings.Split(decoded, "/") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "root"
	}
	cleaned := make([]string, 0, len(parts))
	for i, part := range parts {
		c := nonAlnum.ReplaceAllString(part, "")
		if c == "" {
			c = "folder"
		}
		if i < len(parts)-1 {
			if len(c) > 4 {
				condensed := vowels.ReplaceAllString(c, "")
				if len(condensed) >= 3 {
					c = condensed[:4]
				} else {
					c = c[:3]
				}
			}
			cleaned = append(cleaned, strings.ToUpper(c))
		} else {
			cleaned = append(cleaned, c)
		}
	}
	return strings.Join(cleaned, "_")
}

// GetBreadcrumbs produces navigation crumbs for a folder path.
func GetBreadcrumbs(folder string) []Breadcrumb {
	if folder == "" {
		return []Breadcrumb{{Label: "🏠", URL: "/browse"}}
	}
	if folder == "favorites" {
		return []Breadcrumb{
			{Label: "🏠", URL: "/browse"},
			{Label: "⭐ Favorites", URL: "/browse/favorites"},
		}
	}
	decoded, err := url.QueryUnescape(folder)
	if err != nil {
		decoded = folder
	}
	crumbs := []Breadcrumb{{Label: "🏠", URL: "/browse"}}
	current := []string{}
	for _, p := range strings.Split(decoded, "/") {
		if p == "" {
			continue
		}
		current = append(current, p)
		urlPath := strings.Join(current, "/")
		crumbs = append(crumbs, Breadcrumb{
			Label: p,
			URL:   "/browse/" + url.QueryEscape(urlPath),
		})
	}
	return crumbs
}

// RemoveFromRecents filters a path out of a recents list.
func RemoveFromRecents(recents []map[string]string, folderPath string) []map[string]string {
	out := make([]map[string]string, 0, len(recents))
	for _, r := range recents {
		if r["path"] != folderPath {
			out = append(out, r)
		}
	}
	return out
}

// ErrNotImplemented is a placeholder for routes we have not yet ported.
var ErrNotImplemented = errors.New("not implemented")
