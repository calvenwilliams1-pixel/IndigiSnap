package com.indigisnap.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import java.io.File

class ServerService : Service() {

    companion object {
        private const val TAG = "IndigiSnap.Server"
        private const val CHANNEL_ID = "indigisnap_server"
        private const val NOTIFICATION_ID = 1
        private const val PORT = 8080
    }

    override fun onCreate() {
        super.onCreate()
        android.util.Log.e("IndigiSnapDebug", "ServerService onCreate START")
        try {
            createNotificationChannel()
            android.util.Log.e("IndigiSnapDebug", "Notification channel created")
            startForeground(NOTIFICATION_ID, buildNotification())
            android.util.Log.e("IndigiSnapDebug", "startForeground called")
            startServer()
        } catch (e: Throwable) {
            android.util.Log.e("IndigiSnapDebug", "ServerService onCreate FAILED", e)
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        android.util.Log.e("IndigiSnapDebug", "ServerService onStartCommand")
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onDestroy() {
        android.util.Log.e("IndigiSnapDebug", "ServerService onDestroy")
        try {
            ServerBridge.StopServer()
        } catch (e: Throwable) {
            Log.w(TAG, "StopServer failed", e)
        }
        super.onDestroy()
    }

    private fun startServer() {
        val baseDir = StorageConfig.baseDirFile(this)
        if (baseDir == null) {
            android.util.Log.e("IndigiSnapDebug", "Server cannot start: no base dir configured")
            return
        }
        if (!baseDir.exists()) {
            baseDir.mkdirs()
        }
        val basePath = baseDir.absolutePath

        // Metadata root: app-private filesDir. Survives app updates, dies
        // on uninstall. Contains favorites, recents, per-folder meta.
        val metaDir = filesDir
        if (!metaDir.exists()) {
            metaDir.mkdirs()
        }
        if (!metaDir.canWrite()) {
            android.util.Log.e("IndigiSnapDebug", "metaDir not writable: " + metaDir.absolutePath)
            return
        }
        val metaPath = metaDir.absolutePath

        // Thumbnail root: app-cache, safe to clear under storage pressure.
        val thumbDir = java.io.File(cacheDir, "indigisnap-thumbs")
        if (!thumbDir.exists()) {
            thumbDir.mkdirs()
        }
        val thumbPath = thumbDir.absolutePath

        android.util.Log.e("IndigiSnapDebug", "BUILD_MARKER=2ai-1")  // confirm fresh APK
        android.util.Log.e("IndigiSnapDebug", "Base path:  $basePath")
        android.util.Log.e("IndigiSnapDebug", "Meta path:  $metaPath")
        android.util.Log.e("IndigiSnapDebug", "Thumb path: $thumbPath")
        android.util.Log.e("IndigiSnapDebug", "About to call ServerBridge.StartServer(4-param)")
        try {
            ServerBridge.StartServer(PORT, basePath, metaPath, thumbPath)
            android.util.Log.e("IndigiSnapDebug", "ServerBridge.StartServer returned OK")
        } catch (e: Throwable) {
            android.util.Log.e("IndigiSnapDebug", "ServerBridge.StartServer FAILED", e)
        }
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "IndigiSnap Server",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Keeps the local photo server running"
            }
            val nm = getSystemService(NotificationManager::class.java)
            nm.createNotificationChannel(channel)
        }
    }

    private fun buildNotification(): Notification {
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("IndigiSnap")
            .setContentText("Photo server running")
            .setSmallIcon(android.R.drawable.ic_menu_gallery)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .build()
    }
}
