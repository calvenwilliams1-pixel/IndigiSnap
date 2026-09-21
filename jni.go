package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	serverInstance *http.Server
	serverMu       sync.Mutex
)

// StartServer is called from Kotlin via JNI.
// It starts the IndigiSnap HTTP server on 127.0.0.1:port
// with the given base directory. Returns immediately.
//
//export StartServer
func StartServer(port C.int, baseDir *C.char) {
	serverMu.Lock()
	defer serverMu.Unlock()

	if serverInstance != nil {
		log.Printf("Server already running; ignoring StartServer")
		return
	}

	base := C.GoString(baseDir)
	if base == "" {
		base = "./IndigiSnap"
	}

	// Ensure base dir exists
	if err := os.MkdirAll(base, 0755); err != nil {
		log.Printf("Cannot create base dir %s: %v", base, err)
		return
	}

	mux := buildMux(base)
	addr := "127.0.0.1:" + itoa(int(port))

	serverInstance = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("IndigiSnap Go backend starting on %s", addr)
		log.Printf("Base dir: %s", base)
		if err := serverInstance.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
		serverMu.Lock()
		serverInstance = nil
		serverMu.Unlock()
	}()
}

// StopServer shuts down the HTTP server.
//
//export StopServer
func StopServer() {
	serverMu.Lock()
	defer serverMu.Unlock()
	if serverInstance != nil {
		_ = serverInstance.Close()
		serverInstance = nil
	}
}

// itoa is a tiny int-to-string to avoid extra imports in JNI code.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// main is required by Go for c-shared builds but does nothing here.
func main() {}
