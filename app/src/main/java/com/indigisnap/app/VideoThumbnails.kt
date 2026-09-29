package com.indigisnap.app

import android.content.Context
import android.graphics.Bitmap
import android.media.MediaMetadataRetriever
import android.util.Log
import java.io.File
import java.io.FileOutputStream
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Generates video thumbnails using Android's MediaMetadataRetriever.
 * No external binary or ffmpeg dependency — works on stock Android.
 *
 * Thumbnails live in {cacheDir}/indigisnap-thumbs/, mirroring the media
 * tree under {Pictures}/IndigiSnap/. The filename is the full video
 * filename plus ".jpg", e.g. "clip.mp4" -> "clip.mp4.jpg", so clip.mp4
 * and clip.mov do not collide.
 *
 * The Go backend's /thumb/<videoRelPath> route serves from the same
 * location (see internal/handlers/thumb.go and internal/paths).
 */
object VideoThumbnails {
    private const val TAG = "IndigiSnap.VideoThumb"
    private const val FRAME_PERCENT = 10
    private const val JPEG_QUALITY = 75
    private const val SUBFOLDER = "indigisnap-thumbs"

    private val scanRunning = AtomicBoolean(false)

    /**
     * Returns the thumbnail file for a video, given the base media dir and
     * the cache dir. Both are needed because we mirror the tree structure.
     *
     * videoFile must be inside baseDir.
     */
    fun thumbFileFor(cacheDir: File, baseDir: File, videoFile: File): File? {
        val base = baseDir.canonicalFile
        val video = videoFile.canonicalFile
        if (!video.path.startsWith(base.path + File.separator)) {
            Log.w(TAG, "video outside base dir: " + video.path)
            return null
        }
        val rel = video.path.substring(base.path.length + 1)
        return File(File(cacheDir, SUBFOLDER), rel + ".jpg")
    }

    /**
     * Generates a thumbnail for a single video file at 10% of its duration.
     * Writes to {cacheDir}/indigisnap-thumbs/<relPath>.jpg.
     * Returns true on success.
     */
    fun generate(cacheDir: File, baseDir: File, videoFile: File): Boolean {
        if (!videoFile.exists() || videoFile.length() == 0L) return false
        if (!isVideoExtension(videoFile.extension)) return false

        val thumbFile = thumbFileFor(cacheDir, baseDir, videoFile) ?: return false

        val retriever = MediaMetadataRetriever()
        return try {
            retriever.setDataSource(videoFile.absolutePath)
            val durationMs = retriever
                .extractMetadata(MediaMetadataRetriever.METADATA_KEY_DURATION)
                ?.toLongOrNull() ?: 0L
            val frameUs = (durationMs * 1000L * FRAME_PERCENT) / 100L
            val bitmap = retriever.getFrameAtTime(
                frameUs,
                MediaMetadataRetriever.OPTION_CLOSEST_SYNC
            ) ?: return false

            val parent = thumbFile.parentFile
            if (parent != null && !parent.exists() && !parent.mkdirs()) {
                Log.w(TAG, "cannot create thumb dir " + parent.absolutePath)
                bitmap.recycle()
                return false
            }

            FileOutputStream(thumbFile).use { out ->
                bitmap.compress(Bitmap.CompressFormat.JPEG, JPEG_QUALITY, out)
            }
            bitmap.recycle()
            Log.i(TAG, "generated " + thumbFile.absolutePath)
            true
        } catch (e: Exception) {
            Log.w(TAG, "generate failed for " + videoFile.name + ": " + e.message)
            false
        } finally {
            try { retriever.release() } catch (e: Exception) { /* ignored */ }
        }
    }

    /**
     * Walks baseDir and generates any missing thumbnails in cacheDir.
     * Skips if a scan is already in progress.
     */
    fun generateMissingIn(cacheDir: File, baseDir: File) {
        if (!scanRunning.compareAndSet(false, true)) {
            Log.d(TAG, "scan already in progress, skipping")
            return
        }
        try {
            val start = System.currentTimeMillis()
            var generated = 0
            var checked = 0
            walk(baseDir) { file ->
                if (file.isFile && isVideoExtension(file.extension)) {
                    checked++
                    val thumb = thumbFileFor(cacheDir, baseDir, file) ?: return@walk
                    if (!thumb.exists()) {
                        if (generate(cacheDir, baseDir, file)) generated++
                    }
                }
            }
            val elapsed = System.currentTimeMillis() - start
            Log.i(TAG, "scan complete: checked=" + checked + " generated=" + generated + " in " + elapsed + "ms")
        } finally {
            scanRunning.set(false)
        }
    }

    private fun isVideoExtension(ext: String?): Boolean {
        if (ext == null) return false
        return when (ext.lowercase()) {
            "mp4", "mov", "m4v", "webm", "mkv", "3gp", "avi" -> true
            else -> false
        }
    }

    private fun walk(dir: File, action: (File) -> Unit) {
        val entries = dir.listFiles() ?: return
        for (entry in entries) {
            if (entry.isDirectory) {
                if (entry.name.startsWith(".")) continue
                walk(entry, action)
            } else {
                action(entry)
            }
        }
    }
}
