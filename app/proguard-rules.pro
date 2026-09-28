

# ---- IndigiSnap specific keeps ----

# Go JNI bridge (called from native code via JNI)
-keep class com.indigisnap.app.ServerBridge { *; }

# JS bridge (called from WebView via @JavascriptInterface)
-keepclassmembers class com.indigisnap.app.MainActivity$WebAppBridge {
    @android.webkit.JavascriptInterface <methods>;
}

# Enum values accessed by name (CapturePreset.fromKey, ShutterState)
-keepclassmembers enum com.indigisnap.app.* {
    public static **[] values();
    public static ** valueOf(java.lang.String);
}

# Keep all Parcelable CREATOR fields (SessionState)
-keepclassmembers class * implements android.os.Parcelable {
    public static final ** CREATOR;
}
