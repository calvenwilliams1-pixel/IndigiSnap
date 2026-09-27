package com.indigisnap.app

import android.app.Activity
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.Log
import android.util.TypedValue
import android.view.Gravity
import android.view.MotionEvent
import android.view.ScaleGestureDetector
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.FrameLayout
import android.widget.HorizontalScrollView
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageCapture
import androidx.camera.view.PreviewView
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import java.io.File

/**
 * CameraActivity hosts the CameraX preview and all session UI.
 *
 * Layout:
 *   Top row:    [X]  ...  [flash] [switch] [done]
 *   Center:     PreviewView (full)
 *   Bottom:     [shot counter]
 *               [thumbnail strip]
 *               [shutter button]
 *
 * Review overlay (activated by tapping a thumbnail):
 *   Full-size image, swipe to navigate, delete/restore toggle, close.
 *
 * Commit: moves non-deleted shots from inbox to destination folder, renames
 *         per format, deletes inbox. Returns RESULT_OK with SessionState.
 * Discard: deletes entire inbox session, returns RESULT_CANCELED.
 */
class CameraActivity : AppCompatActivity(), CameraController.Listener {

    companion object {
        private const val TAG = "IndigiSnap.Camera"
        const val EXTRA_SESSION_ID = "session_id"
        const val EXTRA_FOLDER = "folder"
        const val RESULT_CAMERA_UNAVAILABLE = Activity.RESULT_FIRST_USER + 1
    }

    private lateinit var sessionId: String
    private lateinit var folder: String
    private lateinit var inboxDir: File
    private var controller: CameraController? = null

    // UI
    private lateinit var previewView: PreviewView
    private lateinit var rootLayout: FrameLayout
    private lateinit var shutterButton: View
    private lateinit var closeButton: Button
    private lateinit var doneButton: Button
    private lateinit var flashButton: Button
    private lateinit var switchButton: Button
    private lateinit var presetButton: Button
    private lateinit var shotCountLabel: TextView
    private lateinit var thumbnailStrip: LinearLayout
    private lateinit var thumbnailScroll: HorizontalScrollView

    // Review overlay
    private var reviewOverlay: FrameLayout? = null
    private var reviewDeleteButton: Button? = null
    private var reviewIndex: Int = -1

    // State
    private var fallbackRequested = false
    private var fallbackReason: String? = null

    private lateinit var scaleDetector: ScaleGestureDetector

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        sessionId = intent.getStringExtra(EXTRA_SESSION_ID) ?: SessionState.newSessionId()
        folder = intent.getStringExtra(EXTRA_FOLDER) ?: ""
        inboxDir = File(File(filesDir, "IndigiSnap/.inbox"), sessionId)

        Log.i(TAG, "onCreate session=$sessionId folder='$folder'")

        setupFullscreenWindow()
        buildLayout()

        val prefs = getSharedPreferences(CapturePreset.PREFS_FILE, MODE_PRIVATE)
        val initialPreset = CapturePreset.fromKey(prefs.getString(CapturePreset.PREFS_KEY, null))
        Log.i(TAG, "initial preset: " + initialPreset.key)

