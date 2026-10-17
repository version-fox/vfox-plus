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

import '../../core/models.dart';
import '../../core/settings/app_settings.dart';
import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../../state/vfox_config_controller.dart';
import '../../state/vfox_controller.dart';
import '../../version.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../platform_services.dart';
import '../theme/motion.dart';
import '../theme/tokens.dart';
import '../widgets/button_group.dart';
import '../widgets/list_group.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/page_parts.dart';
import '../widgets/polygon_border.dart';

/// Seed colours offered when the desktop accent colour is not used.
const List<int> seedColors = <int>[
  AppSettings.defaultSeedColor,
  0xFFD9473B,
  0xFFC2407A,
  0xFF7E57C2,
  0xFF3F6FD8,
  0xFF0B8A9A,
  0xFF3E8E41,
  0xFF8D7B1E,
];

/// The configuration keys of vfox that get a dedicated editor.
abstract final class VfoxConfigKeys {
  /// `proxy.enable`
  static const String proxyEnable = 'proxy.enable';

  /// `proxy.url`
  static const String proxyUrl = 'proxy.url';

  /// `registry.address`
  static const String registry = 'registry.address';

  /// `storage.sdkPath`
  static const String storage = 'storage.sdkPath';

  /// `legacyVersionFile.enable`
  static const String legacyEnable = 'legacyVersionFile.enable';

  /// `legacyVersionFile.strategy`
  static const String legacyStrategy = 'legacyVersionFile.strategy';

  /// `cache.availableHookDuration`
  static const String cache = 'cache.availableHookDuration';

  /// `plugin.http.timeout`
  static const String httpTimeout = 'plugin.http.timeout';

  /// `plugin.http.downloadTimeout`
  static const String downloadTimeout = 'plugin.http.downloadTimeout';

  /// `gitignore.enable`
  static const String gitignore = 'gitignore.enable';

  /// All keys above.
  static const Set<String> known = <String>{
    proxyEnable,
    proxyUrl,
    registry,
    storage,
    legacyEnable,
    legacyStrategy,
    cache,
    httpTimeout,
    downloadTimeout,
    gitignore,
  };
}

