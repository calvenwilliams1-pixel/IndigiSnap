package com.indigisnap.app

import android.Manifest
import android.annotation.SuppressLint
import android.app.Activity
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.provider.MediaStore
import android.provider.Settings
import android.util.Log
import android.webkit.JavascriptInterface
import android.webkit.ValueCallback
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.result.ActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.core.content.FileProvider
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import kotlin.concurrent.thread

class MainActivity : AppCompatActivity() {

    companion object {
        private const val TAG = "IndigiSnap.Main"
        private const val PERMISSION_REQUEST_CODE = 2001
        private const val FILE_CHOOSER_REQUEST_CODE = 1001
        private const val CAMERA_ACTIVITY_REQUEST_CODE = 1002

        // WebView is expected to load ONLY from these origins. Any other
        // navigation is blocked. Extend if the backend port changes.
        private val ALLOWED_HOSTS = setOf("127.0.0.1:8080", "localhost:8080")
    }

    private lateinit var webView: WebView

    // WebView file chooser (used by <input type="file"> in interface.html)
    private var pendingFileCallback: ValueCallback<Array<Uri>>? = null

    // Legacy camera output tracking (used by LEGACY_SINGLE and LEGACY_FALLBACK)
    private var legacyCameraOutputUri: Uri? = null
    private var legacyCameraOutputFile: File? = null

    // Session state (only used by CAMERA_X and LEGACY_FALLBACK paths)
    private var pendingSessionId: String? = null
    private var activeSessionId: String? = null
    private var activeSessionShotCount: Int = 0
    private var activeSessionLastShotPath: String? = null
    private var fallbackDialog: AlertDialog? = null

    // Launch guard — prevents overlapping camera launcher calls
    private enum class LaunchType { CAMERA_X, LEGACY_SINGLE, LEGACY_FALLBACK }
    @Volatile private var isLaunchInProgress: Boolean = false
    private var pendingLaunchType: LaunchType? = null

    // ---------------------------------------------------------------
    // Launchers (modern Activity Result API)
    // ---------------------------------------------------------------

