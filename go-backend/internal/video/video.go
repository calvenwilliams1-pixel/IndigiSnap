package video

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// GenerateThumbnail creates a <base>_thumb.jpg next to the video.
// Returns true if a thumbnail now exists (either pre-existing or newly created).
func GenerateThumbnail(videoPath string) bool {
	if !HasFFmpeg() {
		return false
	}

	base := strings.TrimSuffix(videoPath, filepath.Ext(videoPath))
	thumbPath := base + "_thumb.jpg"

	if _, err := os.Stat(thumbPath); err == nil {
		return true
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

// CleanOrphanThumbnails removes orphaned thumbnail files whose video no
// longer exists. Handles two naming conventions:
//
//   Legacy: <basename>_thumb.jpg alongside the video (same folder)
//   New:    .thumbs/<basename>.jpg in a hidden subfolder
//
// Also removes the .thumbs/ folder if it becomes empty.
func CleanOrphanThumbnails(folderPath string) {
	cleanLegacyThumbs(folderPath)
	cleanHiddenThumbs(folderPath)
}

// cleanLegacyThumbs handles the <basename>_thumb.jpg naming.
func cleanLegacyThumbs(folderPath string) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, "_thumb.jpg") {
			continue
		}
		videoBase := strings.TrimSuffix(name, "_thumb.jpg")
		if !hasMatchingVideo(folderPath, videoBase) {
			_ = os.Remove(filepath.Join(folderPath, name))
			log.Printf("Removed orphan legacy thumbnail: %s", name)
		}
	}
}

// cleanHiddenThumbs handles the .thumbs/<basename>.jpg naming.
// Also removes the .thumbs/ folder if it becomes empty.
func cleanHiddenThumbs(folderPath string) {
	thumbsDir := filepath.Join(folderPath, ".thumbs")
	entries, err := os.ReadDir(thumbsDir)
	if err != nil {
		return
	}
	remaining := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".jpg") {
			remaining++
			continue
		}
		// Thumbnail name is <video_basename>.jpg
		videoBase := strings.TrimSuffix(name, ".jpg")
		if !hasMatchingVideo(folderPath, videoBase) {
			_ = os.Remove(filepath.Join(thumbsDir, name))
			log.Printf("Removed orphan hidden thumbnail: %s", name)
		} else {
			remaining++
		}
	}
	// Remove .thumbs/ folder if empty
	if remaining == 0 {
		_ = os.Remove(thumbsDir)
	}
}

// hasMatchingVideo reports whether a video with the given basename (without
// extension) exists in the folder, checking all known video extensions.
func hasMatchingVideo(folderPath, videoBase string) bool {
	for _, ext := range []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv", ".m4v"} {
		if _, err := os.Stat(filepath.Join(folderPath, videoBase+ext)); err == nil {
			return true
		}
	}
	return false
}
