package com.indigisnap.app

import android.content.Context
import android.net.Uri
import android.os.Environment
import android.provider.DocumentsContract
import android.util.Log
import java.io.File

/**
 * Central storage configuration for IndigiSnap.
 *
 * On first launch, the user picks a folder via SAF (Storage Access
 * Framework). We extract the real filesystem path from the SAF URI so
 * the Go backend (which uses os.Open / os.Rename) can work with it
 * directly.
 *
 * Only primary external storage is supported. SD cards and cloud
 * providers return null from extractRealPath() and the picker reopens.
 *
 * Storage:
 *   - SharedPreferences "indigisnap_storage"
 *   - key "tree_uri"   : the persisted SAF URI (for re-requesting access)
 *   - key "real_path"  : the extracted /storage/emulated/0/... path
 */
object StorageConfig {
    private const val TAG = "IndigiSnap.Storage"
    private const val PREFS = "indigisnap_storage"
    private const val KEY_TREE_URI = "tree_uri"
    private const val KEY_REAL_PATH = "real_path"

    /** Returns true if a base dir has been picked and is still resolvable. */
    fun hasBaseDir(ctx: Context): Boolean {
        val path = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY_REAL_PATH, null) ?: return false
        val f = File(path)
        return f.exists() && f.isDirectory
    }

    /**
     * Returns the base directory as a File, or null if not configured.
     */
    fun baseDirFile(ctx: Context): File? {
        val path = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY_REAL_PATH, null) ?: return null
        return File(path)
    }

    /**
     * Saves a picked SAF folder. Extracts the real path from the URI.
     * Returns true on success, false if the folder is not on primary
     * external storage (SD card or cloud).
     */
    fun saveFromUri(ctx: Context, uri: Uri): Boolean {
        val realPath = extractRealPath(uri)
        if (realPath == null) {
            Log.w(TAG, "Cannot extract real path from URI: " + uri)
            return false
        }
        val f = File(realPath)
        if (!f.exists()) {
            if (!f.mkdirs()) {
                Log.w(TAG, "Cannot create directory: " + realPath)
                return false
            }
        }
        ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_TREE_URI, uri.toString())
            .putString(KEY_REAL_PATH, realPath)
            .apply()
        Log.i(TAG, "Base dir set: " + realPath)
        return true
    }

    /** Forgets the current base dir. Used only for testing/change-folder. */
    fun clear(ctx: Context) {
        ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .remove(KEY_TREE_URI)
            .remove(KEY_REAL_PATH)
            .apply()
    }

    /**
     * Extracts a real filesystem path from a SAF tree URI.
     * Only works for primary external storage:
     *   content://com.android.externalstorage.documents/tree/primary%3ADownload%2FFoo
     *   -> /storage/emulated/0/Download/Foo
     *
     * Returns null for SD cards, cloud providers, or non-external URIs.
     */
    private fun extractRealPath(uri: Uri): String? {
        try {
            val treeId = DocumentsContract.getTreeDocumentId(uri)
            val parts = treeId.split(":", limit = 2)
            if (parts.size != 2) return null
            val authority = parts[0]
            val relativePath = parts[1]
            if (authority != "primary") {
                Log.w(TAG, "Not primary storage: " + authority)
                return null
            }
            val primary = Environment.getExternalStorageDirectory()
                ?: return null
            return if (relativePath.isEmpty()) {
                primary.absolutePath
            } else {
                File(primary, relativePath).absolutePath
            }
        } catch (e: Exception) {
            Log.w(TAG, "extractRealPath failed: " + e.message)
            return null
        }
    }
}
