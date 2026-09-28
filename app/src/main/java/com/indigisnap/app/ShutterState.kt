package com.indigisnap.app

/**
 * Shutter button state, communicated from CameraController to
 * CameraActivity via Listener.onShutterStateChanged.
 *
 *   READY     - tappable, green accent
 *   COOLDOWN  - brief post-capture lockout, amber accent
 *   DISABLED  - camera busy or unavailable, dim
 *
 * Color mapping is applied in CameraActivity; this enum is UI-agnostic.
 */
enum class ShutterState {
    READY,
    COOLDOWN,
    DISABLED,
}
