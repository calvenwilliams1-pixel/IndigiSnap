package meta

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
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


// GetLogoPath returns the absolute path to the logo file in BASE_DIR/logo/,
// or "" if no suitable file exists.
//
// Priority:
//  1. First file whose basename (without extension) is exactly "logo"
//     (alphabetically first if multiple extensions exist)
//  2. Otherwise, first file alphabetically
//  3. Otherwise, ""
//
// Hidden files (leading dot) are always excluded.
//
// SVG is allowed because this is a single-user installation. Multi-user
// deployments may wish to restrict SVG uploads because SVG can contain
// active content.
func GetLogoPath(baseDir string) string {
	logoDir := filepath.Join(baseDir, "logo")
	entries, err := os.ReadDir(logoDir)
	if err != nil {
		return ""
	}

	allowed := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".webp": true, ".svg": true,
	}

	var logoMatches []string
	var others []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !allowed[ext] {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		if base == "logo" {
			logoMatches = append(logoMatches, name)
		} else {
			others = append(others, name)
		}
	}

	if len(logoMatches) > 0 {
		sort.Strings(logoMatches)
		return filepath.Join(logoDir, logoMatches[0])
	}
	if len(others) > 0 {
		sort.Strings(others)
		return filepath.Join(logoDir, others[0])
	}
	return ""
}

// GetLogoURL returns the URL path for the logo (/logo/<filename>), or ""
// if no logo file exists.
func GetLogoURL(baseDir string) string {
	path := GetLogoPath(baseDir)
	if path == "" {
		return ""
	}
	return "/logo/" + url.PathEscape(filepath.Base(path))
}


// RecentEntry is one item in the recents list.
type RecentEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

const recentsFilename = ".indigisnap_recents.json"

var recentsMu sync.Mutex

// LoadRecents returns the last-5 recents list, filtering stale entries.
// Stale = path no longer exists on disk.
//
// If the file is missing or malformed, returns an empty list.
// Transient stat errors (not IsNotExist) keep the entry.
func LoadRecents(baseDir string) []RecentEntry {
	path := filepath.Join(baseDir, recentsFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return []RecentEntry{}
	}
	var entries []RecentEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []RecentEntry{}
	}

	kept := make([]RecentEntry, 0, len(entries))
	for _, e := range entries {
		full := filepath.Join(baseDir, filepath.FromSlash(e.Path))
		if _, err := os.Stat(full); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			// transient error - keep the entry
			kept = append(kept, e)
			continue
		}
		kept = append(kept, e)
	}
	return kept
}

// SaveRecents writes recents to BASE_DIR/.indigisnap_recents.json
func SaveRecents(baseDir string, entries []RecentEntry) error {
	if entries == nil {
		entries = []RecentEntry{}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(baseDir, recentsFilename), data, 0644)
}

// AddRecent updates recents with a newly visited folder and returns the new list.
// Deduplicates by path, prepends, trims to 5, filters stale entries.
// Returns the current list unchanged if folder is "" (root) or "favorites".
func AddRecent(baseDir, folder string) []RecentEntry {
	recentsMu.Lock()
	defer recentsMu.Unlock()

	if folder == "" || folder == "favorites" {
		return LoadRecents(baseDir)
	}

	entries := LoadRecents(baseDir)

	// Remove existing entry with same path (dedupe)
	newEntries := make([]RecentEntry, 0, len(entries)+1)
	for _, e := range entries {
		if e.Path != folder {
			newEntries = append(newEntries, e)
		}
	}

	// Prepend new entry
	entry := RecentEntry{
		Name: filepath.Base(folder),
		Path: folder,
	}
	newEntries = append([]RecentEntry{entry}, newEntries...)

	// Trim to 5
	if len(newEntries) > 5 {
		newEntries = newEntries[:5]
	}

	_ = SaveRecents(baseDir, newEntries)
	return newEntries
}
