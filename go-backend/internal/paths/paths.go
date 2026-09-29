package paths

import (
	"errors"
	"log"
	"path/filepath"
	"strings"
	"sync"
)

// Package paths holds the three root directories the Go backend operates on:
//
//   BaseDir  - media root (user-visible). e.g. {Pictures}/IndigiSnap
//   MetaDir  - metadata root (app-private). e.g. {filesDir}
//   ThumbDir - generated thumbnails (app-cache). e.g. {cacheDir}/indigisnap-thumbs
//
// The three are set once at server start via Init. Subsequent Init calls
// with different paths log a warning and are ignored (the server is
// single-instance per process).

var (
	once     sync.Once
	baseDir  string
	metaDir  string
	thumbDir string
)

// ErrOutsideBase is returned by RelOf when the given path is not inside
// BaseDir.
var ErrOutsideBase = errors.New("paths: path outside base dir")

// Init sets the three root directories. Only the first call takes effect.
// Subsequent calls with different paths are logged and ignored.
func Init(base, meta, thumb string) {
	base = filepath.Clean(base)
	meta = filepath.Clean(meta)
	thumb = filepath.Clean(thumb)

	once.Do(func() {
		baseDir = base
		metaDir = meta
		thumbDir = thumb
		log.Printf("paths: base=%s meta=%s thumb=%s", base, meta, thumb)
	})

	if baseDir != base || metaDir != meta || thumbDir != thumb {
		log.Printf("paths: ignoring re-init with different paths (base=%s meta=%s thumb=%s)", base, meta, thumb)
	}
}

// BaseDir returns the media root.
func BaseDir() string { return baseDir }

// MetaDir returns the metadata root.
func MetaDir() string { return metaDir }

// ThumbDir returns the thumbnail root.
func ThumbDir() string { return thumbDir }

// RelOf returns the slash-separated path of full relative to BaseDir.
// Returns "" for BaseDir itself. Returns an error if full is outside BaseDir.
func RelOf(full string) (string, error) {
	full = filepath.Clean(full)
	rel, err := filepath.Rel(baseDir, full)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return "", nil
	}
	if strings.HasPrefix(rel, "..") {
		return "", ErrOutsideBase
	}
	return filepath.ToSlash(rel), nil
}

// splitRel splits a slash-separated rel path into segments, dropping
// empty, "." and ".." segments. This prevents crafted folder names from
// escaping MetaDir or ThumbDir.
func splitRel(rel string) []string {
	if rel == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(rel, "/") {
		if p == "" || p == "." || p == ".." {
			continue
		}
		out = append(out, p)
	}
	return out
}

// MetaFile returns the mirrored metadata path for a folder.
// The root folder maps to {metaDir}/meta/_indigisnap_root.json.
// A folder "a/b" maps to {metaDir}/meta/a/b/_indigisnap_meta.json.
//
// The "_indigisnap_" prefix on the metadata filename avoids collision
// with user files named "_meta.json" or similar.
func MetaFile(relFolder string) string {
	parts := splitRel(relFolder)
	if len(parts) == 0 {
		return filepath.Join(metaDir, "meta", "_indigisnap_root.json")
	}
	dir := filepath.Join(append([]string{metaDir, "meta"}, parts...)...)
	return filepath.Join(dir, "_indigisnap_meta.json")
}

// MetaDirFor returns the mirrored metadata directory for a folder.
// Used by rename/delete operations that need the whole subtree.
func MetaDirFor(relFolder string) string {
	parts := splitRel(relFolder)
	if len(parts) == 0 {
		return filepath.Join(metaDir, "meta")
	}
	return filepath.Join(append([]string{metaDir, "meta"}, parts...)...)
}

// ThumbFile returns the mirrored thumbnail path for a video.
// The full original filename is kept, with the extension replaced by ".jpg".
// e.g. "Tokyo day 2/clip.mp4" -> {thumbDir}/Tokyo day 2/clip.mp4.jpg
// Keeps the extension in the name so clip.mp4 and clip.mov do not collide.
func ThumbFile(relVideo string) string {
	parts := splitRel(relVideo)
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	thumbName := last + ".jpg"
	dir := filepath.Join(append([]string{thumbDir}, parts[:len(parts)-1]...)...)
	return filepath.Join(dir, thumbName)
}

// Global metadata files, all under MetaDir root.

// FavoritesFile returns the path to the favorites JSON.
func FavoritesFile() string {
	return filepath.Join(metaDir, "favorites.json")
}

// RecentsFile returns the path to the recents JSON.
func RecentsFile() string {
	return filepath.Join(metaDir, "recents.json")
}

// HiddenFile returns the path to the hidden index JSON.
func HiddenFile() string {
	return filepath.Join(metaDir, "hidden.json")
}

// LogoDir returns the directory holding the custom logo.
func LogoDir() string {
	return filepath.Join(metaDir, "logo")
}
