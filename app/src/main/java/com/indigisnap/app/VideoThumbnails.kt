package com.indigisnap.app

import android.graphics.Bitmap
import android.media.MediaMetadataRetriever
import android.util.Log
import java.io.File
import java.io.FileOutputStream
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Generates video thumbnails using Android's MediaMetadataRetriever.
 * No external binary or ffmpeg dependency.
 *
 * Thumbnails are written to {video.parent}/.thumbs/{nameWithoutExt}.jpg,
 * matching the convention that the Go backend already reads in
 * browse.go's findVideoThumbURL.
 */
object VideoThumbnails {
    private const val TAG = "IndigiSnap.VideoThumb"
    private const val FRAME_PERCENT = 10
    private const val JPEG_QUALITY = 75

    private val scanRunning = AtomicBoolean(false)

    /**
     * Generates a thumbnail for a single video file at 10% of its duration.
     * Returns true on success, false if generation failed or the input is
     * not a valid video.
     */
    fun generate(videoFile: File): Boolean {
        if (!videoFile.exists() || videoFile.length() == 0L) return false
        if (!isVideoExtension(videoFile.extension)) return false

        val retriever = MediaMetadataRetriever()
        return try {
            retriever.setDataSource(videoFile.absolutePath)
            val durationMs = retriever
                .extractMetadata(MediaMetadataRetriever.METADATA_KEY_DURATION)
                ?.toLongOrNull() ?: 0L
            // 10 percent into the video, in microseconds
            val frameUs = (durationMs * 1000L * FRAME_PERCENT) / 100L
            val bitmap = retriever.getFrameAtTime(
                frameUs,
                MediaMetadataRetriever.OPTION_CLOSEST_SYNC
            ) ?: return false

            val thumbsDir = File(videoFile.parentFile, ".thumbs")
            if (!thumbsDir.exists() && !thumbsDir.mkdirs()) {
                Log.w(TAG, "cannot create .thumbs dir at " + thumbsDir.absolutePath)
                bitmap.recycle()
                return false
            }
            val thumbFile = File(thumbsDir, videoFile.nameWithoutExtension + ".jpg")

            FileOutputStream(thumbFile).use { out ->
                bitmap.compress(Bitmap.CompressFormat.JPEG, JPEG_QUALITY, out)
            }
            bitmap.recycle()
            Log.i(TAG, "generated " + thumbFile.name)
            true
        } catch (e: Exception) {
            Log.w(TAG, "generate failed for " + videoFile.name + ": " + e.message)
            false
        } finally {
            try { retriever.release() } catch (e: Exception) { /* ignored */ }
        }
    }

    /**
     * Walks a directory tree and generates any missing video thumbnails.
     * Skips if another scan is already in progress. Runs on whatever
     * thread the caller provides (callers should use a background thread).
     */
    fun generateMissingIn(rootDir: File) {
        if (!scanRunning.compareAndSet(false, true)) {
            Log.d(TAG, "scan already in progress, skipping")
            return
        }
        try {
            val start = System.currentTimeMillis()
            var generated = 0
            var checked = 0
            walk(rootDir) { file ->
                if (file.isFile && isVideoExtension(file.extension)) {
                    checked++
                    val thumbsDir = File(file.parentFile, ".thumbs")
                    val thumbFile = File(thumbsDir, file.nameWithoutExtension + ".jpg")
                    if (!thumbFile.exists()) {
                        if (generate(file)) generated++
                    }
                }
            }
            val elapsed = System.currentTimeMillis() - start
            Log.i(TAG, "scan complete: checked=$checked generated=$generated in ${elapsed}ms")
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
            // Skip hidden directories to avoid recursing into .thumbs, .inbox metadata
            if (entry.isDirectory) {
                if (entry.name.startsWith(".")) continue
                walk(entry, action)
            } else {
                action(entry)
            }
        }
    }
}
