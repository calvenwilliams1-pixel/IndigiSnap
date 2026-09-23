package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// DupesHandler serves GET /find_duplicates/<path>
// Walks the folder recursively, hashes every file, returns groups of matches.
type DupesHandler struct {
	BaseDir string
}

func NewDupesHandler(baseDir string) *DupesHandler {
	return &DupesHandler{BaseDir: baseDir}
}

func (h *DupesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/find_duplicates/")
	raw = strings.TrimPrefix(raw, "/")
	folder, _ := security.ValidatePath(raw)

	base := h.BaseDir
	if folder != "" {
		base = filepath.Join(h.BaseDir, security.ToOSPath(folder))
	}
	if !security.IsSafePath(h.BaseDir, base) {
		writeJSONError(w, "Access denied", 403)
		return
	}

	hashes := map[string][]string{}
	var total int

	filepath.Walk(base, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		name := fi.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if !security.AllowedFile(name) {
			return nil
		}
		sum, err := hashFile(p)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(h.BaseDir, p)
		hashes[sum] = append(hashes[sum], filepath.ToSlash(rel))
		total++
		return nil
	})

	groups := [][]string{}
	for _, files := range hashes {
		if len(files) > 1 {
			groups = append(groups, files)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"duplicates":        groups,
		"total_files":       total,
		"duplicate_groups":  len(groups),
	})
}

func hashFile(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
