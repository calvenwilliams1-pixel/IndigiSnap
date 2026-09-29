package video

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/paths"
)

// ffmpegAvailable caches whether ffmpeg/ffprobe are on PATH.
var ffmpegAvailable *bool

// HasFFmpeg returns true if ffprobe is available on PATH.
func HasFFmpeg() bool {
	if ffmpegAvailable != nil {
		return *ffmpegAvailable
	}
	_, err := exec.LookPath("ffprobe")
	available := err == nil
	ffmpegAvailable = &available
	if !available {
		log.Printf("ffprobe not found on PATH; video durations and thumbnails will be skipped")
	}
	return available
}

// GetDuration returns the duration of a video in seconds, or 0 if unavailable.
func GetDuration(videoPath string) float64 {
	if !HasFFmpeg() {
		return 0
	}
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("ffprobe failed for %s: %v", videoPath, err)
		return 0
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0
	}
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return d
}

// GenerateThumbnail creates a thumbnail in ThumbDir for the given video.
// Returns true if a thumbnail now exists (either pre-existing or newly created).
//
// Thumbnails live in {ThumbDir}/<relVideoPath>.jpg, mirroring the media tree.
// The full original filename is kept so clip.mp4 and clip.mov do not collide.
func GenerateThumbnail(videoPath string) bool {
	if !HasFFmpeg() {
		return false
	}

	rel, err := paths.RelOf(videoPath)
	if err != nil {
		log.Printf("thumb: relOf(%s) failed: %v", videoPath, err)
		return false
	}
	thumbPath := paths.ThumbFile(rel)
	if thumbPath == "" {
		return false
	}

	if _, err := os.Stat(thumbPath); err == nil {
		return true
	}

	// Ensure the thumbnail directory exists (mirror tree).
	if err := os.MkdirAll(filepath.Dir(thumbPath), 0755); err != nil {
		log.Printf("thumb: cannot create %s: %v", filepath.Dir(thumbPath), err)
		return false
	}

	seek := 1.0
	if d := GetDuration(videoPath); d > 2 {
		seek = d * 0.1
	}

	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-ss", strconv.FormatFloat(seek, 'f', 2, 64),
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-q:v", "2",
		thumbPath,
		"-y",
	)
	if err := cmd.Run(); err != nil {
		log.Printf("ffmpeg thumbnail failed for %s: %v", videoPath, err)
		return false
	}
	return true
}

// GenerateThumbnailBackground kicks off thumbnail generation in a goroutine.
func GenerateThumbnailBackground(videoPath string) {
	go func() {
		GenerateThumbnail(videoPath)
	}()
}

// CleanOrphanThumbnails removes orphaned thumbnail files in ThumbDir whose
// source video no longer exists in the corresponding media folder.
//
// Thumbnails live at {ThumbDir}/<relVideoPath>.jpg. For each thumbnail in
// the current folder's subdirectory of ThumbDir, verify the source video
// exists under BaseDir. Delete thumbnails whose source is gone.
//
// Also removes empty subdirectories left behind.
func CleanOrphanThumbnails(mediaFolderPath string) {
	rel, err := paths.RelOf(mediaFolderPath)
	if err != nil {
		return
	}
	thumbFolder := filepath.Join(paths.ThumbDir(), filepath.FromSlash(rel))
	entries, err := os.ReadDir(thumbFolder)
	if err != nil {
		return
	}
	remaining := 0
	for _, e := range entries {
		if e.IsDir() {
			remaining++
			continue
		}
		name := e.Name()
		// Thumbnails are "<videoFilename>.<ext>.jpg".
		// Strip trailing .jpg to get the video filename.
		if !strings.HasSuffix(name, ".jpg") {
			remaining++
			continue
		}
		videoName := strings.TrimSuffix(name, ".jpg")
		srcVideo := filepath.Join(mediaFolderPath, videoName)
		if _, err := os.Stat(srcVideo); err != nil {
			_ = os.Remove(filepath.Join(thumbFolder, name))
			log.Printf("Removed orphan thumbnail: %s", name)
		} else {
			remaining++
		}
	}
	// Remove empty thumbnail subfolder if applicable (keep ThumbDir root).
	if remaining == 0 && thumbFolder != paths.ThumbDir() {
		_ = os.Remove(thumbFolder)
	}
}
