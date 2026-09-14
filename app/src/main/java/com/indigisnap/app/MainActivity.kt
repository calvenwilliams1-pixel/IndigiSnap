package com.indigisnap.app

import android.annotation.SuppressLint
import android.os.Bundle
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.appcompat.app.AppCompatActivity

class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        webView = WebView(this)
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        webView.webViewClient = WebViewClient()

        // For Week 1, we just show a static page.
        // Later, we'll load http://127.0.0.1:5000
        webView.loadDataWithBaseURL(
            "file:///android_asset/",
            HTML_PLACEHOLDER,
            "text/html",
            "UTF-8",
            null
        )

        setContentView(webView)
    }

    override fun onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack()
        } else {
            super.onBackPressed()
        }
    }

    companion object {
        private const val HTML_PLACEHOLDER = """
            <!DOCTYPE html>
            <html>
            <head>
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <style>
                    body {
                        background: #07040d;
                        color: #ffffff;
                        font-family: 'Courier New', monospace;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        height: 100vh;
                        margin: 0;
                        text-align: center;
                        flex-direction: column;
                    }
                    h1 {
                        color: #ff00ff;
                        text-shadow: 0 0 20px #ff00ff, 0 0 40px #ff00ff;
                        font-size: 2.5rem;
                        letter-spacing: 4px;
                    }
                    p {
                        color: #00ff99;
                        font-size: 1.2rem;
                    }
                    .icon {
                        font-size: 5rem;
                        margin-bottom: 2rem;
                    }
                </style>
            </head>
            <body>
                <div class="icon">👾</div>
                <h1>INDIGISNAP</h1>
                <p>Hello from IndigiSnap</p>
                <p style="font-size: 0.9rem; opacity: 0.6; margin-top: 2rem;">
                    Week 1: Pipeline proven ✅
                </p>
            </body>
            </html>
        """
    }
}
