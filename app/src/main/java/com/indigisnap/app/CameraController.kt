package com.indigisnap.app

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.hardware.camera2.CameraCharacteristics
import android.hardware.camera2.CameraManager
import android.os.Handler
import android.os.Looper
import android.util.Log
import android.util.Size
import androidx.camera.core.Camera
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageCapture
import androidx.camera.core.ImageCaptureException
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.core.content.ContextCompat
import androidx.lifecycle.LifecycleOwner
import java.io.File
import java.util.concurrent.ExecutorService
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit

/**
 * CameraController owns all CameraX wiring and shot-session state.
 *
 * Responsibilities:
 *   - Probe camera availability and bind CameraX to lifecycle
 *   - Manage preview stream warm-up detection
 *   - Capture photos to .inbox/<session-id>/ with verification
 *   - Maintain the ordered list of SessionShot in this session
 *   - Handle front/back switch, flash mode, pinch zoom
 *   - Manage shutter state (isCapturing, isSaving, cooldown)
 *   - Signal fallback to legacy camera on unrecoverable failure
 *
 * Commit and discard are handled by CameraActivity. CameraController only
 * manages in-session state.
 */
class CameraController(
    private val context: Context,
    private val lifecycleOwner: LifecycleOwner,
    private val previewView: PreviewView,
    private val sessionId: String,
    private val inboxDir: File,
    private val listener: Listener,
    initialPreset: CapturePreset = CapturePreset.DEFAULT,
) {

    interface Listener {
        fun onShotAdded(shot: SessionShot)
        fun onShotsChanged(shots: List<SessionShot>)
        fun onShotFailed(reason: String)
        fun onShutterStateChanged(state: ShutterState)
        fun onCaptureStarted()
        fun onCameraUnavailable(reason: String)
        fun onPreviewReady()
        fun onCameraStateChanged(lensFacing: Int, flashMode: Int)
    }

    companion object {
        private const val TAG = "IndigiSnap.CameraCtl"
        private const val BIND_TIMEOUT_MS = 3000L
        private const val WARMUP_TIMEOUT_MS = 2000L
        private const val SHUTTER_COOLDOWN_MS = 400L
        private const val THUMBNAIL_SIZE_PX = 256
    }

    private val mainHandler = Handler(Looper.getMainLooper())
    private val cameraExecutor: ExecutorService = Executors.newSingleThreadExecutor()
    private val bindStartMs = System.currentTimeMillis()

    private var cameraProvider: ProcessCameraProvider? = null
    private var imageCapture: ImageCapture? = null
    private var currentCamera: Camera? = null

    // Shutter state
    private var isCapturing = false
    private var isSaving = false
    private var shutterCooldown = false

    // Bind lifecycle
    private var bindSucceeded = false
    private var previewReady = false
    private var bindTimeoutFired = false
    private var warmupTimeoutFired = false

    // Camera state
    private var currentLensFacing = CameraSelector.LENS_FACING_BACK
    private var currentFlashMode = ImageCapture.FLASH_MODE_OFF
    private var currentPreset: CapturePreset = initialPreset

    // Session state
    private val shots = mutableListOf<SessionShot>()

    // ---------------------------------------------------------------
    // Public API
    // ---------------------------------------------------------------

    fun start() {
        if (!probeCameraAvailable()) {
            listener.onCameraUnavailable("ReadinessProbeFailed")
            return
        }

        mainHandler.postDelayed({
            if (!bindSucceeded && !bindTimeoutFired) {
                bindTimeoutFired = true
                Log.w(TAG, "Bind timeout after ${BIND_TIMEOUT_MS}ms")
                listener.onCameraUnavailable("BindTimeout")
            }
        }, BIND_TIMEOUT_MS)

        mainHandler.postDelayed({
            if (!previewReady && bindSucceeded && !warmupTimeoutFired) {
                warmupTimeoutFired = true
                Log.w(TAG, "Preview warm-up timeout after ${WARMUP_TIMEOUT_MS}ms")
                listener.onCameraUnavailable("PreviewWarmupTimeout")
            }
        }, BIND_TIMEOUT_MS + WARMUP_TIMEOUT_MS)

        val future = ProcessCameraProvider.getInstance(context)
        future.addListener({
            try {
                val provider = future.get()
                cameraProvider = provider
                bindUseCases(provider, currentLensFacing)
            } catch (e: SecurityException) {
                Log.e(TAG, "SecurityException on bind: ${e.message}")
                listener.onCameraUnavailable("PermissionRevoked: ${e.message}")
            } catch (e: Exception) {
                Log.e(TAG, "Bind failure: ${e.message}")
                listener.onCameraUnavailable("BindFailure: ${e.message}")
            }
        }, ContextCompat.getMainExecutor(context))
    }

    fun stop() {
        mainHandler.removeCallbacksAndMessages(null)
        cameraProvider?.unbindAll()
        cameraProvider = null
        imageCapture = null
        currentCamera = null
        cameraExecutor.shutdown()
        try {
            cameraExecutor.awaitTermination(500, TimeUnit.MILLISECONDS)
        } catch (e: InterruptedException) {
            Log.w(TAG, "executor shutdown interrupted", e)
        }
    }

    fun capturePhoto() {
        mainHandler.post {
            if (isCapturing || isSaving || shutterCooldown) {
                Log.d(TAG, "Shutter ignored: busy")
                return@post
            }
            val capture = imageCapture
            if (capture == null) {
                Log.w(TAG, "capturePhoto with no ImageCapture")
                return@post
            }

            isCapturing = true
            listener.onCaptureStarted()
            updateShutterState()

            // Re-read rotation with a fallback so targetRotation is never
            // left stale. Falls back to config orientation if display is
            // unavailable.
            val rotation = previewView.display?.rotation ?: run {
                when (context.resources.configuration.orientation) {
                    android.content.res.Configuration.ORIENTATION_PORTRAIT ->
                        android.view.Surface.ROTATION_0
                    android.content.res.Configuration.ORIENTATION_LANDSCAPE ->
                        android.view.Surface.ROTATION_90
                    else -> android.view.Surface.ROTATION_0
                }
            }
            capture.targetRotation = rotation
            Log.d(TAG, "capturePhoto: targetRotation=" + rotation)

            val outputFile = nextOutputFile()
            val options = ImageCapture.OutputFileOptions.Builder(outputFile).build()

            capture.takePicture(
                options,
                cameraExecutor,
                object : ImageCapture.OnImageSavedCallback {
                    override fun onImageSaved(output: ImageCapture.OutputFileResults) {
                        onCaptureComplete(outputFile)
                    }
                    override fun onError(exception: ImageCaptureException) {
                        Log.e(TAG, "Capture error: ${exception.message}", exception)
                        mainHandler.post {
                            isCapturing = false
                            updateShutterState()
                            listener.onShotFailed("CaptureError: ${exception.message}")
                            cleanupFailedFile(outputFile)
                        }
                    }
                }
            )
        }
    }

    fun shotsSnapshot(): List<SessionShot> = shots.toList()

    fun toggleShotDeletion(index: Int) {
        mainHandler.post {
            if (index !in shots.indices) return@post
            val old = shots[index]
            shots[index] = old.copy(markedForDeletion = !old.markedForDeletion)
            listener.onShotsChanged(shots.toList())
        }
    }

    /**
     * Batch version of toggleShotDeletion. Sets markedForDeletion on every
     * shot in [indices] to [marked]. Reversible, same as individual delete.
     * Used by the multi-select batch toolbar.
     */
    fun markShotsDeleted(indices: Set<Int>, marked: Boolean) {
        mainHandler.post {
            if (indices.isEmpty()) return@post
            var changed = false
            for (i in indices) {
                if (i !in shots.indices) continue
                val old = shots[i]
                if (old.markedForDeletion != marked) {
                    shots[i] = old.copy(markedForDeletion = marked)
                    changed = true
                }
            }
            if (changed) listener.onShotsChanged(shots.toList())
        }
    }

    fun switchCamera() {
        mainHandler.post {
            val provider = cameraProvider ?: return@post
            val newFacing = if (currentLensFacing == CameraSelector.LENS_FACING_BACK)
                CameraSelector.LENS_FACING_FRONT
            else
                CameraSelector.LENS_FACING_BACK
            try {
                bindUseCases(provider, newFacing)
            } catch (e: Exception) {
                Log.e(TAG, "switchCamera failed: ${e.message}", e)
            }
        }
    }

    fun cycleFlash() {
        mainHandler.post {
            val newMode = when (currentFlashMode) {
                ImageCapture.FLASH_MODE_OFF -> ImageCapture.FLASH_MODE_AUTO
                ImageCapture.FLASH_MODE_AUTO -> ImageCapture.FLASH_MODE_ON
                else -> ImageCapture.FLASH_MODE_OFF
            }
            currentFlashMode = newMode
            imageCapture?.flashMode = newMode
            listener.onCameraStateChanged(currentLensFacing, newMode)
        }
    }

    /**
     * Rotates a shot file on disk by the given degrees.
     * Updates the in-memory thumbnail to match.
     * Must be called from the main thread.
     */
    fun rotateShot(index: Int, degrees: Int) {
        mainHandler.post {
            if (index !in shots.indices) return@post
            if (degrees != 90 && degrees != 180 && degrees != 270) {
                Log.w(TAG, "rotateShot: invalid degrees=$degrees")
                return@post
            }
            val shot = shots[index]
            val original = shot.file
            val temp = File(original.parentFile, original.name + ".rotatetmp")

            try {
                // 1. Decode with EXIF orientation applied, so we rotate the
                //    pixels as the user currently sees them.
                val bmp = loadFullResolution(original)
                    ?: throw IllegalStateException("decode returned null")

                // 2. Apply rotation
                val matrix = Matrix()
                matrix.postRotate(degrees.toFloat())
                val rotated = Bitmap.createBitmap(
                    bmp, 0, 0, bmp.width, bmp.height, matrix, true
                )

                // 3. Encode to temp file. compress() strips EXIF, so the
                //    saved file has NORMAL orientation implicitly.
                java.io.FileOutputStream(temp).use { out ->
                    val ok = rotated.compress(Bitmap.CompressFormat.JPEG, 95, out)
                    if (!ok) throw IllegalStateException("compress returned false")
                }

                // 4. Verify temp file: exists, non-empty, decodable.
                if (!temp.exists()) throw IllegalStateException("temp missing after write")
                if (temp.length() <= 0L) throw IllegalStateException("temp is zero bytes")
                val opts = BitmapFactory.Options().apply { inJustDecodeBounds = true }
                BitmapFactory.decodeFile(temp.absolutePath, opts)
                if (opts.outWidth <= 0 || opts.outHeight <= 0) {
                    throw IllegalStateException("temp not decodable (w=${opts.outWidth} h=${opts.outHeight})")
                }

                // 5. Atomic swap: delete original, rename temp -> original.
                if (!original.delete()) {
                    throw IllegalStateException("failed to delete original")
                }
                if (!temp.renameTo(original)) {
                    throw IllegalStateException("rename temp -> original failed")
                }
                if (!original.exists()) {
                    throw IllegalStateException("original missing after rename")
                }

                // 6. Regenerate thumbnail from the swapped-in file.
                val newThumb = generateThumbnail(original)
                shots[index] = shot.copy(thumbnail = newThumb)
                listener.onShotsChanged(shots.toList())
                Log.i(TAG, "rotateShot: index=$index degrees=$degrees file=" + original.name)

            } catch (e: Exception) {
                Log.e(TAG, "rotateShot failed: " + e.message, e)
                // Clean up temp on any failure. Original remains untouched
                // unless we got past step 5 partially, in which case the
                // exception message names the exact failure point.
                try {
                    if (temp.exists()) temp.delete()
                } catch (cleanupEx: Exception) {
                    Log.w(TAG, "rotateShot cleanup failed: " + cleanupEx.message)
                }
                listener.onShotFailed("RotateFailed: " + e.message)
            }
        }
    }

    /**
     * Deletes a set of shots from the session (files + list entries).
     * Internal cleanup only. User-facing delete goes through
     * toggleShotDeletion (mark) or markShotsDeleted (batch mark),
     * which are both reversible.
     * Must be called from the main thread.
     */
    private fun removeShotsInternal(indices: Set<Int>) {
        mainHandler.post {
            if (indices.isEmpty()) return@post
            // Sort descending so removals do not shift indices
            val sorted = indices.sortedDescending()
            for (i in sorted) {
                if (i !in shots.indices) continue
                val shot = shots[i]
                try {
                    if (shot.file.exists()) shot.file.delete()
                } catch (e: Exception) {
                    Log.w(TAG, "deleteShots: file delete failed for " + shot.file.name + ": " + e.message)
                }
                shots.removeAt(i)
            }
            listener.onShotsChanged(shots.toList())
            Log.i(TAG, "deleteShots: removed " + indices.size + " shots, " + shots.size + " remaining")
        }
    }

    /**
     * Toggles the batch-selection flag on a shot.
     * Used during multi-select mode.
     */
    fun toggleBatchSelection(index: Int) {
        mainHandler.post {
            if (index !in shots.indices) return@post
            val old = shots[index]
            shots[index] = old.copy(selectedForBatch = !old.selectedForBatch)
            listener.onShotsChanged(shots.toList())
        }
    }

    /**
     * Clears all batch-selection flags. Called when exiting multi-select mode.
     */
    fun clearBatchSelection() {
        mainHandler.post {
            var changed = false
            for (i in shots.indices) {
                if (shots[i].selectedForBatch) {
                    shots[i] = shots[i].copy(selectedForBatch = false)
                    changed = true
                }
            }
            if (changed) listener.onShotsChanged(shots.toList())
        }
    }

    /**
     * Updates the ImageCapture target rotation. Call from the activity when
     * the display rotation changes (onConfigurationChanged).
     */
    fun updateTargetRotation(rotation: Int) {
        imageCapture?.targetRotation = rotation
        Log.d(TAG, "updateTargetRotation: " + rotation)
    }

    /**
     * Decodes a full-resolution Bitmap with EXIF orientation applied.
     * Used by the review overlay for fullscreen display. Callers must
     * not hold references to multiple full-size bitmaps concurrently.
     *
     * Public so CameraActivity can use it for full-size review display.
     */
    fun loadFullResolution(file: File): Bitmap? {
        return try {
            val raw = BitmapFactory.decodeFile(file.absolutePath) ?: return null
            applyExifOrientation(file, raw)
        } catch (e: Exception) {
            Log.w(TAG, "loadFullResolution failed: " + e.message)
            null
        }
    }

    /**
     * Decodes a sampled-down Bitmap with EXIF orientation applied.
     *
     * Uses BitmapFactory.Options.inSampleSize to decode at roughly 2x the
     * requested target size, then scales down to fit. Memory usage is
     * bounded by targetSize rather than by the source image dimensions.
     *
     * Use this for thumbnails and any list/grid display. Do not use
     * loadFullResolution for lists.
     */
    fun loadThumbnail(file: File, targetSizePx: Int): Bitmap? {
        return try {
            // Pass 1: bounds only, no pixel allocation
            val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
            BitmapFactory.decodeFile(file.absolutePath, bounds)
            if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null

            // Compute inSampleSize: largest power of 2 such that the
            // decoded dimensions are still >= 2x target. This lets the
            // final scale-down be a clean division and avoids decoding
            // more pixels than needed.
            val target = (targetSizePx * 2).coerceAtLeast(1)
            var sample = 1
            while ((bounds.outWidth / (sample * 2)) >= target &&
                   (bounds.outHeight / (sample * 2)) >= target) {
                sample *= 2
            }

            // Pass 2: decode with inSampleSize
            val opts = BitmapFactory.Options().apply {
                inSampleSize = sample
                inPreferredConfig = Bitmap.Config.ARGB_8888
            }
            val raw = BitmapFactory.decodeFile(file.absolutePath, opts) ?: return null

            // Apply EXIF orientation
            val oriented = applyExifOrientation(file, raw) ?: return null

            // Final scale to exact target bound (long edge)
            val maxDim = maxOf(oriented.width, oriented.height).coerceAtLeast(1)
            if (maxDim <= targetSizePx) return oriented
            val scale = maxDim.toFloat() / targetSizePx.toFloat()
            val newW = (oriented.width / scale).toInt().coerceAtLeast(1)
            val newH = (oriented.height / scale).toInt().coerceAtLeast(1)
            Bitmap.createScaledBitmap(oriented, newW, newH, true)
        } catch (e: Exception) {
            Log.w(TAG, "loadThumbnail failed: " + e.message)
            null
        }
    }

    /**
     * Applies EXIF orientation to a decoded bitmap. Returns the input
     * bitmap if orientation is NORMAL or the tag is missing/unknown.
     * Returns null if the orientation matrix application fails.
     */
    private fun applyExifOrientation(file: File, raw: Bitmap): Bitmap? {
        return try {
            val exif = androidx.exifinterface.media.ExifInterface(file.absolutePath)
            val orientation = exif.getAttributeInt(
                androidx.exifinterface.media.ExifInterface.TAG_ORIENTATION,
                androidx.exifinterface.media.ExifInterface.ORIENTATION_NORMAL
            )
            val matrix = android.graphics.Matrix()
            when (orientation) {
                androidx.exifinterface.media.ExifInterface.ORIENTATION_ROTATE_90 ->
                    matrix.postRotate(90f)
                androidx.exifinterface.media.ExifInterface.ORIENTATION_ROTATE_180 ->
                    matrix.postRotate(180f)
                androidx.exifinterface.media.ExifInterface.ORIENTATION_ROTATE_270 ->
                    matrix.postRotate(270f)
                androidx.exifinterface.media.ExifInterface.ORIENTATION_FLIP_HORIZONTAL -> {
                    matrix.postScale(-1f, 1f)
                }
                androidx.exifinterface.media.ExifInterface.ORIENTATION_FLIP_VERTICAL -> {
                    matrix.postScale(1f, -1f)
                }
                androidx.exifinterface.media.ExifInterface.ORIENTATION_TRANSPOSE -> {
                    matrix.postRotate(90f)
                    matrix.postScale(-1f, 1f)
                }
                androidx.exifinterface.media.ExifInterface.ORIENTATION_TRANSVERSE -> {
                    matrix.postRotate(270f)
                    matrix.postScale(-1f, 1f)
                }
                else -> return raw
            }
            Bitmap.createBitmap(raw, 0, 0, raw.width, raw.height, matrix, true)
        } catch (e: Exception) {
            Log.w(TAG, "applyExifOrientation failed: " + e.message)
            null
        }
    }

    fun isBusy(): Boolean = isCapturing || isSaving

    /**
     * Applies a zoom ratio, clamped to the camera supported range.
     * Safe to call from the gesture handler on the main thread.
     */
    fun setZoomRatio(ratio: Float) {
        mainHandler.post {
            try {
                val cam = currentCamera ?: return@post
                val state = cam.cameraInfo.zoomState.value
                val minRatio = state?.minZoomRatio ?: 1.0f
                val maxRatio = state?.maxZoomRatio ?: 1.0f
                val clamped = ratio.coerceIn(minRatio, maxRatio)
                cam.cameraControl.setZoomRatio(clamped)
            } catch (e: Exception) {
                Log.w(TAG, "setZoomRatio failed: ${e.message}")
            }
        }
    }

    fun currentZoomRatio(): Float {
        return try {
            currentCamera?.cameraInfo?.zoomState?.value?.zoomRatio ?: 1.0f
        } catch (e: Exception) {
            1.0f
        }
    }

    fun zoomBounds(): Pair<Float, Float> {
        val state = try {
            currentCamera?.cameraInfo?.zoomState?.value
        } catch (e: Exception) {
            null
        }
        val min = state?.minZoomRatio ?: 1.0f
        val max = state?.maxZoomRatio ?: 1.0f
        return Pair(min, max)
    }

    /**
     * Requests AF/AE at a point in preview view coordinates.
     * The callback fires on the main thread with true on success,
     * false on any failure (canceled, no camera, exception).
     */
    fun focusAt(viewX: Float, viewY: Float, callback: (Boolean) -> Unit) {
        mainHandler.post {
            val cam = currentCamera
            if (cam == null) {
                callback(false)
                return@post
            }
            try {
                val factory = previewView.meteringPointFactory
                val point = factory.createPoint(viewX, viewY)
                val action = androidx.camera.core.FocusMeteringAction
                    .Builder(point)
                    .setAutoCancelDuration(3, TimeUnit.SECONDS)
                    .build()
                val future = cam.cameraControl.startFocusAndMetering(action)
                future.addListener({
                    val ok = try {
                        future.get()
                        true
                    } catch (e: Exception) {
                        Log.d(TAG, "focus failed: " + e.message)
                        false
                    }
                    mainHandler.post { callback(ok) }
                }, ContextCompat.getMainExecutor(context))
            } catch (e: Exception) {
                Log.w(TAG, "focusAt exception: " + e.message)
                callback(false)
            }
        }
    }

    /**
     * Returns the current capture preset. Used by the Activity to render
     * the preset picker with the correct selection highlighted.
     */
    fun currentPreset(): CapturePreset = currentPreset

    /**
     * Changes the capture preset and rebinds the ImageCapture use case.
     *
     * Rebind is required because JPEG quality and target resolution are
     * builder-time configuration on ImageCapture (CameraX 1.3.x does not
     * support changing them at runtime on a live use case).
     *
     * Preview blink during rebind (~100-200ms) is expected and acceptable
     * for a user-initiated settings change.
     */
    fun setPreset(preset: CapturePreset) {
        mainHandler.post {
            if (preset == currentPreset) return@post
            currentPreset = preset
            val provider = cameraProvider ?: return@post
            try {
                bindUseCases(provider, currentLensFacing)
                Log.i(TAG, "setPreset: " + preset.key)
            } catch (e: Exception) {
                Log.e(TAG, "setPreset rebind failed: " + e.message, e)
            }
        }
    }

    // ---------------------------------------------------------------
    // Internal
    // ---------------------------------------------------------------

    private fun probeCameraAvailable(): Boolean {
        return try {
            val cm = context.getSystemService(Context.CAMERA_SERVICE) as CameraManager
            val ids = cm.cameraIdList
            if (ids.isEmpty()) return false
            var hasFacing = false
            for (id in ids) {
                val chars = cm.getCameraCharacteristics(id)
                val facing = chars.get(CameraCharacteristics.LENS_FACING)
                if (facing == CameraCharacteristics.LENS_FACING_BACK ||
                    facing == CameraCharacteristics.LENS_FACING_FRONT) {
                    hasFacing = true
                    break
                }
            }
            hasFacing
        } catch (e: Exception) {
            Log.w(TAG, "Probe failed: ${e.message}")
            false
        }
    }

    /**
     * Builds an ImageCapture use case configured from the current preset.
     * Flash mode is preserved across preset changes.
     */
    private fun buildImageCapture(): ImageCapture {
        val builder = ImageCapture.Builder()
            .setCaptureMode(currentPreset.captureMode)
            .setJpegQuality(currentPreset.jpegQuality)
            .setFlashMode(currentFlashMode)

        // Using deprecated target resolution as a simple upper-bound cap.
        // Can be migrated to ResolutionSelector in a future CameraX upgrade.
        currentPreset.targetResolution?.let { size ->
            builder.setTargetResolution(size)
        }

        return builder.build()
    }

    private fun bindUseCases(provider: ProcessCameraProvider, lensFacing: Int) {
        try {
            provider.unbindAll()

            val preview = Preview.Builder().build().also {
                it.setSurfaceProvider(previewView.surfaceProvider)
            }

            val capture = buildImageCapture()

            val selector = CameraSelector.Builder()
                .requireLensFacing(lensFacing)
                .build()

            val camera = provider.bindToLifecycle(lifecycleOwner, selector, preview, capture)

            imageCapture = capture
            currentCamera = camera
            currentLensFacing = lensFacing
            bindSucceeded = true

            // Set targetRotation at bind time (defaulting to portrait if
            // display isn't ready yet) and keep it in sync via a layout
            // listener that fires once the surface is attached.
            capture.targetRotation = previewView.display?.rotation
                ?: android.view.Surface.ROTATION_0

            previewView.addOnLayoutChangeListener { _, _, _, _, _, _, _, _, _ ->
                previewView.display?.let { display ->
                    imageCapture?.targetRotation = display.rotation
                    Log.d(TAG, "Layout: targetRotation=" + display.rotation)
                }
            }
            Log.i(TAG, "CameraX bound (lens=$lensFacing) in ${System.currentTimeMillis() - bindStartMs}ms")

            previewView.previewStreamState.observe(lifecycleOwner) { state ->
                if (state == PreviewView.StreamState.STREAMING && !previewReady) {
                    previewReady = true
                    Log.i(TAG, "Preview streaming after ${System.currentTimeMillis() - bindStartMs}ms")
                    listener.onPreviewReady()
                }
            }

            listener.onCameraStateChanged(currentLensFacing, currentFlashMode)
            updateShutterState()
        } catch (e: Exception) {
            Log.e(TAG, "bindUseCases failed: ${e.message}", e)
            listener.onCameraUnavailable("BindUseCasesFailed: ${e.message}")
        }
    }

    private fun onCaptureComplete(outputFile: File) {
        val verified = verifyCapture(outputFile)

        mainHandler.post {
            isCapturing = false

            if (verified) {
                val thumb = generateThumbnail(outputFile)
                val shot = SessionShot(
                    file = outputFile,
                    capturedAt = System.currentTimeMillis(),
                    markedForDeletion = false,
                    thumbnail = thumb,
                )
                shots.add(shot)
                listener.onShotAdded(shot)
                listener.onShotsChanged(shots.toList())
                startShutterCooldown()
            } else {
                cleanupFailedFile(outputFile)
                listener.onShotFailed("VerificationFailed")
                updateShutterState()
            }
        }
    }

    private fun verifyCapture(file: File): Boolean {
        if (!file.exists()) return false
        if (file.length() <= 0L) return false
        val opts = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeFile(file.absolutePath, opts)
        return opts.outWidth > 0 && opts.outHeight > 0
    }

    private fun generateThumbnail(file: File): Bitmap? {
        // loadThumbnail already applies EXIF orientation and downscales
        // to THUMBNAIL_SIZE_PX. Memory bounded by target size regardless
        // of source resolution.
        return loadThumbnail(file, THUMBNAIL_SIZE_PX)
    }

    private fun cleanupFailedFile(file: File) {
        try {
            if (file.exists()) file.delete()
        } catch (e: Exception) {
            Log.w(TAG, "cleanupFailedFile: ${e.message}")
        }
    }

    private fun startShutterCooldown() {
        shutterCooldown = true
        updateShutterState()
        mainHandler.postDelayed({
            shutterCooldown = false
            updateShutterState()
        }, SHUTTER_COOLDOWN_MS)
    }

    private fun updateShutterState() {
        val state = when {
            !bindSucceeded -> ShutterState.DISABLED
            isCapturing || isSaving -> ShutterState.DISABLED
            shutterCooldown -> ShutterState.COOLDOWN
            else -> ShutterState.READY
        }
        listener.onShutterStateChanged(state)
    }

    private fun nextOutputFile(): File {
        if (!inboxDir.exists()) inboxDir.mkdirs()
        val ts = System.currentTimeMillis()
        val seq = shots.size + 1
        return File(inboxDir, "IMG_${ts}_$seq.jpg")
    }
}
