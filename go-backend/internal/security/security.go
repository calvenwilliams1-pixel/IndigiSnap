package security

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AllowedExtensions is the set of file extensions the app will serve/accept.
var AllowedExtensions = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true, "webp": true, "svg": true,
	"mp4": true, "mov": true, "avi": true, "mkv": true, "webm": true,
	"flv": true, "wmv": true, "m4v": true,
}

// VideoExtensions is the subset that are treated as video files.
var VideoExtensions = map[string]bool{
	"mp4": true, "mov": true, "avi": true, "mkv": true, "webm": true,
	"flv": true, "wmv": true, "m4v": true,
}

// LogoFolder is the reserved name for the app's branding folder.
const LogoFolder = "logo"

// IsSafePath returns true if target is inside or equal to base.
// Prevents path traversal outside the base directory.
func IsSafePath(base, target string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	// Ensure separator boundary so /foo does not match /foobar
	if absTarget == absBase {
		return true
	}
	return strings.HasPrefix(absTarget, absBase+string(os.PathSeparator))
}

// ValidatePath URL-decodes and sanitizes a path from the request.
// Returns the cleaned relative path.
func ValidatePath(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}
	// Remove any ../ sequences
	traversal := regexp.MustCompile(`\.\.(?:/|\|$)`)
	decoded = traversal.ReplaceAllString(decoded, "")
	decoded = strings.TrimLeft(decoded, "/")
	if decoded == "" {
		return "", nil
	}
	return decoded, nil
}

// ToOSPath converts a URL-style path (slashes) to an OS path.
func ToOSPath(urlPath string) string {
	if urlPath == "" {
		return ""
	}
	decoded, err := url.QueryUnescape(urlPath)
	if err != nil {
		decoded = urlPath
	}
	return filepath.FromSlash(decoded)
}

// AllowedFile returns true if the filename has an allowed extension.
func AllowedFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	return AllowedExtensions[strings.TrimPrefix(ext, ".")]
}

// IsVideoFile returns true if the filename has a video extension.
func IsVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	return VideoExtensions[strings.TrimPrefix(ext, ".")]
}

var illegalFilenameChars = regexp.MustCompile(`[<>:"/\\|?*]`)

// SanitizeFilename strips illegal characters and trims dangerous edges.
func SanitizeFilename(name string) string {
	if name == "" {
		return "untitled"
	}
	name = illegalFilenameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, ". ")
	if name == "" {
		return "untitled"
	}
	// Enforce max length, preserving extension
	if len(name) > 255 {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(base) > 245 {
			base = base[:245]
		}
		name = base + ext
	}
	return name
}

// IsHiddenFolder returns true if the folder should be hidden from the UI.
func IsHiddenFolder(name string) bool {
	if name == "" {
		return false
	}
	return name == LogoFolder || strings.HasPrefix(name, ".")
}
