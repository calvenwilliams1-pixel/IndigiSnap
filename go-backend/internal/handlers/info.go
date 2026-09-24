package handlers

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/security"
)

// InfoHandler serves GET /image_info/<path>
// Returns JSON metadata for an image. EXIF fields are populated when present.
// Use ?debug=1 to include the full raw EXIF tag map.
type InfoHandler struct {
	BaseDir string
}

func NewInfoHandler(baseDir string) *InfoHandler {
	return &InfoHandler{BaseDir: baseDir}
}

// infoResponse is the JSON shape returned by /image_info.
type infoResponse struct {
	HasExif     bool              `json:"has_exif"`
	CameraMake  string            `json:"camera_make"`
	CameraModel string            `json:"camera_model"`
	ISO         int               `json:"iso"`
	Shutter     string            `json:"shutter"`
	Aperture    string            `json:"aperture"`
	DateTime    string            `json:"datetime"`
	Orientation int               `json:"orientation"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	SizeBytes   int64             `json:"size_bytes"`
	RawExif     map[string]string `json:"raw_exif,omitempty"`
}

func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/image_info/")
	raw = strings.TrimPrefix(raw, "/")

	rel, err := security.ValidatePath(raw)
	if err != nil {
		http.Error(w, "Invalid path", 400)
		return
	}

	full := filepath.Join(h.BaseDir, security.ToOSPath(rel))
	if !security.IsSafePath(h.BaseDir, full) {
		http.Error(w, "Access denied", 403)
		return
	}

	info, err := os.Stat(full)
	if err != nil {
		http.Error(w, "File not found", 404)
		return
	}

	resp := infoResponse{
		HasExif:     false,
		Orientation: 1,
		SizeBytes:   info.Size(),
	}

	// Get base dimensions via image.DecodeConfig (works for PNG, GIF, JPEG)
	if w0, h0, err := decodeDimensions(full); err == nil {
		resp.Width = w0
		resp.Height = h0
	}

	// Only attempt EXIF on JPEG
	if isJpeg(full) {
		extractExif(full, &resp)
	}

	// Debug: include raw EXIF tag dump if requested
	if r.URL.Query().Get("debug") == "1" && resp.HasExif {
		resp.RawExif = dumpRawExif(full)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}

// extractExif opens the file, decodes EXIF, and populates resp fields.
// On any failure, resp is left as-is (HasExif stays false).
func extractExif(path string, resp *infoResponse) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("info.go: open failed for %s: %v", path, err)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("info.go: close failed for %s: %v", path, cerr)
		}
	}()

	x, err := exif.Decode(f)
	if err != nil {
		// Common for files without EXIF. Not an error worth logging at WARN.
		return
	}
	resp.HasExif = true

	if v, err := x.Get(exif.Make); err == nil {
		if s, err := v.StringVal(); err == nil {
			resp.CameraMake = strings.TrimSpace(s)
		}
	}
	if v, err := x.Get(exif.Model); err == nil {
		if s, err := v.StringVal(); err == nil {
			resp.CameraModel = strings.TrimSpace(s)
		}
	}
	if v, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if n, err := v.Int(0); err == nil {
			resp.ISO = n
		}
	}
	if v, err := x.Get(exif.ExposureTime); err == nil {
		if r, err := v.Rat(0); err == nil {
			resp.Shutter = formatShutter(r)
		}
	}
	if v, err := x.Get(exif.FNumber); err == nil {
		if r, err := v.Rat(0); err == nil {
			resp.Aperture = formatAperture(r)
		}
	}
	if v, err := x.Get(exif.DateTimeOriginal); err == nil {
		if s, err := v.StringVal(); err == nil {
			resp.DateTime = parseExifDate(s)
		}
	} else if v, err := x.Get(exif.DateTime); err == nil {
		if s, err := v.StringVal(); err == nil {
			resp.DateTime = parseExifDate(s)
		}
	}
	if v, err := x.Get(exif.Orientation); err == nil {
		if n, err := v.Int(0); err == nil && n > 0 && n <= 8 {
			resp.Orientation = n
		}
	}
	// EXIF dimensions override the base dimensions if present
	if v, err := x.Get(exif.PixelXDimension); err == nil {
		if n, err := v.Int(0); err == nil && n > 0 {
			resp.Width = n
		}
	}
	if v, err := x.Get(exif.PixelYDimension); err == nil {
		if n, err := v.Int(0); err == nil && n > 0 {
			resp.Height = n
		}
	}
}

// dumpRawExif returns the full EXIF tag map as strings. Debug only.
func dumpRawExif(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	x, err := exif.Decode(f)
	if err != nil {
		return out
	}
	x.Walk(&tiffMapWalker{out: out})
	return out
}

// tiffMapWalker implements exif.Walker by filling a string map.
type tiffMapWalker struct {
	out map[string]string
}

func (w *tiffMapWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	w.out[string(name)] = tag.String()
	return nil
}

// decodeDimensions returns width and height using image.DecodeConfig.
func decodeDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// isJpeg reports whether the file has a .jpg or .jpeg extension (case-insensitive).
func isJpeg(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg"
}

// formatShutter formats exposure time as "1/125s" or "10s".
func formatShutter(r *big.Rat) string {
	if r == nil || r.Sign() == 0 {
		return ""
	}
	// Common case: 1/N
	if r.Num().Cmp(big.NewInt(1)) == 0 {
		return r.RatString() + "s"
	}
	// If denominator > numerator, we want the reciprocal as 1/N
	if r.Num().Cmp(r.Denom()) < 0 {
		recip := new(big.Rat).Inv(r)
		if recip.Num().Cmp(big.NewInt(1)) == 0 {
			return "1/" + recip.Denom().String() + "s"
		}
		// Non-exact reciprocal, format as decimal
		return fmt.Sprintf("%.4fs", r64(r))
	}
	return r.RatString() + "s"
}

// formatAperture formats F-number as "f/2.8".
func formatAperture(r *big.Rat) string {
	if r == nil || r.Sign() == 0 {
		return ""
	}
	v := r64(r)
	if v == float64(int(v)) {
		return fmt.Sprintf("f/%.0f", v)
	}
	return fmt.Sprintf("f/%.1f", v)
}

// r64 converts a *big.Rat to float64.
func r64(r *big.Rat) float64 {
	f, _ := r.Float64()
	return f
}

// parseExifDate converts "2006:01:02 15:04:05" to ISO 8601.
// Returns the input unchanged on parse failure.
func parseExifDate(s string) string {
	s = strings.TrimSpace(s)
	t, err := time.Parse("2006:01:02 15:04:05", s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02T15:04:05")
}
