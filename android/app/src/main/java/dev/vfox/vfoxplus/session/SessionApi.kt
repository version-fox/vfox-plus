// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dev.vfox.vfoxplus.session

import android.content.Context
import dev.vfox.vfoxplus.Fingerprint
import java.io.ByteArrayOutputStream
import java.io.Closeable
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import java.security.SecureRandom
import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import javax.net.ssl.HttpsURLConnection
import javax.net.ssl.SSLContext
import javax.net.ssl.TrustManagerFactory
import javax.net.ssl.X509TrustManager

// The answer of the server: its status, its body, and where it was read.
data class ApiResponse(val code: Int, val body: String, val finalUrl: String)

// The native transport of the paired session: plain HttpURLConnection and
// org.json, no networking dependency, so that the measured APK carries only
// what the native UI needs. Cookies persist in SharedPreferences, like the
// cookies of the WebView, so a restart reconnects without pairing again.
class SessionApi(context: Context) {
    private val appContext = context.applicationContext

    // The fingerprint the user trusted, for this process run: trusting once
    // is what the WebView dialog does. A restart asks again.
    @Volatile
    var trustedFingerprint: String? = null

    private val cookieJar = PersistentCookies(appContext)
    private val ssl = PinningSsl()

    // GETs the URL, following redirects by hand so the session cookie is
    // collected on the way. Returns where it landed.
    @Throws(IOException::class, UntrustedChain::class)
    fun get(url: String, readTimeoutMs: Int = 30_000): ApiResponse =
        exchange("GET", url, null, null, readTimeoutMs)

    @Throws(IOException::class, UntrustedChain::class)
    fun postJson(url: String, json: String): ApiResponse =
        exchange("POST", url, json.toByteArray(Charsets.UTF_8), "application/json", 30_000)

    @Throws(IOException::class, UntrustedChain::class)
    fun putJson(url: String, json: String): ApiResponse =
        exchange("PUT", url, json.toByteArray(Charsets.UTF_8), "application/json", 30_000)

    @Throws(IOException::class, UntrustedChain::class)
    fun delete(url: String): ApiResponse =
        exchange("DELETE", url, null, null, 30_000)

    // The plugin file upload: multipart/form-data with the file and the alias.
    @Throws(IOException::class, UntrustedChain::class)
    fun postMultipart(url: String, fileName: String, fileBytes: ByteArray, alias: String): ApiResponse {
        val boundary = "vfox${System.currentTimeMillis().toString(36)}"
        val out = ByteArrayOutputStream()
        fun part(headers: String, bytes: ByteArray) {
            out.write("--$boundary\r\n$headers\r\n\r\n".toByteArray(Charsets.UTF_8))
            out.write(bytes)
            out.write("\r\n".toByteArray(Charsets.UTF_8))
        }
        part(
            "Content-Disposition: form-data; name=\"file\"; filename=\"$fileName\"\r\nContent-Type: application/octet-stream",
            fileBytes,
        )
        part("Content-Disposition: form-data; name=\"alias\"", alias.toByteArray(Charsets.UTF_8))
        out.write("--$boundary--\r\n".toByteArray(Charsets.UTF_8))
        return exchange("POST", url, out.toByteArray(), "multipart/form-data; boundary=$boundary", 120_000)
    }

    // The session cookie is the credential: forgetting it disconnects the
    // phone, like clearing the cookies of the WebView does.
    fun forgetSession() {
        cookieJar.clear()
        trustedFingerprint = null
    }

    // Opens the server-sent events of the session on the calling thread until
    // closed: `event: <kind>` lines come back as kinds with an empty payload,
    // which means refetching, like the web interface does.
    fun openEvents(url: String, onEvent: (String) -> Unit): Closeable {
        val task = EventTask(url, onEvent)
        Thread(task, "vfox-events").apply { isDaemon = true; start() }
        return task
    }

    @Throws(IOException::class, UntrustedChain::class)
    private fun exchange(
        method: String,
        url: String,
        body: ByteArray?,
        contentType: String?,
        readTimeoutMs: Int,
    ): ApiResponse {
        var current = url
        var redirects = 0
        while (true) {
            val conn = open(current)
            try {
                conn.requestMethod = method
                conn.connectTimeout = 10_000
                conn.readTimeout = readTimeoutMs
                conn.instanceFollowRedirects = false
                conn.setRequestProperty("Accept", "application/json, */*")
                val cookies = cookieJar.header(URL(current).host)
                if (cookies != null) conn.setRequestProperty("Cookie", cookies)
                if (body != null) {
                    conn.doOutput = true
                    if (contentType != null) conn.setRequestProperty("Content-Type", contentType)
                    conn.outputStream.use { it.write(body) }
                }
                val code = try {
                    conn.responseCode
                } catch (e: javax.net.ssl.SSLHandshakeException) {
                    throw findUntrusted(e) ?: e
                }
                cookieJar.store(URL(current).host, conn.headerFields["Set-Cookie"].orEmpty())
                if (code in 301..308) {
                    if (redirects++ >= 8) throw IOException("too many redirects")
                    val location = conn.getHeaderField("Location")
                        ?: throw IOException("redirect without a location")
                    current = URL(URL(current), location).toString()
                    // A 303 turns the follow-up into a GET without a body.
                    if (code == 303) {
                        return exchangeGet(current, readTimeoutMs)
                    }
                    continue
                }
                val stream = if (code in 200..299) conn.inputStream else conn.errorStream
                val text = stream?.bufferedReader(Charsets.UTF_8)?.use { it.readText().take(256 * 1024) }.orEmpty()
                return ApiResponse(code, text, current)
            } finally {
                conn.disconnect()
            }
        }
    }

