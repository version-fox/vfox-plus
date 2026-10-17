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

import 'package:dynamic_color/dynamic_color.dart';
import 'package:file_selector/file_selector.dart';
import 'package:flutter/services.dart';
import 'package:url_launcher/url_launcher.dart';

/// Desktop integration that the user interface needs: the browser, file
/// choosers, the clipboard and the accent colour.
///
/// Widgets reach these through [PlatformServices] instead of calling plugins
/// directly, so widget tests can replace them.
abstract interface class PlatformServices {
  /// The accent colour of the desktop, if it exposes one.
  Future<Color?> accentColor();

  /// Opens [url] in the default browser.
  Future<bool> openUrl(Uri url);

  /// Lets the user choose a directory.
  Future<String?> pickDirectory({String? initialDirectory});

  /// Lets the user choose a file, optionally one with an extension in
  /// [extensions] (without dots), described by [typeLabel].
  Future<String?> pickFile({String? initialDirectory, List<String>? extensions, String? typeLabel});

  /// Copies [text] to the clipboard.
  Future<void> copyText(String text);
}

/// The [PlatformServices] backed by Flutter plugins.
final class DesktopPlatformServices implements PlatformServices {
  /// Creates the desktop services.
  const DesktopPlatformServices();

  @override
  Future<Color?> accentColor() async {
    try {
      return await DynamicColorPlugin.getAccentColor();
    } on PlatformException {
      return null;
    } on MissingPluginException {
      return null;
    }
  }

  @override
  Future<bool> openUrl(Uri url) => launchUrl(url, mode: LaunchMode.externalApplication);

  @override
  Future<String?> pickDirectory({String? initialDirectory}) =>
      getDirectoryPath(initialDirectory: initialDirectory, canCreateDirectories: true);

  @override
  Future<String?> pickFile({String? initialDirectory, List<String>? extensions, String? typeLabel}) async {
    final List<XTypeGroup> groups = extensions == null
        ? const <XTypeGroup>[]
        : <XTypeGroup>[XTypeGroup(label: typeLabel, extensions: extensions)];
    return (await openFile(initialDirectory: initialDirectory, acceptedTypeGroups: groups))?.path;
  }

  @override
  Future<void> copyText(String text) => Clipboard.setData(ClipboardData(text: text));
}

/// Links that the user interface opens.
abstract final class Links {
  /// Installation guide of vfox.
  static final Uri vfoxInstall = Uri.parse('https://vfox.dev/guides/quick-start.html');

  /// Documentation of vfox.
  static final Uri vfoxDocs = Uri.parse('https://vfox.dev/');

  /// Source code of this application.
  static final Uri repository = Uri.parse('https://github.com/version-fox/vfox-plus');
}
