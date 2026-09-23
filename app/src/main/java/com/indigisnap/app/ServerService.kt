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
        val baseDir = File(filesDir, "IndigiSnap")
        if (!baseDir.exists()) {
            baseDir.mkdirs()
        }
        val basePath = baseDir.absolutePath
        android.util.Log.e("IndigiSnapDebug", "Base path: $basePath")
        android.util.Log.e("IndigiSnapDebug", "About to call ServerBridge.StartServer")
        try {
            ServerBridge.StartServer(PORT, basePath)
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
