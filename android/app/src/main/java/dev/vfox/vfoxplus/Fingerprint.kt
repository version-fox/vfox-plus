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

package dev.vfox.vfoxplus

import android.net.http.SslCertificate
import java.security.MessageDigest

// Renders the SHA-256 fingerprint of a TLS certificate the way the
// settings of vfox+ on the PC show it, so the user can compare the two
// before trusting a self-signed certificate.
object Fingerprint {
    fun sha256HexColon(certificate: SslCertificate): String? {
        // SslCertificate.getX509Certificate needs API 29; below that there is
        // no fingerprint to compare, and the dialog says so.
        if (android.os.Build.VERSION.SDK_INT < android.os.Build.VERSION_CODES.Q) return null
        return try {
            val x509 = certificate.x509Certificate ?: return null
            sha256HexColon(x509.encoded)
        } catch (_: Exception) {
            null
        }
    }

    // The same fingerprint for a natively checked certificate: the SHA-256
    // of its DER encoding, as uppercase hex pairs joined by colons.
    fun sha256HexColon(encoded: ByteArray): String? {
        return try {
            val digest = MessageDigest.getInstance("SHA-256").digest(encoded)
            digest.joinToString(":") { "%02X".format(it) }
        } catch (_: Exception) {
            null
        }
    }

    fun sha256HexColon(certificate: java.security.cert.X509Certificate): String? {
        return try {
            sha256HexColon(certificate.encoded)
        } catch (_: Exception) {
            null
        }
    }
}
