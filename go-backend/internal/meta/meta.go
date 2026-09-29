package meta

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/fsutil"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/paths"
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

// perFileLocks serializes read-modify-write sequences per file path.
// Without this, two concurrent ToggleFavorite or AddRecent calls can
// lose updates even with atomic writes.
var perFileLocks sync.Map

func withFileLock(path string, fn func() error) error {
	m, _ := perFileLocks.LoadOrStore(path, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

// relOf converts an absolute folder path into a slash-separated path
// relative to BaseDir. Returns "" on error (callers log and continue).
func relOf(full string) string {
	rel, err := paths.RelOf(full)
	if err != nil {
		log.Printf("meta: relOf(%q) failed: %v", full, err)
		return ""
	}
	return rel
}

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

// GetMeta reads folder metadata from the mirrored metadata tree.
// Fills in defaults for missing fields.
func GetMeta(folderPath string) FolderMeta {
	path := paths.MetaFile(relOf(folderPath))
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

// SaveMeta writes folder metadata to the mirrored metadata tree.
// Atomic write via fsutil.
func SaveMeta(folderPath string, m FolderMeta) error {
	m.UpdatedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(paths.MetaFile(relOf(folderPath)), data, 0644)
}

// LoadFavorites reads the global favorites list from MetaDir.
func LoadFavorites(_ string) []string {
	data, err := os.ReadFile(paths.FavoritesFile())
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

// SaveFavorites writes the favorites list to MetaDir atomically.
func SaveFavorites(_ string, favs []string) error {
	data, err := json.MarshalIndent(favs, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(paths.FavoritesFile(), data, 0644)
}

// ToggleFavorite flips the favorite state of a relative path and returns
// the new state. Serialized per-file to avoid lost updates.
func ToggleFavorite(_ string, relPath string) (bool, error) {
	var newState bool
	err := withFileLock(paths.FavoritesFile(), func() error {
		favs := LoadFavorites("")
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
		if err := SaveFavorites("", newFavs); err != nil {
			return err
		}
		newState = !found
		return nil
	})
	return newState, err
}

// CleanupFavorites removes favorites that no longer exist on disk.
func CleanupFavorites(_ string) []string {
	favs := LoadFavorites("")
	baseDir := paths.BaseDir()
	kept := make([]string, 0, len(favs))
	for _, rel := range favs {
		full := filepath.Join(baseDir, filepath.FromSlash(rel))
		if _, err := os.Stat(full); err == nil {
			kept = append(kept, rel)
		}
	}
	if len(kept) != len(favs) {
		_ = SaveFavorites("", kept)
	}
	return kept
}

// Reconcile prunes metadata entries whose target paths no longer exist.
// Called at the top of every browse. Self-healing: catches orphans from
// out-of-band changes (Samsung Files, system gallery, adb) that hooks
// would miss, and from crashes mid-operation.
//
// Prunes:
//   1. favorites.json entries whose file is gone
//   2. recents.json entries whose folder is gone
//   3. per-folder metadata for folders that no longer exist
//   4. folder meta "thumb" pointers to gone files (cleared, not removed)
//   5. folder meta "previews" entries pointing at gone files
func Reconcile() {
	reconcileFavorites()
	reconcileRecents()
	reconcileFolderMeta()
}

// reconcileFavorites drops favorites whose file no longer exists.
func reconcileFavorites() {
	favs := LoadFavorites("")
	if len(favs) == 0 {
		return
	}
	baseDir := paths.BaseDir()
	kept := make([]string, 0, len(favs))
	for _, rel := range favs {
		full := filepath.Join(baseDir, filepath.FromSlash(rel))
		if _, err := os.Stat(full); err == nil {
			kept = append(kept, rel)
		}
	}
	if len(kept) != len(favs) {
		_ = SaveFavorites("", kept)
		log.Printf("reconcile: favorites %d -> %d", len(favs), len(kept))
	}
}

// reconcileRecents drops recents whose folder no longer exists.
func reconcileRecents() {
	recentsMu.Lock()
	defer recentsMu.Unlock()

	entries := LoadRecents("")
	if len(entries) == 0 {
		return
	}
	baseDir := paths.BaseDir()
	kept := make([]RecentEntry, 0, len(entries))
	for _, e := range entries {
		full := filepath.Join(baseDir, filepath.FromSlash(e.Path))
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			kept = append(kept, e)
		}
	}
	if len(kept) != len(entries) {
		_ = SaveRecents("", kept)
		log.Printf("reconcile: recents %d -> %d", len(entries), len(kept))
	}
}

// reconcileFolderMeta walks the mirror-tree metadata root and removes
// metadata for folders that no longer exist under BaseDir. It also
// prunes stale "thumb" and "previews" pointers within surviving folders.
func reconcileFolderMeta() {
	metaRoot := filepath.Join(paths.MetaDir(), "meta")
	if _, err := os.Stat(metaRoot); err != nil {
		return
	}

	baseDir := paths.BaseDir()
	removed := 0
	prunedPtrs := 0

	// Walk the metadata tree. For each _indigisnap_meta.json file, derive
	// the corresponding media folder path and check existence.
	_ = filepath.Walk(metaRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() != "_indigisnap_meta.json" {
			return nil
		}

		// Derive the media folder this metadata belongs to.
		// {MetaDir}/meta/a/b/_indigisnap_meta.json -> a/b
		rel, err := filepath.Rel(metaRoot, filepath.Dir(path))
		if err != nil {
			return nil
		}
		if rel == "." {
			rel = ""
		}
		var mediaFolder string
		if rel == "" {
			mediaFolder = baseDir
		} else {
			mediaFolder = filepath.Join(baseDir, rel)
		}

		// If the media folder no longer exists, delete the metadata file
		// and any empty directories left behind.
		if _, err := os.Stat(mediaFolder); os.IsNotExist(err) {
			_ = os.Remove(path)
			removed++
			// Prune empty parent directories up to metaRoot
			dir := filepath.Dir(path)
			for dir != metaRoot {
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) > 0 {
					break
				}
				_ = os.Remove(dir)
				dir = filepath.Dir(dir)
			}
			return nil
		}

		// Media folder exists: check thumb and previews pointers.
		m := GetMeta(mediaFolder)
		changed := false
		if m.Thumb != nil && *m.Thumb != "" {
			thumbRel := filepath.FromSlash(*m.Thumb)
			if _, err := os.Stat(filepath.Join(mediaFolder, thumbRel)); err != nil {
				m.Thumb = nil
				changed = true
			}
		}
		if len(m.Previews) > 0 {
			kept := make([]string, 0, len(m.Previews))
			for _, p := range m.Previews {
				// Previews may be URL-encoded (e.g. "logo%2F...")
				decoded, err := url.QueryUnescape(p)
				if err != nil {
					decoded = p
				}
				decoded = filepath.FromSlash(decoded)
				if _, err := os.Stat(filepath.Join(mediaFolder, decoded)); err == nil {
					kept = append(kept, p)
				}
			}
			removedCount := len(m.Previews) - len(kept)
			if removedCount > 0 {
				m.Previews = kept
				changed = true
				prunedPtrs += removedCount
			}
		}
		if changed {
			_ = SaveMeta(mediaFolder, m)
		}
		return nil
	})

	if removed > 0 || prunedPtrs > 0 {
		log.Printf("reconcile: folder meta removed=%d pruned_ptrs=%d", removed, prunedPtrs)
	}
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
func GetLogoPath(_ string) string {
	logoDir := paths.LogoDir()
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
func GetLogoURL(_ string) string {
	path := GetLogoPath("")
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

var recentsMu sync.Mutex

// LoadRecents returns the last-5 recents list, filtering stale entries.
// Stale = path no longer exists on disk.
//
// If the file is missing or malformed, returns an empty list.
// Transient stat errors (not IsNotExist) keep the entry.
func LoadRecents(_ string) []RecentEntry {
	data, err := os.ReadFile(paths.RecentsFile())
	if err != nil {
		return []RecentEntry{}
	}
	var entries []RecentEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []RecentEntry{}
	}

	baseDir := paths.BaseDir()
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
func SaveRecents(_ string, entries []RecentEntry) error {
	if entries == nil {
		entries = []RecentEntry{}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(paths.RecentsFile(), data, 0644)
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
