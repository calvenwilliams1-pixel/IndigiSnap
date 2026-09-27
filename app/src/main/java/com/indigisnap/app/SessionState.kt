package com.indigisnap.app

import android.os.Parcel
import android.os.Parcelable
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * SessionState carries the payload for a camera capture session.
 *
 * Passed:
 *   - MainActivity -> CameraActivity via Intent extras (initial state)
 *   - CameraActivity -> MainActivity via setResult (final state)
 *   - MainActivity -> legacy camera flow (fallback path)
 *
 * `folder` is the destination folder the session was launched from.
 * Captured files move from .inbox/<sessionId>/ to {BaseDir}/{folder}/ on commit.
 */
data class SessionState(
    val sessionId: String,
    val folder: String = "",
    val shotCount: Int = 0,
    val lastShotPath: String? = null,
    val sessionFinished: Boolean = false,
    val fallbackUsed: Boolean = false,
    val failureReason: String? = null,
    val cameraStartTimeMs: Long = 0L,
) : Parcelable {

    constructor(parcel: Parcel) : this(
        sessionId = parcel.readString() ?: "",
        folder = parcel.readString() ?: "",
        shotCount = parcel.readInt(),
        lastShotPath = parcel.readString(),
        sessionFinished = parcel.readInt() == 1,
        fallbackUsed = parcel.readInt() == 1,
        failureReason = parcel.readString(),
        cameraStartTimeMs = parcel.readLong(),
    )

    override fun writeToParcel(parcel: Parcel, flags: Int) {
        parcel.writeString(sessionId)
        parcel.writeString(folder)
        parcel.writeInt(shotCount)
        parcel.writeString(lastShotPath)
        parcel.writeInt(if (sessionFinished) 1 else 0)
        parcel.writeInt(if (fallbackUsed) 1 else 0)
        parcel.writeString(failureReason)
        parcel.writeLong(cameraStartTimeMs)
    }

    override fun describeContents(): Int = 0

    companion object {
        @JvmField
        val CREATOR: Parcelable.Creator<SessionState> = object : Parcelable.Creator<SessionState> {
            override fun createFromParcel(parcel: Parcel): SessionState = SessionState(parcel)
            override fun newArray(size: Int): Array<SessionState?> = arrayOfNulls(size)
        }

        fun newSessionId(): String {
            val sdf = SimpleDateFormat("yyyyMMdd_HHmmss", Locale.US)
            return sdf.format(Date())
        }
    }
}
