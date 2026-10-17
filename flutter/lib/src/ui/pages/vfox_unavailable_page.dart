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

import 'package:material_new_shapes/material_new_shapes.dart';
import 'package:material_ui/material_ui.dart';

import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../../state/vfox_controller.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../platform_services.dart';
import '../widgets/page_parts.dart';

/// Shown instead of the pages while vfox is missing or broken.
class VfoxUnavailablePage extends StatelessWidget {
  /// Creates the page.
  const VfoxUnavailablePage({super.key});

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    return Pane(
      child: ListenableBuilder(
        listenable: model.vfox,
        builder: (BuildContext context, Widget? _) {
          final VfoxController vfox = model.vfox;
          switch (vfox.availability) {
            case VfoxAvailability.unknown || VfoxAvailability.checking:
              return LoadingMessage(label: l10n.vfoxCheckingTitle);
            case VfoxAvailability.ready:
              return const SizedBox.shrink();
            case VfoxAvailability.notFound || VfoxAvailability.broken:
              final bool missing = vfox.availability == VfoxAvailability.notFound;
              return StatusMessage(
                icon: missing ? AppIcons.notFound : AppIcons.warning,
                shape: missing ? MaterialShapes.clover4Leaf : MaterialShapes.softBurst,
                isError: !missing,
                title: missing ? l10n.vfoxMissingTitle : l10n.vfoxBrokenTitle,
                body: missing ? l10n.vfoxMissingBody : l10n.vfoxBrokenBody(vfox.executable ?? 'vfox'),
                actions: <Widget>[
                  FilledButton.icon(
                    onPressed: () => unawaited(context.services.openUrl(Links.vfoxInstall)),
                    icon: const Icon(AppIcons.openInNew),
                    label: Text(l10n.vfoxInstallGuide),
                  ),
                  FilledButton.tonalIcon(
                    onPressed: () => unawaited(_locate(context)),
                    icon: const Icon(AppIcons.folder),
                    label: Text(l10n.vfoxLocate),
                  ),
                  OutlinedButton.icon(
                    onPressed: () => unawaited(vfox.apply(model.settings.settings)),
                    icon: const Icon(AppIcons.retry),
                    label: Text(l10n.actionRetry),
                  ),
                ],
              );
          }
        },
      ),
    );
  }

  static Future<void> _locate(BuildContext context) async {
    final AppModel model = context.model;
    final String? path = await context.services.pickFile(initialDirectory: '/usr/local/bin');
    if (path != null) {
      await model.settings.update((settings) => settings.copyWith(vfoxExecutable: path));
    }
  }
}
