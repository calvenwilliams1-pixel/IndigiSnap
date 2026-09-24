package com.indigisnap.app

import android.annotation.SuppressLint
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.webkit.ValueCallback
import android.webkit.WebChromeClient
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import java.net.HttpURLConnection
import java.net.URL
import kotlin.concurrent.thread

class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView
    private var pendingFileCallback: ValueCallback<Array<Uri>>? = null
    private val fileChooserRequestCode = 1001

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        android.util.Log.e("IndigiSnapDebug", "MainActivity onCreate START")

        // Request runtime permissions
        if (android.os.Build.VERSION.SDK_INT >= 23) {
            val perms = mutableListOf<String>()
            perms.add(android.Manifest.permission.CAMERA)
            if (android.os.Build.VERSION.SDK_INT >= 33) {
                perms.add("android.permission.READ_MEDIA_IMAGES")
                perms.add("android.permission.READ_MEDIA_VIDEO")
            } else {
                perms.add(android.Manifest.permission.READ_EXTERNAL_STORAGE)
                perms.add(android.Manifest.permission.WRITE_EXTERNAL_STORAGE)
            }
            requestPermissions(perms.toTypedArray(), 2001)
        }

        webView = WebView(this)
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        webView.settings.databaseEnabled = true
        webView.settings.allowFileAccess = true
        webView.settings.allowContentAccess = true
        webView.settings.mediaPlaybackRequiresUserGesture = false
        webView.webViewClient = WebViewClient()

        webView.webChromeClient = object : WebChromeClient() {
            override fun onShowFileChooser(
                wv: WebView?,
                callback: ValueCallback<Array<Uri>>?,
                params: FileChooserParams?
            ): Boolean {
                pendingFileCallback?.onReceiveValue(null)
                pendingFileCallback = callback
                return try {
                    val intent = params?.createIntent()
                    if (intent == null) {
                        pendingFileCallback = null
                        return false
                    }
                    if (params.mode == WebChromeClient.FileChooserParams.MODE_OPEN_MULTIPLE) {
                        intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true)
                    }
                    startActivityForResult(intent, fileChooserRequestCode)
                    true
                } catch (e: Exception) {
                    android.util.Log.e("IndigiSnapDebug", "File chooser failed", e)
                    pendingFileCallback = null
                    false
                }
            }
        }

        setContentView(webView)

        android.util.Log.e("IndigiSnapDebug", "About to start ServerService")
        try {
            val serviceIntent = Intent(this, ServerService::class.java)
            ContextCompat.startForegroundService(this, serviceIntent)
            android.util.Log.e("IndigiSnapDebug", "ServerService start OK")
        } catch (e: Throwable) {
            android.util.Log.e("IndigiSnapDebug", "ServerService start FAILED", e)
        }

        webView.loadData(
            """
            <!DOCTYPE html><html><head><meta name="viewport" content="width=device-width,initial-scale=1">
            <style>body{background:#07040d;color:#fff;font-family:monospace;display:flex;flex-direction:column;align-items:center;justify-content:center;height:100vh;margin:0}
            h1{color:#ff00ff;text-shadow:0 0 20px #ff00ff;letter-spacing:4px;font-size:2rem}
            .icon{font-size:4rem;margin-bottom:1rem}
            p{color:#00ff99}</style></head>
            <body><div class="icon">👾</div><h1>INDIGISNAP</h1><p>Loading...</p></body></html>
            """.trimIndent(),
            "text/html",
            "UTF-8"
        )

        waitForServerAndLoad()
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != fileChooserRequestCode || pendingFileCallback == null) return

        val uris = linkedSetOf<Uri>()

        if (resultCode == RESULT_OK) {
            data?.clipData?.let { clip ->
                for (i in 0 until clip.itemCount) {
                    clip.getItemAt(i).uri?.let { uris.add(it) }
                }
            }
            data?.data?.let { uris.add(it) }
        }

        android.util.Log.e("IndigiSnapDebug", "File chooser result: ${uris.size} URI(s)")

        pendingFileCallback?.onReceiveValue(
            uris.takeIf { it.isNotEmpty() }?.toTypedArray()
        )
        pendingFileCallback = null
    }

    private fun waitForServerAndLoad() {
        thread {
            val maxAttempts = 50
            var loaded = false
            for (i in 1..maxAttempts) {
                if (isServerUp()) {
                    loaded = true
                    break
                }
                Thread.sleep(100)
            }
            runOnUiThread {
                if (loaded) {
                    webView.loadUrl("http://127.0.0.1:8080/browse")
                } else {
                    webView.loadData(
                        "<html><body style='background:#07040d;color:#fff;font-family:monospace;padding:30px'><h2 style='color:#ff0055'>Server did not start</h2><p>Check logcat for errors.</p></body></html>",
                        "text/html",
                        "UTF-8"
                    )
                }
            }
        }
    }

    private fun isServerUp(): Boolean {
        return try {
            val url = URL("http://127.0.0.1:8080/health")
            val conn = url.openConnection() as HttpURLConnection
            conn.connectTimeout = 500
            conn.readTimeout = 500
            conn.requestMethod = "GET"
            val code = conn.responseCode
            conn.disconnect()
            code == 200
        } catch (e: Exception) {
            false
        }
    }

    override fun onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack()
        } else {
            @Suppress("DEPRECATION")
            super.onBackPressed()
        }
    }
}
