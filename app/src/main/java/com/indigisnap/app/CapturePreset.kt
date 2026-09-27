package com.indigisnap.app

import android.util.Size
import androidx.camera.core.ImageCapture

/**
 * Capture presets for CameraX.
 *
 * Each preset controls JPEG quality, capture mode, and (optionally) an
 * upper-bound resolution cap. Presets are persisted in SharedPreferences
 * under the key defined in [CapturePreset.PREFS_KEY].
 */
enum class CapturePreset(
    val key: String,
    val displayName: String,
    val description: String,
    val jpegQuality: Int,
    val captureMode: Int,
    val targetResolution: Size?,
) {
    HIGHEST(
        key = "highest",
        displayName = "Highest Quality",
        description = "Maximum JPEG quality, slower capture",
        jpegQuality = 100,
        captureMode = ImageCapture.CAPTURE_MODE_MAXIMIZE_QUALITY,
        targetResolution = null,
    ),
    BALANCED(
        key = "balanced",
        displayName = "Balanced",
        description = "Default. Good quality, fast capture",
        jpegQuality = 90,
        captureMode = ImageCapture.CAPTURE_MODE_MINIMIZE_LATENCY,
        targetResolution = null,
    ),
    STORAGE_SAVER(
        key = "saver",
        displayName = "Storage Saver",
        description = "Smaller files, reduced resolution",
        jpegQuality = 75,
        captureMode = ImageCapture.CAPTURE_MODE_MINIMIZE_LATENCY,
        targetResolution = Size(1920, 1440),
    ),
    FAST(
        key = "fast",
        displayName = "Fast Capture",
        description = "Smallest files, fastest capture",
        jpegQuality = 60,
        captureMode = ImageCapture.CAPTURE_MODE_MINIMIZE_LATENCY,
        targetResolution = Size(1920, 1440),
    );

    companion object {
        const val PREFS_FILE = "indigisnap_camera"
        const val PREFS_KEY = "capture_preset"

        val DEFAULT = BALANCED

        fun fromKey(k: String?): CapturePreset =
            entries.firstOrNull { it.key == k } ?: DEFAULT
    }
}
