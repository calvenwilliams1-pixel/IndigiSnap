package com.indigisnap.app

import android.content.Context
import android.graphics.Color
import android.graphics.drawable.Drawable
import android.graphics.drawable.GradientDrawable
import android.graphics.drawable.LayerDrawable
import android.util.TypedValue
import android.view.Gravity
import android.widget.Button
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import androidx.core.content.ContextCompat

/**
 * CameraTheme centralizes colors, sizes, and reusable view builders for
 * CameraActivity and the review overlay.
 *
 * Color roles (locked, see MASTER_PROMPT):
 *   - Magenta #FF00FF: theme / borders / icons
 *   - Green   #00FF99: counters / accents / glows
 *   - Amber   #FFB000: cooldown / busy
 *   - Red     #FF3355: delete / destructive
 *   - BG      #07040D: background
 *   - Surface #110822: panels, dialogs
 */
object CameraTheme {
    const val MAGENTA = 0xFFFF00FF.toInt()
    const val GREEN = 0xFF00FF99.toInt()
    const val AMBER = 0xFFFFB000.toInt()
    const val RED = 0xFFFF3355.toInt()
    const val BG = 0xFF07040D.toInt()
    const val SURFACE = 0xFF110822.toInt()
    const val ICON_NEUTRAL = 0xFFFFFFFF.toInt()
    const val ICON_DIM = 0x99FFFFFF.toInt()
    const val OVERLAY_BG = 0x99000000.toInt()

    fun dp(ctx: Context, dp: Int): Int = TypedValue.applyDimension(
        TypedValue.COMPLEX_UNIT_DIP, dp.toFloat(), ctx.resources.displayMetrics
    ).toInt()

    /**
     * Holder for an icon button so callers can swap the icon at runtime
     * (e.g. flash cycling) without re-walking the view hierarchy.
     */
    class IconButton(val container: FrameLayout, val icon: ImageView)

    /**
     * Icon button: circular container with a tinted vector icon centered.
     * Replaces text-emoji Button styling. `tint` defaults to neutral white.
     */
    fun makeIconButton(ctx: Context, iconResId: Int, tint: Int = ICON_NEUTRAL): IconButton {
        val size = dp(ctx, 48)
        val container = FrameLayout(ctx).apply {
            layoutParams = FrameLayout.LayoutParams(size, size)
            isClickable = true
            isFocusable = true
            background = GradientDrawable().apply {
                shape = GradientDrawable.OVAL
                setColor(OVERLAY_BG)
            }
            setPadding(dp(ctx, 12), dp(ctx, 12), dp(ctx, 12), dp(ctx, 12))
        }
        val icon = ImageView(ctx).apply {
            layoutParams = FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            ).apply { gravity = Gravity.CENTER }
            setImageResource(iconResId)
            setColorFilter(tint)
        }
        container.addView(icon)
        return IconButton(container, icon)
    }

    /**
     * Replaces the icon and tint on an existing IconButton.
     */
    fun updateIcon(button: IconButton, iconResId: Int, tint: Int = ICON_NEUTRAL) {
        button.icon.setImageResource(iconResId)
        button.icon.setColorFilter(tint)
    }

    /**
     * Pill button for labeled actions (Delete, Mark, Commit Selected, Cancel).
     */
    fun makePillButton(ctx: Context, label: String, tint: Int = ICON_NEUTRAL): Button {
        return Button(ctx).apply {
            text = label
            setBackgroundColor(OVERLAY_BG)
            setTextColor(tint)
            alpha = 0.95f
            setPadding(dp(ctx, 12), dp(ctx, 8), dp(ctx, 12), dp(ctx, 8))
            minWidth = 0
            minimumWidth = 0
            minHeight = 0
            minimumHeight = 0
        }
    }

    /**
     * Layered glow background for the shutter ring. Uses two concentric
     * ovals rather than setElevation, per working rules (no elevation for
     * glow because it doesn't render reliably across vendors).
     */
    fun makeShutterRing(ctx: Context, color: Int, outerSizePx: Int, ringWidthPx: Int): Drawable {
        val outer = GradientDrawable().apply {
            shape = GradientDrawable.OVAL
            setColor(Color.TRANSPARENT)
            setStroke(ringWidthPx, color)
        }
        val glow = GradientDrawable().apply {
            shape = GradientDrawable.OVAL
            setColor(Color.TRANSPARENT)
            setStroke(ringWidthPx + dp(ctx, 4), (color and 0x00FFFFFF) or 0x33000000)
        }
        return LayerDrawable(arrayOf(glow, outer))
    }

    /**
     * Thumbnail frame: square container with a subtle border that changes
     * tint based on the shot's state (normal/marked/selected).
     */
    fun makeThumbFrame(ctx: Context, sizePx: Int, borderColor: Int): FrameLayout {
        val container = FrameLayout(ctx).apply {
            layoutParams = LinearLayout.LayoutParams(sizePx, sizePx).apply {
                marginEnd = dp(ctx, 8)
            }
            background = GradientDrawable().apply {
                shape = GradientDrawable.RECTANGLE
                setStroke(dp(ctx, 2), borderColor)
                setColor(Color.TRANSPARENT)
            }
        }
        return container
    }
}