    private val cameraPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        if (granted) {
            launchCameraXSession()
        } else {
            if (shouldShowRequestPermissionRationale(Manifest.permission.CAMERA)) {
                showCameraRationaleDialog()
            } else {
                showCameraSettingsDialog()
            }
        }
    }

    private val cameraLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        val type = pendingLaunchType
        pendingLaunchType = null
        isLaunchInProgress = false  // clear unconditionally

        when (type) {
            LaunchType.CAMERA_X -> handleCameraXResult(result)
            LaunchType.LEGACY_SINGLE -> handleLegacySingleResult(result)
            LaunchType.LEGACY_FALLBACK -> handleLegacyFallbackResult(result)
            null -> Log.w(TAG, "Launcher result with no pending type — ignored")
        }
    }

    // ---------------------------------------------------------------
    // Lifecycle
    // ---------------------------------------------------------------

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Log.i(TAG, "onCreate START")

        // Preserve existing launch-time permission batch request (CAMERA + media)
        if (Build.VERSION.SDK_INT >= 23) {
            val perms = mutableListOf<String>()
            perms.add(Manifest.permission.CAMERA)
            if (Build.VERSION.SDK_INT >= 33) {
                perms.add("android.permission.READ_MEDIA_IMAGES")
                perms.add("android.permission.READ_MEDIA_VIDEO")
            } else {
                perms.add(Manifest.permission.READ_EXTERNAL_STORAGE)
                perms.add(Manifest.permission.WRITE_EXTERNAL_STORAGE)
            }
            requestPermissions(perms.toTypedArray(), PERMISSION_REQUEST_CODE)
        }

        // Orphan inbox detection — log only, no UI (recovery is 4.5)
        logOrphanedInboxes()

        setupWebView()
        setContentView(webView)

        try {
            val serviceIntent = Intent(this, ServerService::class.java)
            ContextCompat.startForegroundService(this, serviceIntent)
            Log.i(TAG, "ServerService start OK")
        } catch (e: Throwable) {
            Log.e(TAG, "ServerService start FAILED", e)
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

    // ---------------------------------------------------------------
    // WebView setup
    // ---------------------------------------------------------------

    @SuppressLint("SetJavaScriptEnabled")
    private fun setupWebView() {
        webView = WebView(this)
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        webView.settings.databaseEnabled = true
        webView.settings.allowFileAccess = true
        webView.settings.allowContentAccess = true
        webView.settings.mediaPlaybackRequiresUserGesture = false

        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(
                view: WebView?,
                request: WebResourceRequest?
            ): Boolean {
                val url = request?.url?.toString() ?: return false
                // Allow about:blank and data: URLs used by loadData
                if (url.startsWith("about:") || url.startsWith("data:")) return false

                val parsed = try { java.net.URI(url) } catch (e: Exception) { return true }
                val hostPort = "${parsed.host}:${if (parsed.port > 0) parsed.port else 80}"
                if (ALLOWED_HOSTS.contains(hostPort)) {
                    return false  // allow
                }
                Log.w(TAG, "Blocked navigation: $url")
                return true  // block
            }
        }

        webView.webChromeClient = object : WebChromeClient() {
            override fun onShowFileChooser(
                wv: WebView?,
                callback: ValueCallback<Array<Uri>>?,
                params: FileChooserParams?
            ): Boolean {
                pendingFileCallback?.onReceiveValue(null)
                pendingFileCallback = callback

                // Capture attribute: route to LEGACY_SINGLE (system camera, one shot)
                if (params?.isCaptureEnabled == true) {
                    if (isLaunchInProgress) {
                        Log.d(TAG, "File chooser capture ignored: launch in progress")
                        pendingFileCallback = null
                        return false
                    }
                    val acceptsVideo = params.acceptTypes?.any { it.startsWith("video/") } == true
                    return launchLegacySingle(acceptsVideo)
                }

                // Regular file picking
                return try {
                    val intent = params?.createIntent()
                    if (intent == null) {
                        pendingFileCallback = null
                        return false
                    }
                    if (params.mode == WebChromeClient.FileChooserParams.MODE_OPEN_MULTIPLE) {
                        intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true)
                    }
                    startActivityForResult(intent, FILE_CHOOSER_REQUEST_CODE)
                    true
                } catch (e: Exception) {
                    Log.e(TAG, "File chooser failed", e)
                    pendingFileCallback = null
                    false
                }
            }
        }

        webView.addJavascriptInterface(WebAppBridge(), "Android")
    }

    // ---------------------------------------------------------------
    // JS bridge
    // ---------------------------------------------------------------

    inner class WebAppBridge {
        @JavascriptInterface
        fun openCamera(folder: String) {
            if (isLaunchInProgress) {
                Log.d(TAG, "openCamera ignored: launch already in progress")
                return
            }
            val target = folder.trim()
            Log.i(TAG, "openCamera from JS, folder= + target + ")
            runOnUiThread { handleOpenCameraRequest(target) }
        }
    }

    private var pendingFolder: String = ""

    private fun handleOpenCameraRequest(folder: String) {
        pendingFolder = folder
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA)
            == PackageManager.PERMISSION_GRANTED
        ) {
            launchCameraXSession()
            return
        }
        if (shouldShowRequestPermissionRationale(Manifest.permission.CAMERA)) {
            showCameraRationaleDialog()
        } else {
            cameraPermissionLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    private fun launchCameraXSession() {
        if (isLaunchInProgress) {
            Log.d(TAG, "launchCameraXSession ignored: launch in progress")
            return
        }
        val sessionId = SessionState.newSessionId()
        activeSessionId = sessionId
        activeSessionShotCount = 0
        activeSessionLastShotPath = null

        val intent = Intent(this, CameraActivity::class.java).apply {
            putExtra(CameraActivity.EXTRA_SESSION_ID, sessionId)
            putExtra(CameraActivity.EXTRA_FOLDER, pendingFolder)
        }
        Log.i(TAG, "launchCameraXSession folder= + pendingFolder + ")
        beginLaunch(LaunchType.CAMERA_X, intent)
    }

    private fun beginLaunch(type: LaunchType, intent: Intent) {
        isLaunchInProgress = true
        pendingLaunchType = type
        try {
            cameraLauncher.launch(intent)
        } catch (e: Exception) {
            isLaunchInProgress = false
            pendingLaunchType = null
            Log.e(TAG, "cameraLauncher.launch failed: ${e.message}", e)
        }
    }

    // ---------------------------------------------------------------
    // Result handlers
    // ---------------------------------------------------------------

    private fun handleCameraXResult(result: ActivityResult) {
        when (result.resultCode) {
            RESULT_OK -> {
                @Suppress("DEPRECATION")
                val state = result.data?.getParcelableExtra<SessionState>("session_state")
                if (state != null) {
                    Log.i(TAG, "CameraX session finished: id=${state.sessionId} shots=${state.shotCount} folder=${state.folder}")
                    reloadWebViewToFolder(state.folder)
                } else {
                    Log.w(TAG, "CameraX RESULT_OK but no SessionState in extras")
                    reloadWebViewToBrowse()
                }
            }
            Activity.RESULT_CANCELED -> {
                Log.i(TAG, "CameraX session cancelled (no shots)")
            }
            CameraActivity.RESULT_CAMERA_UNAVAILABLE -> {
                val reason = result.data?.getStringExtra("reason") ?: "Unknown"
                val sessionId = result.data?.getStringExtra("session_id") ?: SessionState.newSessionId()
                Log.w(TAG, "CameraX unavailable: $reason")
                handleCameraUnavailable(sessionId, reason)
            }
            else -> {
                Log.w(TAG, "CameraX returned unexpected resultCode=${result.resultCode}")
            }
        }
    }

    private fun handleCameraUnavailable(sessionId: String, reason: String) {
        if (reason.startsWith("PermissionRevoked")) {
            // Permission was granted at check-time but revoked mid-launch.
            // Re-check and either re-launch or route to rationale/settings.
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA)
                == PackageManager.PERMISSION_GRANTED
            ) {
                Log.i(TAG, "Permission now granted, relaunching CameraX")
                launchCameraXSession()
            } else if (shouldShowRequestPermissionRationale(Manifest.permission.CAMERA)) {
                showCameraRationaleDialog()
            } else {
                showCameraSettingsDialog()
            }
            return
        }
        // Any other reason: legacy fallback dialog
        promptLegacyFallback(sessionId)
    }

    private fun promptLegacyFallback(sessionId: String) {
        fallbackDialog?.dismiss()
        val dialog = AlertDialog.Builder(this)
            .setTitle("Camera unavailable")
            .setMessage("Use the system camera to capture this session?")
            .setPositiveButton("Use system camera") { _, _ ->
                launchLegacyFallback(sessionId)
            }
            .setNegativeButton("Cancel") { _, _ ->
                Log.i(TAG, "Fallback cancelled by user")
            }
            .setCancelable(false)
            .create()
        dialog.setOnCancelListener {
            Log.i(TAG, "Fallback dialog cancelled via back")
        }
        fallbackDialog = dialog
        dialog.show()
    }

    private fun launchLegacyFallback(sessionId: String) {
        if (isLaunchInProgress) {
            Log.d(TAG, "launchLegacyFallback ignored: launch in progress")
            return
        }
        activeSessionId = sessionId
        activeSessionShotCount = 0
        activeSessionLastShotPath = null

        val intent = Intent(MediaStore.ACTION_IMAGE_CAPTURE).apply {
            val (uri, file) = createInboxOutputFile(sessionId, ".jpg")
            legacyCameraOutputUri = uri
            legacyCameraOutputFile = file
            putExtra(MediaStore.EXTRA_OUTPUT, uri)
            addFlags(Intent.FLAG_GRANT_WRITE_URI_PERMISSION)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }
        beginLaunch(LaunchType.LEGACY_FALLBACK, intent)
    }

    private fun handleLegacyFallbackResult(result: ActivityResult) {
        val file = legacyCameraOutputFile
        val uri = legacyCameraOutputUri
        legacyCameraOutputFile = null
        legacyCameraOutputUri = null

        if (result.resultCode != RESULT_OK || file == null || uri == null) {
            Log.i(TAG, "Legacy fallback cancelled")
            return
        }
        if (!file.exists() || file.length() == 0L) {
            Log.w(TAG, "Legacy fallback produced empty file")
            return
        }

        activeSessionShotCount += 1
        activeSessionLastShotPath = file.absolutePath
        Log.i(TAG, "Legacy fallback captured: ${file.absolutePath}")

        showAddAnotherDialog()
    }

    private fun showAddAnotherDialog() {
        val sessionId = activeSessionId ?: return
        val dialog = AlertDialog.Builder(this)
            .setTitle("Add another photo?")
            .setMessage("Shots this session: $activeSessionShotCount")
            .setPositiveButton("Add another") { _, _ ->
                launchLegacyFallback(sessionId)
            }
            .setNegativeButton("Done") { _, _ ->
                finishLegacyFallbackSession()
            }
            .setCancelable(false)
            .create()
        dialog.setOnCancelListener {
            // Route through the Done path — no third way out
            finishLegacyFallbackSession()
        }
        dialog.show()
    }

    private fun finishLegacyFallbackSession() {
        val sessionId = activeSessionId ?: run {
            Log.w(TAG, "finishLegacyFallbackSession with no active session")
            return
        }
        Log.i(TAG, "Legacy session finished: id=$sessionId shots=$activeSessionShotCount")

        val state = SessionState(
            sessionId = sessionId,
            shotCount = activeSessionShotCount,
            lastShotPath = activeSessionLastShotPath,
            sessionFinished = activeSessionShotCount > 0,
            fallbackUsed = true,
            failureReason = null,
            cameraStartTimeMs = 0L,
        )
        // No setResult needed — this runs on MainActivity. Just refresh the WebView.
        Log.i(TAG, "Session state: $state")
        reloadWebViewToBrowse()
    }

    private fun handleLegacySingleResult(result: ActivityResult) {
        // LEGACY_SINGLE is the plain <input type="file" capture> path.
        // It is NOT part of any session. It must not read or write
        // activeSessionId, lastShotPath, or shotCount.
        val file = legacyCameraOutputFile
        val uri = legacyCameraOutputUri
        legacyCameraOutputFile = null
        legacyCameraOutputUri = null

        val callback = pendingFileCallback
        pendingFileCallback = null

        if (result.resultCode == RESULT_OK && uri != null && file != null && file.length() > 0L) {
            callback?.onReceiveValue(arrayOf(uri))
        } else {
            Log.i(TAG, "LEGACY_SINGLE cancelled or empty")
            callback?.onReceiveValue(null)
        }
    }

    private fun launchLegacySingle(video: Boolean): Boolean {
        return try {
            val action = if (video) MediaStore.ACTION_VIDEO_CAPTURE else MediaStore.ACTION_IMAGE_CAPTURE
            val ext = if (video) ".mp4" else ".jpg"
            val cacheDir = File(cacheDir, "camera").apply { mkdirs() }
            val tempFile = File.createTempFile("capture_", ext, cacheDir)
            val uri = FileProvider.getUriForFile(this, "com.indigisnap.app.fileprovider", tempFile)
            legacyCameraOutputUri = uri
            legacyCameraOutputFile = tempFile

            val intent = Intent(action).apply {
                putExtra(MediaStore.EXTRA_OUTPUT, uri)
                addFlags(Intent.FLAG_GRANT_WRITE_URI_PERMISSION)
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                if (video) putExtra(MediaStore.EXTRA_VIDEO_QUALITY, 1)
            }
            beginLaunch(LaunchType.LEGACY_SINGLE, intent)
            true
        } catch (e: Exception) {
            Log.e(TAG, "Legacy single launch failed", e)
            pendingFileCallback?.onReceiveValue(null)
            pendingFileCallback = null
            false
        }
    }

    // ---------------------------------------------------------------
    // File chooser legacy callback (non-capture path)
    // ---------------------------------------------------------------

    @Deprecated("Kept for legacy WebView file chooser path only")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != FILE_CHOOSER_REQUEST_CODE) return
        val callback = pendingFileCallback ?: return
        pendingFileCallback = null

        val uris = linkedSetOf<Uri>()
        if (resultCode == RESULT_OK) {
            data?.clipData?.let { clip ->
                for (i in 0 until clip.itemCount) clip.getItemAt(i).uri?.let { uris.add(it) }
            }
            data?.data?.let { uris.add(it) }
        }
        Log.i(TAG, "File chooser: ${uris.size} URI(s)")
        callback.onReceiveValue(uris.takeIf { it.isNotEmpty() }?.toTypedArray())
    }

    // ---------------------------------------------------------------
    // Permission dialogs
    // ---------------------------------------------------------------

    private fun showCameraRationaleDialog() {
        AlertDialog.Builder(this)
            .setTitle("Camera access needed")
            .setMessage("IndigiSnap needs camera access to capture photos and videos for your library.")
            .setPositiveButton("Allow") { _, _ ->
                cameraPermissionLauncher.launch(Manifest.permission.CAMERA)
            }
            .setNegativeButton("Not now") { _, _ ->
                Log.i(TAG, "Rationale declined")
            }
            .setCancelable(false)
            .show()
    }

    private fun showCameraSettingsDialog() {
        AlertDialog.Builder(this)
            .setTitle("Camera access blocked")
            .setMessage("Camera permission is permanently denied. Open Settings to enable it?")
            .setPositiveButton("Open Settings") { _, _ ->
                try {
                    val intent = Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
                        data = Uri.fromParts("package", packageName, null)
                    }
                    startActivity(intent)
                } catch (e: Exception) {
                    Log.e(TAG, "Failed to open settings", e)
                }
            }
            .setNegativeButton("Cancel") { _, _ -> }
            .setCancelable(false)
            .show()
    }

    // ---------------------------------------------------------------
    // Helpers
    // ---------------------------------------------------------------

    private fun createInboxOutputFile(sessionId: String, ext: String): Pair<Uri, File> {
        val inboxDir = File(File(filesDir, "IndigiSnap/.inbox"), sessionId)
        if (!inboxDir.exists()) inboxDir.mkdirs()
        val ts = System.currentTimeMillis()
        val seq = activeSessionShotCount + 1
        val file = File(inboxDir, "IMG_${ts}_$seq$ext")
        file.createNewFile()
        val uri = FileProvider.getUriForFile(this, "com.indigisnap.app.fileprovider", file)
        return Pair(uri, file)
    }

    private fun logOrphanedInboxes() {
        try {
            val inboxRoot = File(filesDir, "IndigiSnap/.inbox")
            if (!inboxRoot.exists()) return
            val sessions = inboxRoot.listFiles()?.filter { it.isDirectory } ?: emptyList()
            if (sessions.isEmpty()) return
            Log.w(TAG, "Orphaned inbox sessions found: ${sessions.size}")
            sessions.forEach { s ->
                val files = s.listFiles()?.size ?: 0
                Log.w(TAG, "  session=${s.name} files=$files")
            }
        } catch (e: Exception) {
            Log.w(TAG, "logOrphanedInboxes failed: ${e.message}")
        }
    }

    private fun reloadWebViewToBrowse() {
        webView.post {
            webView.loadUrl("http://127.0.0.1:8080/browse")
        }
    }

    private fun reloadWebViewToFolder(folder: String) {
        val path = if (folder.isEmpty()) {
            "http://127.0.0.1:8080/browse"
        } else {
            "http://127.0.0.1:8080/browse/" + Uri.encode(folder)
        }
        Log.i(TAG, "reloadWebViewToFolder: " + path)
        webView.post {
            webView.loadUrl(path)
        }
    }

    private fun waitForServerAndLoad() {
        thread {
            val maxAttempts = 50
            var loaded = false
            for (i in 1..maxAttempts) {
                if (isServerUp()) { loaded = true; break }
                Thread.sleep(100)
            }
            runOnUiThread {
                if (loaded) {
                    webView.loadUrl("http://127.0.0.1:8080/browse")
                } else {
                    webView.loadData(
                        "<html><body style='background:#07040d;color:#fff;font-family:monospace;padding:30px'><h2 style='color:#ff0055'>Server did not start</h2><p>Check logcat for errors.</p></body></html>",
                        "text/html", "UTF-8"
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
        } catch (e: Exception) { false }
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
