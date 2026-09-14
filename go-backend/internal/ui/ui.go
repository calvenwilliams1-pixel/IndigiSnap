package ui

import (
	"bytes"
	_ "embed"
	"html/template"
)

//go:embed interface.html
var interfaceHTML string

// TemplateData carries the fields the HTML template expects.
// This mirrors the Jinja2 context from the original Flask app.
type TemplateData struct {
	Folder        string
	Items         []FolderItem
	Images        []MediaItem
	Meta          FolderMeta
	Recents       []Recent
	Page          int
	TotalPages    int
	LogoURL       string
	SortBy        string
	FilterType    string
	Breadcrumbs   []Breadcrumb
}

// FolderItem is a subfolder tile.
type FolderItem struct {
	Name      string
	URL       string
	Previews  []string
}

// MediaItem is an image or video tile.
type MediaItem struct {
	Name       string
	RelPath    string
	IsVideo    bool
	ThumbURL   string
	IsFavorite bool
	Type       string
	Size       int64
	Duration   float64
	DateTaken  string
}

// FolderMeta is the subset of meta fields the UI needs.
type FolderMeta struct {
	Sort string
}

// Recent is a recent-folder pill.
type Recent struct {
	Name string
	Path string
}

// Breadcrumb is a nav crumb.
type Breadcrumb struct {
	Label string
	Path  string
}

var parsedTemplate = template.Must(template.New("interface").Parse(interfaceHTML))

// Render executes the interface template with the given data.
func Render(data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := parsedTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
