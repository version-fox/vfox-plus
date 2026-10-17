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

import 'dart:io';

import 'package:material_ui/material_ui.dart';

import '../../l10n/app_localizations.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../theme/motion.dart';
import '../theme/tokens.dart';

/// Shows a dialog that scales in on the default spatial spring and fades in
/// on the default effects spring, as Material 3 Expressive dialogs do.
Future<T?> showExpressiveDialog<T>(BuildContext context, WidgetBuilder builder) {
  final MotionScheme motion = MotionScheme.of(context);
  final ColorScheme colors = Theme.of(context).colorScheme;
  final bool reduced = reduceMotion(context);
  return showGeneralDialog<T>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: colors.scrim.withValues(alpha: 0.32),
    transitionDuration: reduced ? Duration.zero : motion.defaultSpatial.settleDuration,
    pageBuilder: (BuildContext context, Animation<double> animation, Animation<double> secondaryAnimation) =>
        builder(context),
    transitionBuilder:
        (BuildContext context, Animation<double> animation, Animation<double> secondaryAnimation, Widget child) {
          final scale = CurvedAnimation(
            parent: animation,
            curve: motion.defaultSpatial.curve,
            reverseCurve: Curves.easeIn,
          );
          final opacity = CurvedAnimation(parent: animation, curve: motion.defaultEffects.curve);
          return FadeTransition(
            opacity: opacity,
            child: ScaleTransition(scale: Tween<double>(begin: 0.8, end: 1).animate(scale), child: child),
          );
        },
  );
}

/// Asks the user to confirm an action. Returns `true` if they did.
Future<bool> confirmAction(
  BuildContext context, {
  required String title,
  required String body,
  required String confirmLabel,
  IconData? icon,
  bool destructive = false,
}) async {
  final bool? confirmed = await showExpressiveDialog<bool>(context, (BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    return AlertDialog(
      icon: icon == null ? null : Icon(icon, color: destructive ? colors.error : colors.secondary),
      title: Text(title),
      content: Text(body),
      actions: <Widget>[
        TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(context.l10n.actionCancel)),
        FilledButton(
          style: destructive
              ? FilledButton.styleFrom(backgroundColor: colors.error, foregroundColor: colors.onError)
              : null,
          onPressed: () => Navigator.of(context).pop(true),
          child: Text(confirmLabel),
        ),
      ],
    );
  });
  return confirmed ?? false;
}

/// Asks for the project directory in which to pin [name] [version].
Future<String?> askProjectDirectory(BuildContext context, {required String name, required String version}) {
  return showExpressiveDialog<String>(
    context,
    (BuildContext context) => _ProjectDirectoryDialog(name: name, version: version),
  );
}

class _ProjectDirectoryDialog extends StatefulWidget {
  const _ProjectDirectoryDialog({required this.name, required this.version});

  final String name;
  final String version;

  @override
  State<_ProjectDirectoryDialog> createState() => _ProjectDirectoryDialogState();
}

