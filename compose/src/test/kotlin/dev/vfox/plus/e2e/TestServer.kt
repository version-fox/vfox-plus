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

package dev.vfox.plus.e2e

import com.sun.net.httpserver.HttpExchange
import com.sun.net.httpserver.HttpServer
import dev.vfox.plus.support.SHARED
import java.net.InetAddress
import java.net.InetSocketAddress
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import kotlin.io.path.readBytes
import kotlin.io.path.writeBytes

/**
 * A local HTTP server for the offline test plugin of shared/test-plugin,
 * following the protocol of shared/test-plugin/README.md. SDK archives are
 * sent in chunks, slowly, so that vfox reports download progress; [hold]
 * stops downloads after their first chunk until [release].
 */
class TestServer(
    work: Path,
) : AutoCloseable {
    private val plugin = Files.readString(SHARED.resolve("test-plugin/testsdk.lua"))
    private val archives = mutableMapOf<String, ByteArray>()

    @Volatile private var held: CountDownLatch? = null
    private val server = HttpServer.create(InetSocketAddress(InetAddress.getLoopbackAddress(), 0), 0)

    /** The base URL, for example `http://127.0.0.1:43210`. */
    val url: String

    init {
        for ((index, version) in listOf("1.0.0", "2.0.0").withIndex()) {
            val root = work.resolve("build/testsdk-$version")
            Files.createDirectories(root.resolve("bin"))
            root.resolve("bin/testsdk").writeBytes("#!/bin/sh\necho \"testsdk $version\"\n".encodeToByteArray())
            root.resolve("bin/testsdk").toFile().setExecutable(true)
            root.resolve("payload.bin").writeBytes(padding(42 + index))
            val archive = work.resolve("testsdk-$version.tar.gz")
            val tar = ProcessBuilder("tar", "-czf", archive.toString(), "-C", work.resolve("build").toString(), "testsdk-$version").start()
            check(tar.waitFor() == 0) { "tar failed" }
            archives[version] = archive.readBytes()
        }
        server.executor = Executors.newCachedThreadPool()
        server.createContext("/") { exchange -> exchange.use { handle(it) } }
        server.start()
        url = "http://127.0.0.1:${server.address.port}"
    }

    /** Deterministic, incompressible bytes, so that downloads take long enough to report progress. */
    private fun padding(seed: Int): ByteArray {
        val bytes = ByteArray(512 * 1024)
        var state = seed
        for (i in bytes.indices) {
            state = state * 1103515245 + 12345
            bytes[i] = (state ushr 24).toByte()
        }
        return bytes
    }

    /** Makes downloads stop after their first chunk. */
    fun hold() {
        held = CountDownLatch(1)
    }

    /** Lets held downloads continue. */
    fun release() {
        held?.countDown()
        held = null
    }

    private fun handle(exchange: HttpExchange) {
        val path = exchange.requestURI.path
        val sdk = Regex("""^/sdk/testsdk-([0-9.]+)\.tar\.gz$""").find(path)?.groupValues?.get(1)
        when {
            path == "/registry/index.json" -> {
                send(
                    exchange,
                    """[{"name": "testsdk", "desc": "An offline SDK for the vfox-plus end-to-end tests.", """ +
                        """"homepage": "https://github.com/version-fox/vfox-plus"}]""",
                )
            }

            path == "/registry/testsdk.json" -> {
                send(
                    exchange,
                    """{"name": "testsdk", "version": "0.1.0", "license": "Apache-2.0", "author": "vfox-plus", """ +
                        """"downloadUrl": "$url/plugin/testsdk.lua", "minRuntimeVersion": "1.0.0"}""",
                )
            }

            path == "/plugin/testsdk.lua" -> {
                send(exchange, plugin)
            }

            sdk != null && sdk in archives -> {
                val bytes = archives.getValue(sdk)
                exchange.responseHeaders.add("Content-Type", "application/gzip")
                exchange.sendResponseHeaders(200, bytes.size.toLong())
                val chunk = 64 * 1024
                var offset = 0
                runCatching {
                    while (offset < bytes.size) {
                        exchange.responseBody.write(bytes, offset, minOf(chunk, bytes.size - offset))
                        exchange.responseBody.flush()
                        offset += chunk
                        held?.await(60, TimeUnit.SECONDS) ?: Thread.sleep(60)
                    }
                }
            }

            else -> {
                exchange.sendResponseHeaders(404, -1)
            }
        }
    }

    private fun send(
        exchange: HttpExchange,
        text: String,
    ) {
        val bytes = text.encodeToByteArray()
        exchange.sendResponseHeaders(200, bytes.size.toLong())
        exchange.responseBody.write(bytes)
    }

    override fun close() {
        release()
        server.stop(0)
        (server.executor as? java.util.concurrent.ExecutorService)?.shutdownNow()
    }
}