        controller = CameraController(
            context = this,
            lifecycleOwner = this,
            previewView = previewView,
            sessionId = sessionId,
            inboxDir = inboxDir,
            listener = this,
            initialPreset = initialPreset,
        ).also { it.start() }

        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (reviewOverlay != null) {
                    closeReviewOverlay()
                } else {
                    attemptClose()
                }
            }
        })
    }

    override fun onDestroy() {
        controller?.stop()
        controller = null
        super.onDestroy()
    }

    override fun onConfigurationChanged(newConfig: android.content.res.Configuration) {
        super.onConfigurationChanged(newConfig)
        previewView.display?.let { display ->
            controller?.updateTargetRotation(display.rotation)
        }
    }

    // ---------------------------------------------------------------
    // Layout
    // ---------------------------------------------------------------

    private fun setupFullscreenWindow() {
        WindowCompat.setDecorFitsSystemWindows(window, false)
        WindowInsetsControllerCompat(window, window.decorView).apply {
            hide(WindowInsetsCompat.Type.systemBars())
            systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
        }
    }

    private fun buildLayout() {
        rootLayout = FrameLayout(this).apply {
            setBackgroundColor(Color.parseColor("#07040D"))
        }

        // PreviewView fills the screen
        previewView = PreviewView(this).apply {
            layoutParams = FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            )
            scaleType = PreviewView.ScaleType.FILL_CENTER
        }
        rootLayout.addView(previewView)

        // Scale gesture detector for pinch zoom
        scaleDetector = ScaleGestureDetector(this, object : ScaleGestureDetector.SimpleOnScaleGestureListener() {
            override fun onScale(detector: ScaleGestureDetector): Boolean {
                val ctrl = controller ?: return false
                val current = ctrl.currentZoomRatio()
                val newRatio = current * detector.scaleFactor
                ctrl.setZoomRatio(newRatio)
                return true
            }
        })
        previewView.setOnTouchListener { _, event ->
            scaleDetector.onTouchEvent(event)
            true
        }

        buildTopControls()
        buildBottomControls()

        setContentView(rootLayout)
        refreshThumbnails(emptyList())
    }

    private fun buildTopControls() {
        // Close button (top-left)
        closeButton = makeOverlayButton("\u274C").apply {
            setOnClickListener { attemptClose() }
        }
        rootLayout.addView(closeButton, FrameLayout.LayoutParams(
            ViewGroup.LayoutParams.WRAP_CONTENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ).apply {
            gravity = Gravity.TOP or Gravity.START
            setMargins(30, 60, 0, 0)
        })

        // Right-side group: flash, switch, done
        val rightGroup = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
        }
        flashButton = makeOverlayButton("\u26A1").apply {
            setOnClickListener { controller?.cycleFlash() }
        }
        switchButton = makeOverlayButton("\uD83D\uDD04").apply {
            setOnClickListener { controller?.switchCamera() }
        }
        presetButton = makeOverlayButton("\u22EE").apply {
            setOnClickListener { showPresetPicker() }
        }
        doneButton = makeOverlayButton("\u2705").apply {
            visibility = View.GONE
            setOnClickListener { confirmCommit() }
        }
        rightGroup.addView(flashButton)
        rightGroup.addView(switchButton)
        rightGroup.addView(presetButton)
        rightGroup.addView(doneButton)

        rootLayout.addView(rightGroup, FrameLayout.LayoutParams(
            ViewGroup.LayoutParams.WRAP_CONTENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ).apply {
            gravity = Gravity.TOP or Gravity.END
            setMargins(0, 60, 30, 0)
        })
    }

    private fun buildBottomControls() {
        val bottomGroup = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
        }

        // Shot counter
        shotCountLabel = TextView(this).apply {
            setTextColor(Color.parseColor("#00FF99"))
            textSize = 14f
            gravity = Gravity.CENTER
            setPadding(0, 0, 0, 10)
        }
        bottomGroup.addView(shotCountLabel)

        // Thumbnail strip (horizontal scroll)
        thumbnailScroll = HorizontalScrollView(this).apply {
            isHorizontalScrollBarEnabled = false
            visibility = View.GONE
        }
        thumbnailStrip = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(10, 5, 10, 5)
        }
        thumbnailScroll.addView(thumbnailStrip, ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.WRAP_CONTENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ))
        bottomGroup.addView(thumbnailScroll, LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            dpToPx(72)
        ))

        // Shutter button — clean circle
        shutterButton = makeShutterButton()
        bottomGroup.addView(shutterButton, LinearLayout.LayoutParams(
            dpToPx(80), dpToPx(80)
        ).apply {
            gravity = Gravity.CENTER_HORIZONTAL
            topMargin = dpToPx(10)
        })

        rootLayout.addView(bottomGroup, FrameLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.WRAP_CONTENT
        ).apply {
            gravity = Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL
            bottomMargin = dpToPx(30)
        })
    }

    private fun makeOverlayButton(label: String): Button {
        return Button(this).apply {
            text = label
            setBackgroundColor(Color.parseColor("#99000000"))
            setTextColor(Color.WHITE)
            alpha = 0.9f
            setPadding(dpToPx(16), dpToPx(10), dpToPx(16), dpToPx(10))
        }
    }

    private fun showPresetPicker() {
        val ctrl = controller ?: return
        val current = ctrl.currentPreset()

        val labels = CapturePreset.entries.map { it.displayName }.toTypedArray()
        val checked = CapturePreset.entries.indexOf(current)

        AlertDialog.Builder(this)
            .setTitle("Capture preset")
            .setSingleChoiceItems(labels, checked) { dialog, which ->
                val chosen = CapturePreset.entries[which]
                getSharedPreferences(CapturePreset.PREFS_FILE, MODE_PRIVATE)
                    .edit()
                    .putString(CapturePreset.PREFS_KEY, chosen.key)
                    .apply()
                ctrl.setPreset(chosen)
                Log.i(TAG, "preset changed: " + chosen.key)
                dialog.dismiss()
            }
            .setNegativeButton("Cancel") { _, _ -> }
            .show()
    }

    private fun makeShutterButton(): View {
        // Outer ring view with inner filled circle, drawn via a custom Drawable.
        val size = dpToPx(80)
        val innerSize = dpToPx(64)
        val container = FrameLayout(this).apply {
            layoutParams = FrameLayout.LayoutParams(size, size)
        }

        // Outer ring
        val ring = View(this).apply {
            background = GradientDrawable().apply {
                shape = GradientDrawable.OVAL
                setStroke(dpToPx(4), Color.WHITE)
                setColor(Color.TRANSPARENT)
            }
            layoutParams = FrameLayout.LayoutParams(size, size).apply {
                gravity = Gravity.CENTER
            }
        }
        container.addView(ring)

        // Inner circle
        val inner = View(this).apply {
            background = GradientDrawable().apply {
                shape = GradientDrawable.OVAL
                setColor(Color.WHITE)
            }
            layoutParams = FrameLayout.LayoutParams(innerSize, innerSize).apply {
                gravity = Gravity.CENTER
            }
        }
        container.addView(inner)

        container.isClickable = true
        container.isFocusable = true
        container.setOnClickListener {
            controller?.capturePhoto()
        }
        return container
    }

    private fun dpToPx(dp: Int): Int {
        return TypedValue.applyDimension(
            TypedValue.COMPLEX_UNIT_DIP,
            dp.toFloat(),
            resources.displayMetrics
        ).toInt()
    }

    // ---------------------------------------------------------------
    // Thumbnail strip
    // ---------------------------------------------------------------

    private fun refreshThumbnails(shots: List<SessionShot>) {
        thumbnailStrip.removeAllViews()
        if (shots.isEmpty()) {
            thumbnailScroll.visibility = View.GONE
            shotCountLabel.text = ""
            doneButton.visibility = View.GONE
            return
        }
        thumbnailScroll.visibility = View.VISIBLE
        val nonDeleted = shots.count { !it.markedForDeletion }
        shotCountLabel.text = "$nonDeleted of " + shots.size + " to save"
        doneButton.visibility = View.VISIBLE

        shots.forEachIndexed { index, shot ->
            val thumbContainer = FrameLayout(this).apply {
                val size = dpToPx(64)
                layoutParams = LinearLayout.LayoutParams(size, size).apply {
                    marginEnd = dpToPx(8)
                }
            }

            val img = ImageView(this).apply {
                layoutParams = FrameLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.MATCH_PARENT
                )
                scaleType = ImageView.ScaleType.CENTER_CROP
                if (shot.thumbnail != null) {
                    setImageBitmap(shot.thumbnail)
                } else {
                    setBackgroundColor(Color.DKGRAY)
                }
                setOnClickListener {
                    if (batchSelectMode) {
                        controller?.toggleBatchSelection(index)
                    } else {
                        openReviewOverlay(index)
                    }
                }
                setOnLongClickListener {
                    if (!batchSelectMode) {
                        enterBatchSelectMode(index)
                        true
                    } else {
                        false
                    }
                }
            }
            thumbContainer.addView(img)

            // Red border overlay if marked for deletion
            if (shot.markedForDeletion) {
                val overlay = View(this).apply {
                    background = GradientDrawable().apply {
                        setColor(Color.parseColor("#99000000"))
                    }
                    layoutParams = FrameLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        ViewGroup.LayoutParams.MATCH_PARENT
                    )
                }
                thumbContainer.addView(overlay)

                val xLabel = TextView(this).apply {
                    text = "\u2715"
                    setTextColor(Color.RED)
                    textSize = 28f
                    gravity = Gravity.CENTER
                    layoutParams = FrameLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        ViewGroup.LayoutParams.MATCH_PARENT
                    )
                }
                thumbContainer.addView(xLabel)
            }

            // Batch-select checkbox + highlight
            if (batchSelectMode) {
                if (shot.selectedForBatch) {
                    val highlight = View(this).apply {
                        background = GradientDrawable().apply {
                            setColor(Color.parseColor("#6600FF99"))
                        }
                        layoutParams = FrameLayout.LayoutParams(
                            ViewGroup.LayoutParams.MATCH_PARENT,
                            ViewGroup.LayoutParams.MATCH_PARENT
                        )
                    }
                    thumbContainer.addView(highlight)
                }

                val checkbox = TextView(this).apply {
                    text = if (shot.selectedForBatch) "\u2611" else "\u2610"
                    setTextColor(Color.WHITE)
                    textSize = 22f
                    gravity = Gravity.TOP or Gravity.END
                    setPadding(0, 0, dpToPx(4), 0)
                    layoutParams = FrameLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        ViewGroup.LayoutParams.MATCH_PARENT
                    )
                }
                thumbContainer.addView(checkbox)
            }

            thumbnailStrip.addView(thumbContainer)
        }

        updateBatchSelectBar(shots)
    }

    // ---------------------------------------------------------------
    // Batch select mode (long-press a thumb)
    // ---------------------------------------------------------------

    private var batchSelectMode: Boolean = false

    private fun enterBatchSelectMode(startIndex: Int) {
        batchSelectMode = true
        controller?.toggleBatchSelection(startIndex)
        ensureBatchSelectBar()
    }

    private fun exitBatchSelectMode() {
        batchSelectMode = false
        controller?.clearBatchSelection()
        removeBatchSelectBar()
    }

    private fun ensureBatchSelectBar() {
        if (rootLayout.findViewWithTag<View>("batchSelectBar") != null) return

        val bar = LinearLayout(this).apply {
            tag = "batchSelectBar"
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setBackgroundColor(Color.parseColor("#EE120C1F"))
            setPadding(dpToPx(15), dpToPx(10), dpToPx(15), dpToPx(10))
            layoutParams = FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply {
                gravity = Gravity.TOP
            }
        }

        val countLabel = TextView(this).apply {
            tag = "batchSelectCount"
            setTextColor(Color.parseColor("#00FF99"))
            textSize = 16f
            text = "0 selected"
        }
        val spacer = View(this).apply {
            layoutParams = LinearLayout.LayoutParams(0, 1, 1f)
        }
        val deleteBtn = makeOverlayButton("\uD83D\uDDD1\uFE0F Delete").apply {
            setOnClickListener { confirmBatchDelete() }
        }
        val cancelBtn = makeOverlayButton("\u2715 Cancel").apply {
            setOnClickListener { exitBatchSelectMode() }
        }
        bar.addView(countLabel)
        bar.addView(spacer)
        bar.addView(deleteBtn)
        bar.addView(cancelBtn)
        rootLayout.addView(bar)
    }

    private fun removeBatchSelectBar() {
        rootLayout.findViewWithTag<View>("batchSelectBar")?.let {
            rootLayout.removeView(it)
        }
    }

    private fun updateBatchSelectBar(shots: List<SessionShot>) {
        val label = rootLayout.findViewWithTag<TextView>("batchSelectCount") ?: return
        val count = shots.count { it.selectedForBatch }
        label.text = count.toString() + " selected"
    }

    private fun confirmBatchDelete() {
        val ctrl = controller ?: return
        val shots = ctrl.shotsSnapshot()
        val selectedIndices = shots.indices.filter { shots[it].selectedForBatch }.toSet()
        if (selectedIndices.isEmpty()) {
            Toast.makeText(this, "Nothing selected", Toast.LENGTH_SHORT).show()
            return
        }
        AlertDialog.Builder(this)
            .setTitle("Delete selected?")
            .setMessage("Delete " + selectedIndices.size + " shot(s)? This cannot be undone.")
            .setPositiveButton("Delete") { _, _ ->
                ctrl.deleteShots(selectedIndices)
                exitBatchSelectMode()
            }
            .setNegativeButton("Cancel") { _, _ -> }
            .setCancelable(false)
            .show()
    }

    // ---------------------------------------------------------------
    // Review overlay (single image)
    // ---------------------------------------------------------------

    private var reviewImageView: ImageView? = null
    private var reviewMatrix: android.graphics.Matrix = android.graphics.Matrix()
    private var reviewScale: Float = 1f
    private var reviewTranslateX: Float = 0f
    private var reviewTranslateY: Float = 0f

    private fun openReviewOverlay(startIndex: Int) {
        val ctrl = controller ?: return
        val shots = ctrl.shotsSnapshot()
        if (startIndex !in shots.indices) return

        // Full state reset before anything else.
        reviewScale = 1f
        reviewTranslateX = 0f
        reviewTranslateY = 0f
        reviewMatrix.reset()
        reviewIndex = startIndex

        val overlay = FrameLayout(this).apply {
            setBackgroundColor(Color.parseColor("#FF07040D"))
            layoutParams = FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            )
            isClickable = true
            isFocusable = true
        }

        val imageView = ImageView(this).apply {
            layoutParams = FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            ).apply {
                setMargins(dpToPx(20), dpToPx(150), dpToPx(20), dpToPx(200))
            }
            scaleType = ImageView.ScaleType.MATRIX
            imageMatrix = reviewMatrix
        }
        overlay.addView(imageView)

        // Field assignment BEFORE updateReviewDisplay so that the display
        // routine has valid references.
        reviewOverlay = overlay
        reviewImageView = imageView

        // ---- Top bar ----
        val topBar = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            layoutParams = FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply {
                gravity = Gravity.TOP
                setMargins(dpToPx(20), dpToPx(60), dpToPx(20), 0)
            }
        }

        val closeBtn = makeOverlayButton("\u2190").apply {
            setOnClickListener { closeReviewOverlay() }
        }
        val counter = TextView(this).apply {
            setTextColor(Color.parseColor("#00FF99"))
            textSize = 16f
            setPadding(dpToPx(15), 0, dpToPx(15), 0)
        }
        val spacer = View(this).apply {
            layoutParams = LinearLayout.LayoutParams(0, 1, 1f)
        }
        val doneBtn = makeOverlayButton("\u2705").apply {
            setOnClickListener { confirmCommit() }
        }

        topBar.addView(closeBtn)
        topBar.addView(counter)
        topBar.addView(spacer)
        topBar.addView(doneBtn)
        overlay.addView(topBar)

        // ---- Bottom bar ----
        val bottomBar = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER
            layoutParams = FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            ).apply {
                gravity = Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL
                bottomMargin = dpToPx(80)
            }
        }

        val rotateLeft = makeOverlayButton("\u21BA").apply {
            setOnClickListener { rotateCurrent(-90) }
        }
        val deleteBtn = makeOverlayButton("\uD83D\uDDD1\uFE0F Delete").apply {
            setOnClickListener { toggleCurrentReviewShot() }
        }
        val rotateRight = makeOverlayButton("\u21BB").apply {
            setOnClickListener { rotateCurrent(90) }
        }

        bottomBar.addView(rotateLeft)
        bottomBar.addView(deleteBtn)
        bottomBar.addView(rotateRight)
        overlay.addView(bottomBar)

        reviewDeleteButton = deleteBtn

        // Attach gestures to the overlay (full-screen area).
        installReviewGestures(overlay)

        rootLayout.addView(overlay)

        // Update displayed content.
        updateReviewDisplay(counter, deleteBtn)
    }

    private fun installReviewGestures(overlay: FrameLayout) {
        val scaleDetector = android.view.ScaleGestureDetector(this,
            object : android.view.ScaleGestureDetector.SimpleOnScaleGestureListener() {
                override fun onScale(detector: android.view.ScaleGestureDetector): Boolean {
                    reviewScale = (reviewScale * detector.scaleFactor).coerceIn(1f, 5f)
                    applyReviewTransform()
                    return true
                }
            })

        val tapDetector = android.view.GestureDetector(this,
            object : android.view.GestureDetector.SimpleOnGestureListener() {
                override fun onDown(e: MotionEvent): Boolean = true

                override fun onDoubleTap(e: MotionEvent): Boolean {
                    if (reviewScale > 1f) {
                        reviewScale = 1f
                        reviewTranslateX = 0f
                        reviewTranslateY = 0f
                    } else {
                        reviewScale = 2.5f
                    }
                    applyReviewTransform()
                    return true
                }

                override fun onFling(
                    e1: MotionEvent?,
                    e2: MotionEvent,
                    vx: Float,
                    vy: Float
                ): Boolean {
                    if (reviewScale > 1f) return false
                    if (e1 == null) return false
                    val dx = e2.x - e1.x
                    if (kotlin.math.abs(dx) > 150 && kotlin.math.abs(vy) < kotlin.math.abs(vx)) {
                        if (dx > 0) navigateReview(-1) else navigateReview(1)
                        return true
                    }
                    return false
                }
            })

        var lastX = 0f
        var lastY = 0f

        overlay.setOnTouchListener { _, event ->
            scaleDetector.onTouchEvent(event)
            tapDetector.onTouchEvent(event)

            when (event.actionMasked) {
                MotionEvent.ACTION_DOWN -> {
                    lastX = event.x
                    lastY = event.y
                    true
                }
                MotionEvent.ACTION_MOVE -> {
                    if (reviewScale > 1f && event.pointerCount == 1) {
                        reviewTranslateX += event.x - lastX
                        reviewTranslateY += event.y - lastY
                        lastX = event.x
                        lastY = event.y
                        applyReviewTransform()
                    } else {
                        lastX = event.x
                        lastY = event.y
                    }
                    true
                }
                MotionEvent.ACTION_UP, MotionEvent.ACTION_CANCEL -> true
                else -> false
            }
        }
    }

    private fun applyReviewTransform() {
        val img = reviewImageView ?: return
        if (img.width <= 0 || img.height <= 0) {
            img.post { applyReviewTransform() }
            return
        }
        val drawable = img.drawable ?: return
        val bmpW = drawable.intrinsicWidth.toFloat()
        val bmpH = drawable.intrinsicHeight.toFloat()
        if (bmpW <= 0f || bmpH <= 0f) return

        val viewW = img.width.toFloat()
        val viewH = img.height.toFloat()

        // Base scale so the image fits inside the view at reviewScale=1
        val baseScale = minOf(viewW / bmpW, viewH / bmpH)
        val finalScale = baseScale * reviewScale

        // Scaled image dimensions
        val scaledW = bmpW * finalScale
        val scaledH = bmpH * finalScale

        // Center the scaled image in the view
        val centerX = (viewW - scaledW) / 2f
        val centerY = (viewH - scaledH) / 2f

        // Clamp pan offset so the image cannot be dragged entirely off-screen.
        // At scale=1, no panning allowed (offset forced to 0).
        // Above scale=1, allow panning up to the visible overflow bounds.
        val maxPanX = if (scaledW > viewW) (scaledW - viewW) / 2f else 0f
        val maxPanY = if (scaledH > viewH) (scaledH - viewH) / 2f else 0f
        val clampedX = reviewTranslateX.coerceIn(-maxPanX, maxPanX)
        val clampedY = reviewTranslateY.coerceIn(-maxPanY, maxPanY)
        reviewTranslateX = clampedX
        reviewTranslateY = clampedY

        reviewMatrix.reset()
        reviewMatrix.postScale(finalScale, finalScale)
        reviewMatrix.postTranslate(centerX + clampedX, centerY + clampedY)
        img.imageMatrix = reviewMatrix
    }

    private fun rotateCurrent(degrees: Int) {
        val ctrl = controller ?: return
        ctrl.rotateShot(reviewIndex, if (degrees < 0) 360 + degrees else degrees)
    }

    private fun updateReviewDisplay(counter: TextView, deleteBtn: Button) {
        val ctrl = controller ?: return
        val shots = ctrl.shotsSnapshot()
        if (reviewIndex !in shots.indices) { closeReviewOverlay(); return }
        val shot = shots[reviewIndex]

        try {
            val bmp = ctrl.loadOrientedBitmap(shot.file)
            if (bmp != null) {
                reviewImageView?.setImageBitmap(bmp)
            } else {
                Log.w(TAG, "updateReviewDisplay: loadOrientedBitmap returned null")
            }
        } catch (e: Exception) {
            Log.w(TAG, "decode review failed: " + e.message)
        }

        // Reset transform state and apply (idempotent at scale=1).
        reviewScale = 1f
        reviewTranslateX = 0f
        reviewTranslateY = 0f
        applyReviewTransform()

        counter.text = (reviewIndex + 1).toString() + " of " + shots.size

        if (shot.markedForDeletion) {
            deleteBtn.text = "\u21A9\uFE0F Restore"
        } else {
            deleteBtn.text = "\uD83D\uDDD1\uFE0F Delete"
        }
    }

    private fun navigateReview(direction: Int) {
        val ctrl = controller ?: return
        val shots = ctrl.shotsSnapshot()
        if (shots.isEmpty()) return
        reviewIndex = (reviewIndex + direction).mod(shots.size)
        closeReviewOverlay(keepIndex = true)
        openReviewOverlay(reviewIndex)
    }

    private fun toggleCurrentReviewShot() {
        val ctrl = controller ?: return
        ctrl.toggleShotDeletion(reviewIndex)
    }

    private fun closeReviewOverlay(keepIndex: Boolean = false) {
        // Reset transform on the image view while it still exists.
        reviewImageView?.imageMatrix = android.graphics.Matrix()
        reviewScale = 1f
        reviewTranslateX = 0f
        reviewTranslateY = 0f
        reviewMatrix.reset()

        reviewOverlay?.let { rootLayout.removeView(it) }
        reviewOverlay = null
        reviewImageView = null
        reviewDeleteButton = null

        if (!keepIndex) reviewIndex = -1
    }


    // ---------------------------------------------------------------
    // CameraController.Listener
    // ---------------------------------------------------------------

    override fun onShotAdded(shot: SessionShot) {
        Log.i(TAG, "Shot added: ${shot.file.name}")
    }

    override fun onShotsChanged(shots: List<SessionShot>) {
        runOnUiThread {
            refreshThumbnails(shots)
            // If review overlay is open, rebuild it against the new shot list.
            if (reviewOverlay != null) {
                val idx = reviewIndex
                closeReviewOverlay(keepIndex = true)
                if (idx in shots.indices) {
                    openReviewOverlay(idx)
                } else if (shots.isNotEmpty()) {
                    // Current shot was removed — fall back to the last available
                    openReviewOverlay(shots.size - 1)
                } else {
                    // All shots removed — exit review
                    closeReviewOverlay()
                }
            }
        }
    }

    override fun onShotFailed(reason: String) {
        runOnUiThread {
            Toast.makeText(this, "Capture failed - retake", Toast.LENGTH_SHORT).show()
        }
    }

    override fun onShutterStateChanged(enabled: Boolean) {
        runOnUiThread {
            shutterButton.isEnabled = enabled
            shutterButton.alpha = if (enabled) 1.0f else 0.4f
        }
    }

    override fun onPreviewReady() {
        Log.i(TAG, "Preview ready session=$sessionId")
    }

    override fun onCameraStateChanged(lensFacing: Int, flashMode: Int) {
        runOnUiThread {
            val flashLabel = when (flashMode) {
                ImageCapture.FLASH_MODE_OFF -> "\u26A1"
                ImageCapture.FLASH_MODE_AUTO -> "\u26A1A"
                ImageCapture.FLASH_MODE_ON -> "\u26A1\u25CF"
                else -> "\u26A1"
            }
            flashButton.text = flashLabel
        }
    }

    override fun onCameraUnavailable(reason: String) {
        if (fallbackRequested) return
        fallbackRequested = true
        fallbackReason = reason
        Log.w(TAG, "Camera unavailable: $reason")

        val result = Intent().apply {
            putExtra("session_id", sessionId)
            putExtra("folder", folder)
            putExtra("reason", reason)
        }
        setResult(RESULT_CAMERA_UNAVAILABLE, result)
        finish()
    }

    // ---------------------------------------------------------------
    // Close, commit, discard
    // ---------------------------------------------------------------

    private fun attemptClose() {
        val ctrl = controller
        if (ctrl != null && ctrl.isBusy()) {
            Toast.makeText(this, "Please wait…", Toast.LENGTH_SHORT).show()
            return
        }

        val shotCount = ctrl?.shotsSnapshot()?.size ?: 0
        if (shotCount == 0) {
            // Nothing captured — plain cancel
            setResult(Activity.RESULT_CANCELED)
            finish()
            return
        }

        // Shots exist — confirm discard
        AlertDialog.Builder(this)
            .setTitle("Discard session?")
            .setMessage("You have $shotCount unsaved shot(s). Discard them?")
            .setPositiveButton("Discard") { _, _ -> discardSession() }
            .setNegativeButton("Keep editing") { _, _ -> }
            .setCancelable(false)
            .show()
    }

    private fun discardSession() {
        try {
            if (inboxDir.exists()) inboxDir.deleteRecursively()
        } catch (e: Exception) {
            Log.w(TAG, "discard cleanup failed: ${e.message}")
        }
        setResult(Activity.RESULT_CANCELED)
        finish()
    }

    private fun confirmCommit() {
        val ctrl = controller ?: return
        val shots = ctrl.shotsSnapshot()
        if (shots.isEmpty()) return

        val toSave = shots.count { !it.markedForDeletion }
        val toDelete = shots.size - toSave

        val msg = if (toDelete > 0) {
            "Save $toSave shot(s)? $toDelete will be discarded."
        } else {
            "Save $toSave shot(s) to this folder?"
        }

        AlertDialog.Builder(this)
            .setTitle("Save shots")
            .setMessage(msg)
            .setPositiveButton("Save") { _, _ -> commitSession() }
            .setNegativeButton("Cancel") { _, _ -> }
            .setCancelable(false)
            .show()
    }

    private fun commitSession() {
        val ctrl = controller ?: return
        val shots = ctrl.shotsSnapshot()
        val keptShots = shots.filter { !it.markedForDeletion }

        // Destination folder
        val baseDir = filesDir
        val destDir = if (folder.isEmpty()) {
            File(baseDir, "IndigiSnap")
        } else {
            File(File(baseDir, "IndigiSnap"), folder)
        }
        if (!destDir.exists()) destDir.mkdirs()

        // Leaf folder name for filename prefix
        val leafName = if (folder.isEmpty()) "" else folder.substringAfterLast("/")
        val prefix = if (leafName.isEmpty()) "IndigiSnap" else sanitizeLeaf(leafName)
        val dateStr = java.text.SimpleDateFormat("MM-dd-yy", java.util.Locale.US)
            .format(java.util.Date())

        var seq = 1
        var successCount = 0
        keptShots.forEach { shot ->
            try {
                var finalName = "${prefix}_${dateStr}"
                if (seq > 1) finalName += "_$seq"
                finalName += ".jpg"

                val destFile = File(destDir, finalName)
                // If collision from previous sessions, bump seq
                var trySeq = seq
                var actualDest = destFile
                while (actualDest.exists()) {
                    trySeq += 1
                    actualDest = File(destDir, "${prefix}_${dateStr}_$trySeq.jpg")
                }

                shot.file.renameTo(actualDest)
                successCount += 1
                seq += 1
            } catch (e: Exception) {
                Log.w(TAG, "commit move failed for ${shot.file.name}: ${e.message}")
            }
        }

        // Clean up inbox session folder
        try {
            if (inboxDir.exists()) inboxDir.deleteRecursively()
        } catch (e: Exception) {
            Log.w(TAG, "inbox cleanup failed: ${e.message}")
        }

        Log.i(TAG, "Committed $successCount of ${keptShots.size} shots to ${destDir.absolutePath}")

        val state = SessionState(
            sessionId = sessionId,
            folder = folder,
            shotCount = successCount,
            lastShotPath = keptShots.lastOrNull()?.file?.absolutePath,
            sessionFinished = successCount > 0,
            fallbackUsed = false,
            failureReason = null,
            cameraStartTimeMs = 0L,
        )
        val result = Intent().apply {
            putExtra("session_state", state)
        }
        setResult(Activity.RESULT_OK, result)
        finish()
    }

    private fun sanitizeLeaf(name: String): String {
        return name.replace(Regex("[^A-Za-z0-9_-]"), "_")
    }
}
