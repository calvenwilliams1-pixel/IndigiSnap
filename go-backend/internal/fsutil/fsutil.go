package fsutil

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path atomically: it writes to a temp file
// in the same directory, fsyncs, and renames over the target. This prevents
// readers from seeing a partially-written file and prevents corruption on
// crash during write.
//
// NOTE: this helper writes to app-private storage (MetaDir, cacheDir).
// It uses temp filenames starting with ".tmp_" which are NOT valid in
// scoped-storage media folders like Pictures/. Do not use this for media.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp_"+filepath.Base(path)+"_*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup; ignored if rename succeeds.
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
