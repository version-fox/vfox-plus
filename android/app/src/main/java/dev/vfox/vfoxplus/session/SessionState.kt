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

import dev.vfox.vfoxplus.Fingerprint
import java.io.IOException
import java.security.cert.X509Certificate

// What a natively paired session is: the server it talks to, the session
// cookie it sends, and the certificate fingerprint it trusted, if any.
// The cookies persist across restarts:
// reconnecting after the application was killed does not pair again.
data class SessionState(
    val base: String,
    val cookies: Map<String, String> = emptyMap(),
    val trustedFingerprint: String? = null,
)

// The errors of the native session, in the shape the screens show: the
// certificate to compare, or the address that cannot be reached. They extend
// Exception so background work can return them through Result.
sealed interface SessionError {
    // The server presents a certificate the system does not trust.
    class Untrusted(val chain: Array<X509Certificate>) : Exception(), SessionError {
        val fingerprint: String? get() = chain.firstOrNull()?.let { Fingerprint.sha256HexColon(it) }
    }

    // The server cannot be reached at all, or answers without a session.
    class Unreachable(val base: String, cause: IOException?) : Exception(cause), SessionError

    // The server refused the request: the pairing link was used, or the
    // session is gone.
    class Refused(val code: Int, val body: String) : Exception("HTTP $code"), SessionError
}

// Thrown inside runApi tasks for HTTP refusals; runApi maps it to Refused.
class RefusedException(val code: Int, val body: String) : java.io.IOException("HTTP $code")
