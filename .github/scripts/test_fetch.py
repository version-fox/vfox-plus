# Copyright 2026 Han Li and contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Tests of fetch.sh of the action install-linux-packages: python3 -m unittest discover -s .github/scripts

Each mirror is an HTTP server of this process, which a test sets to be slow, to cut its
connections, to have nothing, or to send the wrong file.
"""

import hashlib
import os
import random
import re
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

FETCH = Path(__file__).resolve().parent.parent / "actions" / "install-linux-packages" / "fetch.sh"
NAME = "pkg_1.0_amd64.deb"
PATH = "/ubuntu/pool/main/p/pkg/" + NAME
CONTENT = random.Random(7).randbytes(300_000)
SIZE = len(CONTENT)
SHA256 = "SHA256:" + hashlib.sha256(CONTENT).hexdigest()
# The same size and another content, as a mirror that is wrong would send.
WRONG = CONTENT[:1000] + bytes([CONTENT[1000] ^ 0xFF]) + CONTENT[1001:]


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def do_GET(self):
        settings = self.server.settings
        self.server.requests.append(self.headers.get("Range"))
        if settings.get("status") or self.path != PATH:
            self.send_response(settings.get("status") or 404)
            self.send_header("Content-Length", "0")
            self.end_headers()
            return
        data = settings.get("content", CONTENT)
        start = 0
        requested = self.headers.get("Range")
        if requested and not settings.get("ignore_range"):
            start = int(re.match(r"bytes=(\d+)-", requested)[1])
            if start >= len(data):
                self.send_response(416)
                self.send_header("Content-Range", f"bytes */{len(data)}")
                self.send_header("Content-Length", "0")
                self.end_headers()
                return
            self.send_response(206)
            self.send_header("Content-Range", f"bytes {start}-{len(data) - 1}/{len(data)}")
        else:
            self.send_response(200)
        body = data[start:]
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        rate = settings.get("rate")
        cut = settings.get("cut_after")
        chunk = max(1, rate // 20) if rate else 65536
        sent = 0
        try:
            while sent < len(body):
                if cut is not None and sent >= cut:
                    self.close_connection = True
                    self.request.shutdown(socket.SHUT_RDWR)
                    return
                part = body[sent : sent + chunk]
                if cut is not None:
                    part = part[: cut - sent]
                self.wfile.write(part)
                self.wfile.flush()
                sent += len(part)
                if rate:
                    time.sleep(len(part) / rate)
        except (BrokenPipeError, ConnectionResetError):
            pass


class Server(ThreadingHTTPServer):
    daemon_threads = True

    def handle_error(self, request, client_address):
        # curl drops a connection that is too slow, and resets it.
        pass


class Mirror:
    """A mirror that has the file; the Range header of each request is in requests."""

    def __init__(self, **settings):
        self.server = Server(("127.0.0.1", 0), Handler)
        self.server.settings = settings
        self.server.requests = []
        self.requests = self.server.requests
        threading.Thread(target=self.server.serve_forever, kwargs={"poll_interval": 0.01}, daemon=True).start()
        self.url = f"http://127.0.0.1:{self.server.server_address[1]}{PATH}"

    def close(self):
        self.server.shutdown()
        self.server.server_close()


def refused_url() -> str:
    """The address of a port where nothing listens."""
    with socket.socket() as port:
        port.bind(("127.0.0.1", 0))
        return f"http://127.0.0.1:{port.getsockname()[1]}{PATH}"


@unittest.skipUnless(sys.platform == "linux", "fetch.sh uses the tools of GNU")
class FetchTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name) / "archives"
        (self.directory / "partial").mkdir(parents=True)
        self.home = Path(temporary.name) / "home"
        self.home.mkdir()

    def mirror(self, **settings) -> Mirror:
        mirror = Mirror(**settings)
        self.addCleanup(mirror.close)
        return mirror

    def fetch(self, addresses, seconds=60, hash_=SHA256, size=SIZE, **environment):
        """Runs fetch.sh, which drops a connection under 150000 bytes a second for a second."""
        variables = {
            key: value
            for key, value in os.environ.items()
            if not key.lower().endswith("_proxy") and key not in ("CURL_HOME", "XDG_CONFIG_HOME")
        }
        # Neither a proxy nor a .curlrc of the user changes where the requests go.
        variables.update(HOME=str(self.home), NO_PROXY="127.0.0.1", APT_MIN_SPEED="150000", APT_SPEED_TIME="1")
        variables.update(APT_DEADLINE=str(int(time.time()) + seconds))
        variables.update(environment)
        return subprocess.run(
            ["bash", str(FETCH), str(self.directory), str(size), NAME, hash_, *addresses],
            env=variables,
            check=False,
            capture_output=True,
            text=True,
            timeout=seconds + 30,
        )

    @property
    def archive(self) -> Path:
        return self.directory / NAME

    @property
    def partial(self) -> Path:
        return self.directory / "partial" / NAME

    def assertFetched(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(self.archive.read_bytes(), CONTENT)
        self.assertFalse(self.partial.exists(), "nothing is left in partial/")

    def test_downloads_the_archive_and_moves_it_out_of_partial(self):
        mirror = self.mirror()
        result = self.fetch([mirror.url])
        self.assertFetched(result)
        self.assertEqual(mirror.requests, [None])
        self.assertEqual(result.stdout + result.stderr, "")

    def test_continues_what_is_in_partial(self):
        # The first version of the action wrote the file again from its start, and a file that
        # takes longer than the time of one attempt never ended.
        mirror = self.mirror()
        self.partial.write_bytes(CONTENT[:100_000])
        self.assertFetched(self.fetch([mirror.url]))
        self.assertEqual(mirror.requests, ["bytes=100000-"])

    def test_continues_after_a_connection_that_was_cut(self):
        mirror = self.mirror(cut_after=100_000)
        self.assertFetched(self.fetch([mirror.url]))
        self.assertEqual(mirror.requests, [None, "bytes=100000-", "bytes=200000-"])

    def test_a_partial_that_is_complete_needs_no_request(self):
        mirror = self.mirror()
        self.partial.write_bytes(CONTENT)
        self.assertFetched(self.fetch([mirror.url]))
        self.assertEqual(mirror.requests, [])

    def test_a_partial_that_is_larger_than_the_file_starts_again(self):
        mirror = self.mirror()
        self.partial.write_bytes(CONTENT + b"more")
        self.assertFetched(self.fetch([mirror.url]))
        self.assertEqual(mirror.requests, [None])

    def test_a_server_that_does_not_continue_sends_all_of_the_file(self):
        mirror = self.mirror(ignore_range=True)
        self.partial.write_bytes(CONTENT[:100_000])
        self.assertFetched(self.fetch([mirror.url]))
        self.assertEqual(mirror.requests, ["bytes=100000-", None])

    def test_goes_to_the_next_mirror_when_one_has_not_the_file(self):
        first, second = self.mirror(status=404), self.mirror()
        self.assertFetched(self.fetch([first.url, second.url]))
        self.assertEqual((len(first.requests), len(second.requests)), (1, 1))

    def test_goes_to_the_next_mirror_when_one_does_not_answer(self):
        second = self.mirror()
        self.assertFetched(self.fetch([refused_url(), second.url]))

    def test_goes_to_the_next_mirror_when_one_is_slow_and_continues_there(self):
        first, second = self.mirror(rate=30_000), self.mirror()
        result = self.fetch([first.url, second.url])
        self.assertFetched(result)
        self.assertEqual(len(first.requests), 1)
        self.assertEqual(len(second.requests), 1)
        self.assertRegex(second.requests[0], r"^bytes=[1-9]\d*-$", "the second mirror continued the file")
        self.assertRegex(
            result.stdout, rf"{NAME}: \d+ of {SIZE} bytes, \d+ bytes a second from 127\.0\.0\.1:\d+: curl: \(28\)"
        )

    def test_goes_on_with_the_best_mirror_when_every_one_is_slow(self):
        # All below the minimum speed of 150000: a drops after a second at 30 KB/s, b after a second at
        # 100 KB/s, and then b, the better, continues without being dropped.
        a, b = self.mirror(rate=30_000), self.mirror(rate=100_000)
        self.assertFetched(self.fetch([a.url, b.url]))
        self.assertEqual(len(a.requests), 1, "the worse one is not asked again")
        self.assertEqual(len(b.requests), 2)

    def test_gives_up_when_no_mirror_has_the_file(self):
        first, second = self.mirror(status=404), self.mirror(status=404)
        result = self.fetch([first.url, second.url], seconds=120)
        self.assertEqual(result.returncode, 1)
        self.assertIn("no mirror sends it", result.stderr)
        self.assertFalse(self.archive.exists())
        self.assertEqual((len(first.requests), len(second.requests)), (3, 3), "three rounds")

    def test_ends_at_the_deadline_and_keeps_what_came(self):
        mirror = self.mirror(rate=50_000)
        result = self.fetch([mirror.url], seconds=2, APT_MIN_SPEED="1000", APT_SPEED_TIME="10")
        self.assertEqual(result.returncode, 1)
        self.assertFalse(self.archive.exists())
        self.assertTrue(0 < self.partial.stat().st_size < SIZE)

    def test_checks_the_hash_before_the_archive_is_moved(self):
        mirror = self.mirror(content=WRONG)
        result = self.fetch([mirror.url])
        self.assertEqual(result.returncode, 1)
        self.assertIn("no mirror sends the right file", result.stderr)
        self.assertFalse(self.archive.exists())
        self.assertFalse(self.partial.exists())
        self.assertEqual(len(mirror.requests), 3, "three downloads")

    def test_tries_another_mirror_after_a_wrong_file(self):
        wrong, right = self.mirror(content=WRONG), self.mirror()
        result = self.fetch([wrong.url, right.url])
        self.assertFetched(result)
        self.assertEqual(right.requests, [None], "all of the file again, not what the wrong one sent")
        self.assertIn(f"the hash is not {SHA256}", result.stdout)

    def test_an_archive_in_the_directory_with_the_right_hash_is_kept(self):
        mirror = self.mirror()
        self.archive.write_bytes(CONTENT)
        result = self.fetch([mirror.url])
        self.assertFetched(result)
        self.assertEqual(mirror.requests, [])
        self.assertEqual(result.stdout, "")

    def test_an_archive_in_the_directory_with_another_hash_is_replaced(self):
        # apt takes it for a good one because of its size, and dpkg fails on it.
        mirror = self.mirror()
        self.archive.write_bytes(WRONG)
        result = self.fetch([mirror.url])
        self.assertFetched(result)
        self.assertEqual(mirror.requests, [None])
        self.assertIn("is not the one that apt expects; removed", result.stdout)

    def test_an_archive_in_the_directory_with_another_size_is_replaced(self):
        mirror = self.mirror()
        self.archive.write_bytes(CONTENT[:1000])
        self.assertFetched(self.fetch([mirror.url]))
        self.assertEqual(mirror.requests, [None])

    def test_checks_with_the_hash_type_that_apt_printed(self):
        mirror = self.mirror()
        self.assertFetched(self.fetch([mirror.url], hash_="MD5Sum:" + hashlib.md5(CONTENT).hexdigest()))

    def test_a_hash_type_that_is_not_known_checks_nothing_and_fails(self):
        mirror = self.mirror()
        result = self.fetch([mirror.url], hash_="CRC32:abcdef")
        self.assertEqual(result.returncode, 1)
        self.assertFalse(self.archive.exists())

    def test_needs_its_arguments(self):
        result = subprocess.run(["bash", str(FETCH), str(self.directory)], check=False, capture_output=True, text=True)
        self.assertEqual(result.returncode, 2)
        self.assertIn("Usage: fetch.sh", result.stderr)


if __name__ == "__main__":
    unittest.main()
