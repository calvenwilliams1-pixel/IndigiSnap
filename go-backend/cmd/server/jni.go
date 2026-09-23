package main

/*
#include <stdlib.h>
#include <jni.h>

static char* indigisnap_jstring_to_c(JNIEnv* env, jstring s) {
    return (char*)(*env)->GetStringUTFChars(env, s, NULL);
}

static void indigisnap_release_jstring(JNIEnv* env, jstring s, char* c) {
    (*env)->ReleaseStringUTFChars(env, s, c);
}
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

//export Java_com_indigisnap_app_ServerBridge_StartServer
func Java_com_indigisnap_app_ServerBridge_StartServer(env *C.JNIEnv, clazz C.jclass, port C.jint, baseDir C.jstring) {
	baseC := C.indigisnap_jstring_to_c(env, baseDir)
	base := C.GoString(baseC)
	C.indigisnap_release_jstring(env, baseDir, baseC)

	startServer(int(port), base)
}

//export Java_com_indigisnap_app_ServerBridge_StopServer
func Java_com_indigisnap_app_ServerBridge_StopServer(env *C.JNIEnv, clazz C.jclass) {
	stopServer()
}

func startServer(port int, base string) {
	serverMu.Lock()
	defer serverMu.Unlock()

	if serverInstance != nil {
		log.Printf("Server already running; ignoring StartServer")
		return
	}

	if base == "" {
		base = "./IndigiSnap"
	}

	if err := os.MkdirAll(base, 0755); err != nil {
		log.Printf("Cannot create base dir %s: %v", base, err)
		return
	}

	mux := buildMux(base)
	addr := "127.0.0.1:" + itoa(port)

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

func stopServer() {
	serverMu.Lock()
	defer serverMu.Unlock()
	if serverInstance != nil {
		_ = serverInstance.Close()
		serverInstance = nil
	}
}

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