/// Settings of the application and of vfox.
class SettingsPage extends StatefulWidget {
  /// Creates the page.
  const SettingsPage({super.key});

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  bool _requested = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final AppModel model = context.model;
    if (!_requested && model.vfox.availability == VfoxAvailability.ready) {
      _requested = true;
      unawaited(model.vfoxConfig.load());
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = context.l10n;
    return Pane(
      child: CustomScrollView(
        slivers: <Widget>[
          SliverToBoxAdapter(child: PageHeader(title: l10n.navSettings)),
          const SliverPadding(
            padding: EdgeInsets.fromLTRB(Gaps.xl, 0, Gaps.xl, Gaps.xxl),
            sliver: SliverToBoxAdapter(
              child: ContentWidth(
                maxWidth: 840,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[_AppearanceSection(), _VfoxSection(), _VfoxConfigSection(), _AboutSection()],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _AppearanceSection extends StatelessWidget {
  const _AppearanceSection();

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    return ListenableBuilder(
      listenable: model.settings,
      builder: (BuildContext context, Widget? _) {
        final AppSettings settings = model.settings.settings;
        Future<void> update(AppSettings Function(AppSettings) change) => model.settings.update(change);
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            GroupHeader(title: l10n.settingsAppearance),
            ListGroup(
              children: <Widget>[
                ListGroupItem(
                  wrapTrailing: true,
                  leading: const Icon(AppIcons.darkMode),
                  title: Text(l10n.settingsTheme),
                  trailing: ConnectedButtonGroup<ThemePreference>(
                    segments: <ButtonGroupSegment<ThemePreference>>[
                      ButtonGroupSegment<ThemePreference>(
                        value: ThemePreference.system,
                        label: l10n.themeSystem,
                        icon: AppIcons.systemMode,
                      ),
                      ButtonGroupSegment<ThemePreference>(
                        value: ThemePreference.light,
                        label: l10n.themeLight,
                        icon: AppIcons.lightMode,
                      ),
                      ButtonGroupSegment<ThemePreference>(
                        value: ThemePreference.dark,
                        label: l10n.themeDark,
                        icon: AppIcons.darkMode,
                      ),
                    ],
                    selected: settings.theme,
                    onSelected: (ThemePreference theme) => update((s) => s.copyWith(theme: theme)),
                  ),
                ),
                ListGroupItem(
                  leading: const Icon(AppIcons.palette),
                  title: Text(l10n.settingsSystemColor),
                  trailing: Switch(
                    value: settings.useSystemColor,
                    onChanged: (bool value) => update((s) => s.copyWith(useSystemColor: value)),
                  ),
                  onTap: () => update((s) => s.copyWith(useSystemColor: !s.useSystemColor)),
                ),
                ListGroupItem(
                  leading: const SizedBox(width: 24),
                  title: Text(l10n.settingsColor),
                  subtitle: Padding(
                    padding: const EdgeInsets.only(top: Gaps.s),
                    child: Wrap(
                      spacing: Gaps.s,
                      runSpacing: Gaps.s,
                      children: <Widget>[
                        for (var i = 0; i < seedColors.length; i++)
                          _ColorSwatch(
                            color: Color(seedColors[i]),
                            label: l10n.settingsColorOption(i + 1),
                            selected: !settings.useSystemColor && settings.seedColor == seedColors[i],
                            onTap: () => update((s) => s.copyWith(seedColor: seedColors[i], useSystemColor: false)),
                          ),
                      ],
                    ),
                  ),
                ),
                ListGroupItem(
                  wrapTrailing: true,
                  leading: const Icon(AppIcons.language),
                  title: Text(l10n.settingsLanguage),
                  trailing: ConnectedButtonGroup<LanguagePreference>(
                    segments: <ButtonGroupSegment<LanguagePreference>>[
                      ButtonGroupSegment<LanguagePreference>(
                        value: LanguagePreference.system,
                        label: l10n.languageSystem,
                      ),
                      // Language names are always shown in their own language.
                      const ButtonGroupSegment<LanguagePreference>(value: LanguagePreference.english, label: 'English'),
                      const ButtonGroupSegment<LanguagePreference>(
                        value: LanguagePreference.simplifiedChinese,
                        label: '简体中文',
                      ),
                    ],
                    selected: settings.language,
                    onSelected: (LanguagePreference language) => update((s) => s.copyWith(language: language)),
                  ),
                ),
              ],
            ),
          ],
        );
      },
    );
  }
}

class _ColorSwatch extends StatelessWidget {
  const _ColorSwatch({required this.color, required this.label, required this.selected, required this.onTap});

  /// The selected swatch morphs from a circle into a flower.
  static final Morph _circleToFlower = Morph(MaterialShapes.circle, MaterialShapes.flower);

  final Color color;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final ColorScheme swatch = ColorScheme.fromSeed(seedColor: color, brightness: Theme.of(context).brightness);
    return Semantics(
      button: true,
      selected: selected,
      label: label,
      child: Tooltip(
        message: label,
        child: InkResponse(
          onTap: onTap,
          radius: 24,
          child: SpringBuilder(
            value: selected ? 1 : 0,
            spring: MotionScheme.of(context).fastSpatial,
            builder: (BuildContext context, double t, Widget? _) {
              return DecoratedBox(
                decoration: ShapeDecoration(
                  color: swatch.primary,
                  // The selected colour morphs into a flower while turning.
                  shape: MorphBorder(morph: _circleToFlower, progress: t, rotation: t * 0.6),
                ),
                child: SizedBox.square(
                  dimension: 40,
                  child: Opacity(
                    opacity: t.clamp(0, 1),
                    child: Icon(AppIcons.check, color: swatch.onPrimary, size: 20),
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );
  }
}

class _VfoxSection extends StatefulWidget {
  const _VfoxSection();

  @override
  State<_VfoxSection> createState() => _VfoxSectionState();
}

class _VfoxSectionState extends State<_VfoxSection> {
  final TextEditingController _environment = TextEditingController();
  String? _environmentError;
  bool _initialized = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_initialized) {
      _initialized = true;
      _environment.text = formatEnvironmentLines(context.model.settings.settings.extraEnvironment);
    }
  }

  @override
  void dispose() {
    _environment.dispose();
    super.dispose();
  }

  Future<void> _saveEnvironment() async {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    final ScaffoldMessengerState messenger = ScaffoldMessenger.of(context);
    final ({List<int> invalidLines, Map<String, String> variables}) parsed = parseEnvironmentLines(_environment.text);
    if (parsed.invalidLines.isNotEmpty) {
      setState(() {
        _environmentError = l10n.settingsEnvironmentInvalid(parsed.invalidLines.length, parsed.invalidLines.join(', '));
      });
      return;
    }
    setState(() => _environmentError = null);
    await model.settings.update((s) => s.copyWith(extraEnvironment: parsed.variables));
    messenger.showSnackBar(SnackBar(content: Text(l10n.settingsSaved)));
  }

  Future<void> _chooseExecutable() async {
    final AppModel model = context.model;
    final String? path = await context.services.pickFile(initialDirectory: '/usr/local/bin');
    if (path != null) {
      await model.settings.update((s) => s.copyWith(vfoxExecutable: path));
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    final ThemeData theme = Theme.of(context);
    return ListenableBuilder(
      listenable: Listenable.merge(<Listenable>[model.vfox, model.settings]),
      builder: (BuildContext context, Widget? _) {
        final VfoxController vfox = model.vfox;
        final String source = vfox.executable == null
            ? l10n.settingsExecutableMissing
            : vfox.isCustomExecutable
            ? l10n.settingsExecutableCustom
            : l10n.settingsExecutableDetected;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            GroupHeader(title: l10n.settingsVfox),
            ListGroup(
              children: <Widget>[
                ListGroupItem(
                  wrapTrailing: true,
                  leading: const Icon(AppIcons.terminal),
                  title: Text(l10n.settingsExecutable),
                  subtitle: Text(
                    <String>[
                      vfox.executable ?? 'vfox',
                      source,
                      if (vfox.version != null) l10n.settingsVfoxVersion(vfox.version!),
                    ].join(' · '),
                  ),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: <Widget>[
                      if (vfox.availability == VfoxAvailability.checking)
                        const LoadingIndicator(size: 36)
                      else if (vfox.isCustomExecutable)
                        IconButton(
                          tooltip: l10n.settingsExecutableDetect,
                          onPressed: () =>
                              unawaited(model.settings.update((s) => s.copyWith(clearVfoxExecutable: true))),
                          icon: const Icon(AppIcons.refresh),
                        ),
                      const SizedBox(width: Gaps.xs),
                      FilledButton.tonal(onPressed: _chooseExecutable, child: Text(l10n.actionBrowse)),
                    ],
                  ),
                ),
                ListGroupItem(
                  leading: const Icon(AppIcons.environment),
                  title: Text(l10n.settingsEnvironment),
                  subtitle: Padding(
                    padding: const EdgeInsets.only(top: Gaps.s),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: <Widget>[
                        Text(l10n.settingsEnvironmentHelp),
                        const SizedBox(height: Gaps.m),
                        TextField(
                          controller: _environment,
                          minLines: 3,
                          maxLines: 8,
                          style: theme.textTheme.bodyMedium?.copyWith(fontFamily: 'monospace'),
                          decoration: InputDecoration(
                            hintText: 'VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node',
                            errorText: _environmentError,
                            errorMaxLines: 2,
                          ),
                        ),
                        const SizedBox(height: Gaps.s),
                        Align(
                          alignment: AlignmentDirectional.centerEnd,
                          child: FilledButton(onPressed: _saveEnvironment, child: Text(l10n.actionSave)),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ],
        );
      },
    );
  }
}

class _VfoxConfigSection extends StatelessWidget {
  const _VfoxConfigSection();

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    final VfoxConfigController config = model.vfoxConfig;
    return ListenableBuilder(
      listenable: Listenable.merge(<Listenable>[config, model.vfox]),
      builder: (BuildContext context, Widget? _) {
        final List<Widget> children;
        if (model.vfox.availability != VfoxAvailability.ready) {
          children = const <Widget>[];
        } else if (!config.hasValue) {
          final Object? error = config.error;
          children = <Widget>[
            if (error != null)
              ErrorMessage(title: l10n.settingsVfoxConfigError, error: error, onRetry: config.load)
            else
              const LoadingMessage(),
          ];
        } else {
          final List<ConfigEntry> other = <ConfigEntry>[
            for (final ConfigEntry entry in config.value!)
              if (!VfoxConfigKeys.known.contains(entry.key)) entry,
          ];
          children = <Widget>[
            ListGroup(
              children: <Widget>[
                _ConfigSwitch(
                  configKey: VfoxConfigKeys.proxyEnable,
                  icon: AppIcons.proxy,
                  title: l10n.configProxyEnable,
                ),
                _ConfigText(
                  configKey: VfoxConfigKeys.proxyUrl,
                  title: l10n.configProxyUrl,
                  hint: 'http://127.0.0.1:7890',
                ),
                _ConfigText(
                  configKey: VfoxConfigKeys.registry,
                  icon: AppIcons.registry,
                  title: l10n.configRegistry,
                  hint: l10n.configRegistryHint,
                ),
                _ConfigText(
                  configKey: VfoxConfigKeys.storage,
                  icon: AppIcons.storage,
                  title: l10n.configStorage,
                  hint: l10n.configStorageHint,
                  pickDirectory: true,
                ),
              ],
            ),
            const SizedBox(height: Gaps.l),
            ListGroup(
              children: <Widget>[
                _ConfigSwitch(
                  configKey: VfoxConfigKeys.legacyEnable,
                  icon: AppIcons.legacyFiles,
                  title: l10n.configLegacyEnable,
                  subtitle: l10n.configLegacyEnableHelp,
                ),
                _ConfigChoice(
                  configKey: VfoxConfigKeys.legacyStrategy,
                  title: l10n.configLegacyStrategy,
                  choices: <String, String>{
                    'specified': l10n.strategySpecified,
                    'latest_installed': l10n.strategyLatestInstalled,
                    'latest_available': l10n.strategyLatestAvailable,
                  },
                ),
                _ConfigSwitch(
                  configKey: VfoxConfigKeys.gitignore,
                  icon: AppIcons.gitignore,
                  title: l10n.configGitignore,
                  subtitle: l10n.configGitignoreHelp,
                ),
              ],
            ),
            const SizedBox(height: Gaps.l),
            ListGroup(
              children: <Widget>[
                _ConfigText(
                  configKey: VfoxConfigKeys.cache,
                  icon: AppIcons.cache,
                  title: l10n.configCache,
                  help: l10n.configCacheHelp,
                  hint: '12h',
                ),
                _ConfigText(
                  configKey: VfoxConfigKeys.httpTimeout,
                  icon: AppIcons.timeout,
                  title: l10n.configHttpTimeout,
                  help: l10n.configTimeoutHelp,
                  hint: '30s',
                ),
                _ConfigText(
                  configKey: VfoxConfigKeys.downloadTimeout,
                  title: l10n.configDownloadTimeout,
                  help: l10n.configTimeoutHelp,
                  hint: '30m',
                ),
              ],
            ),
            if (other.isNotEmpty) ...<Widget>[
              GroupHeader(title: l10n.configOther),
              ListGroup(
                children: <Widget>[
                  for (final ConfigEntry entry in other) _ConfigText(configKey: entry.key, title: entry.key),
                ],
              ),
            ],
          ];
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            GroupHeader(title: l10n.settingsVfoxConfig),
            Padding(
              padding: const EdgeInsetsDirectional.only(start: Gaps.l, bottom: Gaps.m),
              child: Text(
                l10n.settingsVfoxConfigHelp,
                style: Theme.of(context).textTheme.bodySmall
                    ?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant),
              ),
            ),
            ...children,
          ],
        );
      },
    );
  }
}

/// Saves a vfox setting and reports failures in a snack bar.
Future<void> _saveConfig(BuildContext context, Future<void> Function() save) async {
  final ScaffoldMessengerState messenger = ScaffoldMessenger.of(context);
  final AppLocalizations l10n = context.l10n;
  try {
    await save();
    messenger.showSnackBar(SnackBar(content: Text(l10n.settingsSaved), duration: const Duration(seconds: 2)));
  } on Exception catch (error) {
    messenger.showSnackBar(SnackBar(content: Text(describeError(error))));
  }
}

class _ConfigSwitch extends StatelessWidget {
  const _ConfigSwitch({required this.configKey, required this.title, this.icon, this.subtitle});

  final String configKey;
  final String title;
  final IconData? icon;
  final String? subtitle;

  @override
  Widget build(BuildContext context) {
    final VfoxConfigController config = context.model.vfoxConfig;
    final bool value = config.valueOf(configKey) == 'true';
    final bool saving = config.isSaving(configKey);
    void toggle() => unawaited(_saveConfig(context, () => config.set(configKey, '${!value}')));
    return ListGroupItem(
      leading: icon == null ? const SizedBox(width: 24) : Icon(icon),
      title: Text(title),
      subtitle: subtitle == null ? null : Text(subtitle!),
      trailing: Switch(value: value, onChanged: saving ? null : (_) => toggle()),
      onTap: saving ? null : toggle,
    );
  }
}

class _ConfigChoice extends StatelessWidget {
  const _ConfigChoice({required this.configKey, required this.title, required this.choices});

  final String configKey;
  final String title;
  final Map<String, String> choices;

  @override
  Widget build(BuildContext context) {
    final VfoxConfigController config = context.model.vfoxConfig;
    final String value = config.valueOf(configKey) ?? '';
    return ListGroupItem(
      wrapTrailing: true,
      leading: const SizedBox(width: 24),
      title: Text(title),
      trailing: DropdownMenu<String>(
        initialSelection: choices.containsKey(value) ? value : null,
        enabled: !config.isSaving(configKey),
        onSelected: (String? selection) {
          if (selection != null && selection != value) {
            unawaited(_saveConfig(context, () => config.set(configKey, selection)));
          }
        },
        dropdownMenuEntries: <DropdownMenuEntry<String>>[
          for (final MapEntry<String, String> choice in choices.entries)
            DropdownMenuEntry<String>(value: choice.key, label: choice.value),
        ],
      ),
    );
  }
}

class _ConfigText extends StatefulWidget {
  const _ConfigText({
    required this.configKey,
    required this.title,
    this.icon,
    this.hint,
    this.help,
    this.pickDirectory = false,
  });

  final String configKey;
  final String title;
  final IconData? icon;
  final String? hint;
  final String? help;
  final bool pickDirectory;

  @override
  State<_ConfigText> createState() => _ConfigTextState();
}

class _ConfigTextState extends State<_ConfigText> {
  final TextEditingController _text = TextEditingController();
  final FocusNode _focus = FocusNode();
  String? _saved;

  @override
  void initState() {
    super.initState();
    _focus.addListener(() {
      if (!_focus.hasFocus) {
        _save();
      }
    });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _sync();
  }

  @override
  void didUpdateWidget(_ConfigText oldWidget) {
    super.didUpdateWidget(oldWidget);
    _sync();
  }

  /// Shows the value stored by vfox unless the user is editing it.
  void _sync() {
    final String value = context.model.vfoxConfig.valueOf(widget.configKey) ?? '';
    if (value != _saved && !_focus.hasFocus) {
      _saved = value;
      _text.text = value;
    }
  }

  void _save() {
    final String value = _text.text.trim();
    if (value == _saved) {
      return;
    }
    // Submitting and then leaving the field must not save twice. If vfox
    // rejects the value, the reload shows the stored value again.
    _saved = value;
    final VfoxConfigController config = context.model.vfoxConfig;
    unawaited(
      _saveConfig(context, () => value.isEmpty ? config.reset(widget.configKey) : config.set(widget.configKey, value)),
    );
  }

  Future<void> _browse() async {
    final String? directory = await context.services.pickDirectory(
      initialDirectory: _text.text.isEmpty ? null : _text.text,
    );
    if (directory != null) {
      _text.text = directory;
      _save();
    }
  }

  @override
  void dispose() {
    _text.dispose();
    _focus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = context.l10n;
    final VfoxConfigController config = context.model.vfoxConfig;
    return ListGroupItem(
      wrapTrailing: true,
      leading: widget.icon == null ? const SizedBox(width: 24) : Icon(widget.icon),
      title: Text(widget.title),
      subtitle: widget.help == null ? null : Text(widget.help!),
      trailing: SizedBox(
        width: 300,
        child: TextField(
          controller: _text,
          focusNode: _focus,
          enabled: !config.isSaving(widget.configKey),
          onSubmitted: (_) => _save(),
          decoration: InputDecoration(
            isDense: true,
            hintText: widget.hint,
            suffixIcon: widget.pickDirectory
                ? IconButton(tooltip: l10n.actionBrowse, onPressed: _browse, icon: const Icon(AppIcons.folder))
                : (_text.text.isNotEmpty
                      ? IconButton(
                          tooltip: l10n.actionReset,
                          onPressed: () {
                            _text.clear();
                            _save();
                          },
                          icon: const Icon(AppIcons.close),
                        )
                      : null),
          ),
        ),
      ),
    );
  }
}

class _AboutSection extends StatelessWidget {
  const _AboutSection();

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = context.l10n;
    final PlatformServices services = context.services;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        GroupHeader(title: l10n.settingsAbout),
        ListGroup(
          children: <Widget>[
            ListGroupItem(
              leading: const ShapeAvatar(name: 'vfox+', icon: AppIcons.sdks, size: 40),
              title: Text(l10n.appTitle),
              subtitle: Text('${l10n.aboutVersion(appVersion)} · ${l10n.aboutLicense}'),
            ),
            ListGroupItem(
              leading: const Icon(AppIcons.sourceCode),
              title: Text(l10n.aboutSource),
              subtitle: Text(Links.repository.toString()),
              trailing: const Icon(AppIcons.openInNew),
              onTap: () => unawaited(services.openUrl(Links.repository)),
            ),
            ListGroupItem(
              leading: const Icon(AppIcons.docs),
              title: Text(l10n.aboutDocs),
              subtitle: Text(Links.vfoxDocs.toString()),
              trailing: const Icon(AppIcons.openInNew),
              onTap: () => unawaited(services.openUrl(Links.vfoxDocs)),
            ),
            ListGroupItem(
              leading: const Icon(AppIcons.license),
              title: Text(l10n.aboutLicenses),
              trailing: const Icon(AppIcons.chevronRight),
              onTap: () => showLicensePage(
                context: context,
                applicationName: l10n.appTitle,
                applicationVersion: appVersion,
                applicationLegalese: l10n.aboutLicense,
              ),
            ),
          ],
        ),
      ],
    );
  }
}