class _ProjectDirectoryDialogState extends State<_ProjectDirectoryDialog> {
  final TextEditingController _path = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _path.dispose();
    super.dispose();
  }

  Future<void> _browse() async {
    final String? directory = await context.services.pickDirectory(
      initialDirectory: _path.text.isEmpty ? Platform.environment['HOME'] : _path.text,
    );
    if (directory != null) {
      setState(() {
        _path.text = directory;
        _error = null;
      });
    }
  }

  void _submit() {
    final String path = _path.text.trim();
    if (path.isEmpty || !Directory(path).existsSync()) {
      setState(() => _error = context.l10n.useProjectInvalid);
      return;
    }
    Navigator.of(context).pop(path);
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = context.l10n;
    final ThemeData theme = Theme.of(context);
    return AlertDialog(
      icon: Icon(AppIcons.projectScope, color: theme.colorScheme.secondary),
      title: Text(l10n.useProjectTitle(widget.name, widget.version)),
      content: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Text(l10n.useProjectBody),
            const SizedBox(height: Gaps.l),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Expanded(
                  child: TextField(
                    controller: _path,
                    autofocus: true,
                    decoration: InputDecoration(labelText: l10n.useProjectDirectory, errorText: _error),
                    onSubmitted: (_) => _submit(),
                  ),
                ),
                const SizedBox(width: Gaps.s),
                Padding(
                  padding: const EdgeInsets.only(top: Gaps.s),
                  child: IconButton.filledTonal(
                    tooltip: l10n.actionBrowse,
                    onPressed: _browse,
                    icon: const Icon(AppIcons.folder),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
      actions: <Widget>[
        TextButton(onPressed: () => Navigator.of(context).pop(), child: Text(l10n.actionCancel)),
        FilledButton(onPressed: _submit, child: Text(l10n.useProjectConfirm)),
      ],
    );
  }
}

/// The source and optional alias of a plugin added from a URL or a path.
typedef CustomPluginSource = ({String source, String? alias});

/// Asks for the source of a plugin that is not in the registry.
Future<CustomPluginSource?> askCustomPlugin(BuildContext context) {
  return showExpressiveDialog<CustomPluginSource>(context, (BuildContext context) => const _CustomPluginDialog());
}

class _CustomPluginDialog extends StatefulWidget {
  const _CustomPluginDialog();

  @override
  State<_CustomPluginDialog> createState() => _CustomPluginDialogState();
}

class _CustomPluginDialogState extends State<_CustomPluginDialog> {
  final TextEditingController _source = TextEditingController();
  final TextEditingController _alias = TextEditingController();
  String? _sourceError;
  String? _aliasError;

  static final RegExp _namePattern = RegExp(r'^[A-Za-z0-9][A-Za-z0-9._+-]*$');

  @override
  void dispose() {
    _source.dispose();
    _alias.dispose();
    super.dispose();
  }

  Future<void> _browse() async {
    final String? file = await context.services.pickFile(
      initialDirectory: Platform.environment['HOME'],
      extensions: const <String>['zip', 'lua'],
      typeLabel: context.l10n.pluginsCustomFileType,
    );
    if (file != null) {
      setState(() => _source.text = file);
    }
  }

  void _submit() {
    final AppLocalizations l10n = context.l10n;
    final String source = _source.text.trim();
    final String alias = _alias.text.trim();
    setState(() {
      _sourceError = source.isEmpty ? l10n.pluginsCustomRequired : null;
      _aliasError = alias.isNotEmpty && !_namePattern.hasMatch(alias) ? l10n.pluginsCustomInvalidName : null;
    });
    if (_sourceError == null && _aliasError == null) {
      Navigator.of(context).pop((source: source, alias: alias.isEmpty ? null : alias));
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = context.l10n;
    final ThemeData theme = Theme.of(context);
    return AlertDialog(
      icon: Icon(AppIcons.plugins, color: theme.colorScheme.secondary),
      title: Text(l10n.pluginsCustomTitle),
      content: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Expanded(
                  child: TextField(
                    controller: _source,
                    autofocus: true,
                    decoration: InputDecoration(
                      labelText: l10n.pluginsCustomSource,
                      helperText: l10n.pluginsCustomSourceHelp,
                      errorText: _sourceError,
                    ),
                    onSubmitted: (_) => _submit(),
                  ),
                ),
                const SizedBox(width: Gaps.s),
                Padding(
                  padding: const EdgeInsets.only(top: Gaps.s),
                  child: IconButton.filledTonal(
                    tooltip: l10n.actionBrowse,
                    onPressed: _browse,
                    icon: const Icon(AppIcons.folder),
                  ),
                ),
              ],
            ),
            const SizedBox(height: Gaps.l),
            TextField(
              controller: _alias,
              decoration: InputDecoration(
                labelText: l10n.pluginsCustomAlias,
                helperText: l10n.pluginsCustomAliasHelp,
                errorText: _aliasError,
              ),
              onSubmitted: (_) => _submit(),
            ),
          ],
        ),
      ),
      actions: <Widget>[
        TextButton(onPressed: () => Navigator.of(context).pop(), child: Text(l10n.actionCancel)),
        FilledButton(onPressed: _submit, child: Text(l10n.pluginsCustomConfirm)),
      ],
    );
  }
}
