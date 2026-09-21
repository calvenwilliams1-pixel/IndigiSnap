package main

import (
	"log"
	"net/http"
	"os"

	"github.com/calvenwilliams1-pixel/indigisnap/internal/handlers"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/meta"
	"github.com/calvenwilliams1-pixel/indigisnap/internal/ui"
)

func main() {
	port := os.Getenv("INDIGISNAP_PORT")
	if port == "" {
		port = "8080"
	}

	baseDir := os.Getenv("INDIGISNAP_BASE_DIR")
	if baseDir == "" {
		baseDir = "./IndigiSnap"
	}

	// Make sure base dir exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Cannot create base dir %s: %v", baseDir, err)
	}

	browseHandler := handlers.NewBrowseHandler(baseDir)
	viewHandler := handlers.NewViewHandler(baseDir)
	actionHandler := handlers.NewActionHandler(baseDir)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})

	mux.Handle("/browse", browseHandler)
	mux.Handle("/browse/", browseHandler)
	mux.Handle("/view/", viewHandler)

	mux.HandleFunc("/create_folder/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", 405)
			return
		}
		actionHandler.CreateFolder(w, r)
	})
	mux.HandleFunc("/rename_folder/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", 405)
			return
		}
		actionHandler.RenameFolder(w, r)
	})
	mux.HandleFunc("/delete_folder/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", 405)
			return
		}
		actionHandler.DeleteFolder(w, r)
	})
	mux.HandleFunc("/delete_picture/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", 405)
			return
		}
		actionHandler.DeletePicture(w, r)
	})
	mux.HandleFunc("/toggle_favorite/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", 405)
			return
		}
		actionHandler.ToggleFavorite(w, r)
	})
	mux.HandleFunc("/set_sort/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", 405)
			return
		}
		actionHandler.SetSort(w, r)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := ui.TemplateData{
			Folder:      "",
			Items:       []ui.FolderItem{},
			Images:      []ui.MediaItem{},
			Meta:        ui.FolderMeta{Sort: "Newest"},
			Recents:     []ui.Recent{},
			Page:        1,
			TotalPages:  1,
			LogoURL:     "",
			SortBy:      "newest",
			FilterType:  "all",
			Breadcrumbs: toUIBreadcrumbs(meta.GetBreadcrumbs("")),
		}
		html, err := ui.Render(data)
		if err != nil {
			log.Printf("Template render error: %v", err)
			http.Error(w, "Template error: "+err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	addr := "127.0.0.1:" + port
	log.Printf("IndigiSnap Go backend starting on %s", addr)
	log.Printf("Base dir: %s", baseDir)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// toUIBreadcrumbs converts meta.Breadcrumb to ui.Breadcrumb.
func toUIBreadcrumbs(in []meta.Breadcrumb) []ui.Breadcrumb {
	out := make([]ui.Breadcrumb, 0, len(in))
	for _, b := range in {
		out = append(out, ui.Breadcrumb{Label: b.Label, Path: b.URL})
	}
	return out
}
