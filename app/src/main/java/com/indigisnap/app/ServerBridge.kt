package com.indigisnap.app

/**
 * JNI bridge to the embedded Go backend (libindigisnap.so).
 *
 * The Go library exports two functions:
 *   StartServer(port int, baseDir *C.char)
 *   StopServer()
 *
 * These are invoked from ServerService to spin up / shut down the local
 * HTTP server that serves the IndigiSnap UI.
 */
object ServerBridge {

    init {
        System.loadLibrary("indigisnap")
    }

    external fun StartServer(port: Int, baseDir: String)
    external fun StopServer()
}