    @Throws(IOException::class, UntrustedChain::class)
    private fun exchangeGet(url: String, readTimeoutMs: Int): ApiResponse =
        exchange("GET", url, null, null, readTimeoutMs)

    @Throws(IOException::class, UntrustedChain::class)
    private fun open(url: String): HttpURLConnection {
        val conn = URL(url).openConnection() as HttpURLConnection
        if (conn is HttpsURLConnection) {
            // Only the trust changes (the fingerprint the user compares);
            // the default hostname check stays, like in the WebView.
            conn.sslSocketFactory = ssl.factory()
        }
        return conn
    }

    // The handshake wraps what the trust manager throws: dig the recorded
    // chain back out so the UI can show its fingerprint.
    private fun findUntrusted(e: Throwable): UntrustedChain? {
        var cause: Throwable? = e
        while (cause != null) {
            if (cause is UntrustedChain) return cause
            cause = cause.cause
        }
        return null
    }

    // Carries the presented chain out of the TLS handshake: the UI shows its
    // fingerprint, and trusting it retries with the pin below.
    class UntrustedChain(val chain: Array<X509Certificate>) : CertificateException()

    private inner class PinningSsl {
        private val default: X509TrustManager = run {
            val factory = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm())
            factory.init(null as java.security.KeyStore?)
            factory.trustManagers.filterIsInstance<X509TrustManager>().first()
        }

        private val pinning = object : X509TrustManager {
            override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String) =
                default.checkClientTrusted(chain, authType)

            override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String) {
                try {
                    default.checkServerTrusted(chain, authType)
                    return
                } catch (_: CertificateException) {
                    // Not trusted by the system: fall through to the pin.
                }
                val trusted = trustedFingerprint
                val presented = chain.firstOrNull()?.let { Fingerprint.sha256HexColon(it) }
                if (trusted != null && presented != null && presented.equals(trusted, ignoreCase = true)) return
                throw UntrustedChain(chain)
            }

            override fun getAcceptedIssuers(): Array<X509Certificate> = default.acceptedIssuers
        }

        private val context: SSLContext = SSLContext.getInstance("TLS").apply {
            init(null, arrayOf(pinning), SecureRandom())
        }

        fun factory() = context.socketFactory
    }

    private inner class EventTask(val url: String, val onEvent: (String) -> Unit) : Runnable, Closeable {
        @Volatile
        private var closed = false

        override fun run() {
            while (!closed) {
                try {
                    readOnce()
                } catch (_: UntrustedChain) {
                    // The trust dialog owns the retry: stop here.
                    return
                } catch (_: IOException) {
                    if (!closed) {
                        try {
                            Thread.sleep(3_000)
                        } catch (_: InterruptedException) {
                            return
                        }
                    }
                }
            }
        }

        @Throws(IOException::class, UntrustedChain::class)
        private fun readOnce() {
            val conn = open(url)
            conn.requestMethod = "GET"
            conn.connectTimeout = 10_000
            conn.readTimeout = 90_000
            conn.instanceFollowRedirects = false
            conn.setRequestProperty("Accept", "text/event-stream")
            val cookies = cookieJar.header(URL(url).host)
            if (cookies != null) conn.setRequestProperty("Cookie", cookies)
            try {
                val code = try {
                    conn.responseCode
                } catch (e: javax.net.ssl.SSLHandshakeException) {
                    throw findUntrusted(e) ?: e
                }
                if (code != 200) {
                    conn.disconnect()
                    throw IOException("events: HTTP ${conn.responseCode}")
                }
                var kind = ""
                conn.inputStream.bufferedReader(Charsets.UTF_8).useLines { lines ->
                    for (line in lines) {
                        if (closed) break
                        when {
                            line.startsWith("event:") -> kind = line.substringAfter("event:").trim()
                            line.isBlank() && kind.isNotEmpty() -> {
                                val dispatch = kind
                                kind = ""
                                handler.post { onEvent(dispatch) }
                            }
                        }
                    }
                }
            } finally {
                conn.disconnect()
            }
        }

        override fun close() {
            closed = true
        }

        private val handler = android.os.Handler(android.os.Looper.getMainLooper())
    }
}
