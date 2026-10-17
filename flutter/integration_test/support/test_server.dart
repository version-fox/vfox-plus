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

import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';
import 'dart:typed_data';

/// A local HTTP server for the offline test plugin in shared/test-plugin.
///
/// It serves a plugin registry, the plugin and two SDK versions, following the
/// protocol in shared/test-plugin/README.md. SDK archives are sent slowly, in
/// chunks, so that vfox reports download progress, and can be held back after
/// their first chunk.
class TestServer {
  TestServer._(this._server, this._plugin, this._archives);

  final HttpServer _server;
  final String _plugin;
  final Map<String, Uint8List> _archives;
  Completer<void>? _held;

  /// Paths requested so far, in order.
  final List<String> requests = <String>[];

  /// Makes SDK downloads stop after their first chunk until
  /// [releaseDownloads], so that a test can act while vfox downloads.
  void holdDownloads() => _held = Completer<void>();

  /// Lets held downloads continue.
  void releaseDownloads() {
    _held?.complete();
    _held = null;
  }

  /// The base URL, to be passed to vfox in `VFOX_PLUS_TEST_SERVER`.
  String get url => 'http://${_server.address.address}:${_server.port}';

  /// The registry address for `vfox config registry.address`.
  String get registry => '$url/registry';

  /// Builds the SDK archives in [workDirectory] and starts serving.
  static Future<TestServer> start({required File plugin, required Directory workDirectory}) async {
    final archives = <String, Uint8List>{};
    final random = Random(42);
    for (final version in <String>['1.0.0', '2.0.0']) {
      final root = Directory('${workDirectory.path}/build/testsdk-$version/bin')..createSync(recursive: true);
      File('${root.path}/testsdk')
        ..writeAsStringSync('#!/bin/sh\necho "testsdk $version"\n')
        ..setLastModifiedSync(DateTime.utc(2026));
      // Incompressible padding makes the download long enough to report progress.
      File('${root.parent.path}/payload.bin')
          .writeAsBytesSync(Uint8List.fromList(List<int>.generate(512 * 1024, (_) => random.nextInt(256))));
      final archive = '${workDirectory.path}/testsdk-$version.tar.gz';
      final ProcessResult result = await Process.run('tar', <String>[
        '-czf',
        archive,
        '-C',
        '${workDirectory.path}/build',
        'testsdk-$version',
      ]);
      if (result.exitCode != 0) {
        throw StateError('tar failed: ${result.stderr}');
      }
      archives[version] = File(archive).readAsBytesSync();
    }
    final HttpServer server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final testServer = TestServer._(server, plugin.readAsStringSync(), archives);
    server.listen(testServer._handle);
    return testServer;
  }

  Future<void> _handle(HttpRequest request) async {
    final String path = request.uri.path;
    requests.add(path);
    final HttpResponse response = request.response;
    // A cancelled vfox closes the connection while its download is sent.
    response.done.ignore();
    final RegExpMatch? sdk = RegExp(r'^/sdk/testsdk-([0-9.]+)\.tar\.gz$').firstMatch(path);
    try {
      if (path == '/registry/index.json') {
        _json(response, <Object?>[
          <String, Object?>{
            'name': 'testsdk',
            'desc': 'An offline SDK for the vfox-plus end-to-end tests.',
            'homepage': 'https://github.com/version-fox/vfox-plus',
          },
        ]);
      } else if (path == '/registry/testsdk.json') {
        _json(response, <String, Object?>{
          'name': 'testsdk',
          'version': '0.1.0',
          'license': 'Apache-2.0',
          'author': 'vfox-plus',
          'downloadUrl': '$url/plugin/testsdk.lua',
          'minRuntimeVersion': '1.0.0',
        });
      } else if (path == '/plugin/testsdk.lua') {
        response.write(_plugin);
      } else if (sdk != null && _archives.containsKey(sdk.group(1))) {
        final Uint8List bytes = _archives[sdk.group(1)]!;
        response
          ..contentLength = bytes.length
          ..headers.contentType = ContentType('application', 'gzip');
        const chunk = 64 * 1024;
        for (var offset = 0; offset < bytes.length; offset += chunk) {
          response.add(bytes.sublist(offset, min(offset + chunk, bytes.length)));
          await response.flush();
          await (_held?.future ?? Future<void>.delayed(const Duration(milliseconds: 60)));
        }
      } else {
        response.statusCode = HttpStatus.notFound;
      }
      await response.close();
    } on IOException {
      // The client went away.
    }
  }

  static void _json(HttpResponse response, Object? body) {
    response
      ..headers.contentType = ContentType.json
      ..write(jsonEncode(body));
  }

  /// Stops serving.
  Future<void> close() => _server.close(force: true);
}
