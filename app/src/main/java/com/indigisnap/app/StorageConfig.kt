package com.indigisnap.app

import android.content.Context
import android.os.Environment
import android.util.Log
import java.io.File

/**
 * Storage configuration for IndigiSnap.
 *
 * v1 uses a fixed folder: {Pictures}/IndigiSnap/. All media lives there.
 * Metadata and thumbnails live in app-private storage.
 *
 * Android 10+ scoped storage rules apply:
 *   - App can create, read, modify, delete its own media files here
 *   - App can read but not modify files created by other apps
 *   - App cannot create non-media files (.json, .txt, dotfiles)
 */
object StorageConfig {
    private const val TAG = "IndigiSnap.Storage"
    private const val PREFS = "indigisnap_storage"
    private const val KEY_REAL_PATH = "real_path"
    private const val SUBFOLDER = "IndigiSnap"

    /** Returns the base directory as a File, or null if not configured. */
    fun baseDirFile(ctx: Context): File? {
        val path = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY_REAL_PATH, null) ?: return null
        return File(path)
    }

    /** True if a base dir is configured and it exists. */
    fun hasBaseDir(ctx: Context): Boolean {
        val f = baseDirFile(ctx) ?: return false
        return f.exists() && f.isDirectory
    }

    /**
     * Ensures the base dir exists at {Pictures}/IndigiSnap/. Creates it if
     * missing. Saves the resolved path in SharedPreferences. Returns true
     * if ready, false on failure.
     */
    fun ensureBaseDir(ctx: Context): Boolean {
        val pictures = Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_PICTURES)
        if (pictures == null) {
            Log.w(TAG, "getExternalStoragePublicDirectory returned null")
            return false
        }
        val baseDir = File(pictures, SUBFOLDER)
        if (!baseDir.exists()) {
            if (!baseDir.mkdirs()) {
                Log.w(TAG, "Failed to create " + baseDir.absolutePath)
                return false
            }
        }
        ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_REAL_PATH, baseDir.absolutePath)
            .apply()
        Log.i(TAG, "Base dir: " + baseDir.absolutePath)
        return true
    }

    /** Forgets the current base dir. Used only for testing. */
    fun clear(ctx: Context) {
        ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .remove(KEY_REAL_PATH)
            .apply()
    }
}
