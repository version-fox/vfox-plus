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
// `vfox-plus serve` terminal prints it, so the user can compare the two
// before trusting a self-signed certificate.
object Fingerprint {
    fun sha256HexColon(certificate: SslCertificate): String? {
        return try {
            val x509 = certificate.x509Certificate ?: return null
            val digest = MessageDigest.getInstance("SHA-256").digest(x509.encoded)
            digest.joinToString(":") { "%02X".format(it) }
        } catch (_: Exception) {
            null
        }
    }
}
