package com.indigisnap.app

/**
 * JNI bridge to the embedded Go backend (libindigisnap.so).
 *
 * The Go library exports two functions:
 *   StartServer(port int, baseDir, metaDir, thumbDir *C.char)
 *   StopServer()
 *
 * baseDir  - media root (user-visible, e.g. Pictures/IndigiSnap)
 * metaDir  - app-private metadata root (filesDir)
 * thumbDir - app-cache thumbnail root (cacheDir/indigisnap-thumbs)
 *
 * These are invoked from ServerService to spin up / shut down the local
 * HTTP server that serves the IndigiSnap UI.
 */
object ServerBridge {

    init {
        android.util.Log.e("IndigiSnapDebug", "Loading libindigisnap.so")
        System.loadLibrary("indigisnap")
        android.util.Log.e("IndigiSnapDebug", "libindigisnap.so loaded OK")
    }

    external fun StartServer(port: Int, baseDir: String, metaDir: String, thumbDir: String)
    external fun StopServer()
}
