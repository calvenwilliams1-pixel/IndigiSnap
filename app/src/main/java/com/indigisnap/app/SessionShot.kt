package com.indigisnap.app

import android.graphics.Bitmap
import java.io.File

/**
 * SessionShot represents a single photo captured during a camera session.
 *
 * A shot exists on disk from the moment the shutter fires. It remains in the
 * session until either (a) the user commits the session (moves non-deleted
 * shots to the destination folder), or (b) the user discards the session
 * (deletes all shots including deleted ones).
 *
 * Flags:
 *   - markedForDeletion: user intent to discard this shot on commit
 *   - selectedForBatch:  user selected this shot in multi-select mode
 *                        (used for batch delete operations)
 *
 * Both flags can be set independently; they serve different workflows.
 */
data class SessionShot(
    val file: File,
    val capturedAt: Long,
    val markedForDeletion: Boolean = false,
    val thumbnail: Bitmap? = null,
    val selectedForBatch: Boolean = false,
)
