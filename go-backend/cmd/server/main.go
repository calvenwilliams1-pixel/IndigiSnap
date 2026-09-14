package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>IndigiSnap</title></head>
<body style="background:#07040d;color:#fff;font-family:monospace;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;flex-direction:column;">
<div style="font-size:5rem;">👾</div>
<h1 style="color:#ff00ff;text-shadow:0 0 20px #ff00ff;letter-spacing:4px;">INDIGISNAP</h1>
<p style="color:#00ff99;">Go backend running</p>
<p style="opacity:0.6;font-size:0.9rem;">Base dir: %s</p>
</body>
</html>`, baseDir)
	})

	addr := "127.0.0.1:" + port
	log.Printf("IndigiSnap Go backend starting on %s", addr)
	log.Printf("Base dir: %s", baseDir)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
