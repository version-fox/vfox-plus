// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'vfox+';

  @override
  String get actionRefresh => 'Refresh';

  @override
  String get actionCancel => 'Cancel';

  @override
  String get actionRetry => 'Retry';

  @override
  String get actionClose => 'Close';

  @override
  String get actionSave => 'Save';

  @override
  String get actionReset => 'Reset to default';

  @override
  String get actionBrowse => 'Browse…';

  @override
  String get actionCopy => 'Copy';

  @override
  String get actionCopied => 'Copied to the clipboard';

  @override
  String get actionOpenHomepage => 'Open homepage';

  @override
  String get actionShowDetails => 'Details';

  @override
  String get loading => 'Loading…';

  @override
  String get errorTitle => 'Something went wrong';

  @override
  String get navHome => 'Home';

  @override
  String get navSdks => 'SDKs';

  @override
  String get navPlugins => 'Plugins';

  @override
  String get navTasks => 'Tasks';

  @override
  String get navSettings => 'Settings';

  @override
  String get navExpand => 'Show labels';

  @override
  String get navCollapse => 'Hide labels';

  @override
  String get navAddPlugin => 'Add plugin';

  @override
  String get vfoxCheckingTitle => 'Looking for vfox…';

  @override
  String get vfoxMissingTitle => 'vfox is not installed';

  @override
  String get vfoxMissingBody =>
      'vfox+ manages your SDKs through the vfox command line tool. Install vfox, or show vfox+ where it is.';

  @override
  String get vfoxBrokenTitle => 'vfox does not respond';

  @override
  String vfoxBrokenBody(String path) {
    return 'The executable $path did not report a vfox version.';
  }

  @override
  String get vfoxInstallGuide => 'Installation guide';

  @override
  String get vfoxLocate => 'Locate vfox…';

  @override
  String homeSubtitle(String version) {
    return 'vfox $version';
  }

  @override
  String homeStatPlugins(int count) {
    String _temp0 = intl.Intl.pluralLogic(count, locale: localeName, other: 'Plugins', one: 'Plugin');
    return '$_temp0';
  }

  @override
  String homeStatVersions(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Installed versions',
      one: 'Installed version',
    );
    return '$_temp0';
  }

  @override
  String get homeStatActive => 'In use';

  @override
  String get homeActiveTitle => 'Versions in use';

  @override
  String get homeActiveEmpty => 'No SDK version is in use yet. Pick one on the SDKs page.';

  @override
  String get homeShellTitle => 'Use the versions in a terminal';

  @override
  String get homeShellBody =>
      'Add this line to ~/.bashrc so that new shells pick up the versions chosen here. Other shells are covered in the vfox documentation.';

  @override
  String get homeExploreTitle => 'Add more SDKs';

  @override
  String get homeExploreBody => 'Plugins add Node.js, Java, Python, Go and many more to vfox.';

  @override
  String get homeDocsTitle => 'vfox documentation';

  @override
  String get homeDocsBody => 'Learn about scopes, plugins and configuration files.';

  @override
  String get sdksSearchHint => 'Search installed plugins';

  @override
  String get sdksEmptyTitle => 'No plugins yet';

  @override
  String get sdksEmptyBody => 'A plugin teaches vfox how to install an SDK. Add one to get started.';

  @override
  String sdksNoMatch(String query) {
    return 'No plugin matches “$query”.';
  }

  @override
  String sdksCurrent(String version) {
    return 'In use: $version';
  }

  @override
  String get sdksNoCurrent => 'No version in use';

  @override
  String sdksVersionCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count versions installed',
      one: '1 version installed',
      zero: 'No version installed',
    );
    return '$_temp0';
  }

  @override
  String get sdksSelectPrompt => 'Select an SDK to manage its versions.';

  @override
  String get sdksLoadError => 'Could not list the installed SDKs.';

  @override
  String sdkPluginVersion(String version) {
    return 'Plugin $version';
  }

  @override
  String get sdkTabInstalled => 'Installed';

  @override
  String get sdkTabAvailable => 'Available';

  @override
  String get sdkUpdatePlugin => 'Update plugin';

  @override
  String get sdkRemovePlugin => 'Remove plugin';

  @override
  String get sdkMoreActions => 'More actions';

  @override
  String sdkRemoveTitle(String name) {
    return 'Remove $name?';
  }

  @override
  String sdkRemoveBody(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'The plugin and its $count installed versions will be deleted.',
      one: 'The plugin and its installed version will be deleted.',
      zero: 'The plugin will be deleted.',
    );
    return '$_temp0';
  }

  @override
  String get sdkRemoveConfirm => 'Remove';

  @override
  String sdkInstalledEmpty(String name) {
    return 'No version of $name is installed yet.';
  }

  @override
  String get sdkBrowseVersions => 'Browse versions';

  @override
  String get sdkInfoError => 'Could not read the plugin information.';

  @override
  String get versionInUse => 'In use';

  @override
  String get versionUse => 'Use';

  @override
  String get versionUseOptions => 'More ways to use this version';

  @override
  String get versionUseGlobally => 'Use as global default';

  @override
  String get versionUseInProject => 'Use in a project…';

  @override
  String get versionUnuse => 'Stop using';

  @override
  String get versionCopyPath => 'Copy installation path';

  @override
  String get versionUninstall => 'Uninstall';

  @override
  String versionUninstallTitle(String name, String version) {
    return 'Uninstall $name $version?';
  }

  @override
  String get versionUninstallBody => 'The files of this version will be deleted.';

  @override
  String get versionUninstallCurrentBody =>
      'This version is in use. After it is deleted, vfox switches to another installed version, if there is one.';

  @override
  String get versionInstall => 'Install';

  @override
  String get versionInstalled => 'Installed';

  @override
  String versionMoreActions(String version) {
    return 'More actions for $version';
  }

  @override
  String get availableSearchHint => 'Filter versions';

  @override
  String get availableFilterAll => 'All';

  @override
  String get availableFilterTagged => 'Tagged';

  @override
  String get availableFilterInstalled => 'Installed';

  @override
  String get availableLoading => 'Fetching the available versions…';

  @override
  String get availableEmpty => 'No version matches.';

  @override
  String availableError(String name) {
    return 'Could not list the versions of $name.';
  }

  @override
  String availableCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(count, locale: localeName, other: '$count versions', one: '1 version');
    return '$_temp0';
  }

  @override
  String useProjectTitle(String name, String version) {
    return 'Use $name $version in a project';
  }

  @override
  String get useProjectBody =>
      'vfox records the version in .vfox.toml in the project directory and links the SDK into .vfox/sdks.';

  @override
  String get useProjectDirectory => 'Project directory';

  @override
  String get useProjectInvalid => 'Choose an existing directory.';

  @override
  String get useProjectConfirm => 'Use in project';

  @override
  String get pluginsTitle => 'Plugins';

  @override
  String get pluginsSubtitle => 'Every plugin adds a language or tool to vfox.';

  @override
  String get pluginsSearchHint => 'Search plugins';

  @override
  String get pluginsFilterAll => 'All';

  @override
  String get pluginsFilterOfficial => 'Official';

  @override
  String get pluginsFilterCommunity => 'Community';

  @override
  String get pluginsOfficial => 'Official plugin';

  @override
  String get pluginsAdd => 'Add';

  @override
  String get pluginsAdded => 'Added';

  @override
  String get pluginsAddCustom => 'Add from source…';

  @override
  String get pluginsUpdateAll => 'Update all plugins';

  @override
  String get pluginsLoadError => 'Could not load the plugin registry.';

  @override
  String get pluginsNoMatch => 'No plugin matches your search.';

  @override
  String get pluginsCustomTitle => 'Add a plugin from source';

  @override
  String get pluginsCustomSource => 'Source';

  @override
  String get pluginsCustomSourceHelp => 'URL or path of a plugin archive (.zip) or script (.lua).';

  @override
  String get pluginsCustomFileType => 'vfox plugins';

  @override
  String get pluginsCustomAlias => 'Name (optional)';

  @override
  String get pluginsCustomAliasHelp => 'Install the plugin under this name instead of its own.';

  @override
  String get pluginsCustomInvalidName => 'Use letters, digits, dots, dashes and underscores only.';

  @override
  String get pluginsCustomConfirm => 'Add plugin';

  @override
  String get pluginsCustomRequired => 'Enter a URL or a path.';

  @override
  String get tasksEmptyTitle => 'Nothing is running';

  @override
  String get tasksEmptyBody => 'Installations and other changes appear here while vfox works on them.';

  @override
  String get tasksClearFinished => 'Clear finished';

  @override
  String get taskStatusQueued => 'Waiting';

  @override
  String get taskStatusRunning => 'Running';

  @override
  String get taskStatusSucceeded => 'Done';

  @override
  String get taskStatusFailed => 'Failed';

  @override
  String get taskStatusCancelled => 'Cancelled';

  @override
  String taskDownloading(int percent) {
    return 'Downloading… $percent%';
  }

  @override
  String get taskShowOutput => 'Show output';

  @override
  String get taskHideOutput => 'Hide output';

  @override
  String get taskCopyOutput => 'Copy output';

  @override
  String get taskNoOutput => 'No output yet.';

  @override
  String taskInstall(String subject) {
    return 'Install $subject';
  }

  @override
  String taskUninstall(String subject) {
    return 'Uninstall $subject';
  }

  @override
  String taskUse(String subject) {
    return 'Use $subject';
  }

  @override
  String taskUnuse(String subject) {
    return 'Stop using $subject';
  }

  @override
  String taskAddPlugin(String subject) {
    return 'Add plugin $subject';
  }

  @override
  String get taskAddPluginFromSource => 'Add plugin from source';

  @override
  String taskRemovePlugin(String subject) {
    return 'Remove plugin $subject';
  }

  @override
  String taskUpdatePlugin(String subject) {
    return 'Update plugin $subject';
  }

  @override
  String get taskUpdateAllPlugins => 'Update all plugins';

  @override
  String taskSucceeded(String title) {
    return '$title: done';
  }

  @override
  String taskFailed(String title) {
    return '$title: failed';
  }

  @override
  String get taskView => 'View';

  @override
  String get settingsAppearance => 'Appearance';

  @override
  String get settingsTheme => 'Theme';

  @override
  String get themeSystem => 'System';

  @override
  String get themeLight => 'Light';

  @override
  String get themeDark => 'Dark';

  @override
  String get settingsColor => 'Color';

  @override
  String get settingsSystemColor => 'Use the desktop accent color';

  @override
  String get settingsSystemColorUnavailable =>
      'The desktop does not share an accent color, so the color below is used.';

  @override
  String settingsColorOption(int index) {
    return 'Accent color $index';
  }

  @override
  String get settingsLanguage => 'Language';

  @override
  String get languageSystem => 'System';

  @override
  String get settingsVfox => 'vfox';

  @override
  String get settingsExecutable => 'Executable';

  @override
  String get settingsExecutableDetected => 'Detected automatically';

  @override
  String get settingsExecutableCustom => 'Chosen manually';

  @override
  String get settingsExecutableMissing => 'Not found';

  @override
  String get settingsExecutableDetect => 'Detect automatically';

  @override
  String settingsVfoxVersion(String version) {
    return 'Version $version';
  }

  @override
  String get settingsEnvironment => 'Environment variables';

  @override
  String get settingsEnvironmentHelp =>
      'Passed to vfox and its plugins, one KEY=VALUE per line. Useful for download mirrors such as VFOX_NODEJS_MIRROR, which a desktop launcher does not read from your shell profile.';

  @override
  String settingsEnvironmentInvalid(int count, String lines) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: 'Lines $lines are not KEY=VALUE.',
      one: 'Line $lines is not KEY=VALUE.',
    );
    return '$_temp0';
  }

  @override
  String get settingsSaved => 'Saved';

  @override
  String get settingsVfoxConfig => 'vfox configuration';

  @override
  String get settingsVfoxConfigHelp => 'Stored by vfox in ~/.vfox/config.yaml and shared with the command line.';

  @override
  String get settingsVfoxConfigError => 'Could not read the vfox configuration.';

  @override
  String get configProxyEnable => 'Download through a proxy';

  @override
  String get configProxyUrl => 'Proxy URL';

  @override
  String get configRegistry => 'Plugin registry';

  @override
  String get configRegistryHint => 'Official registry';

  @override
  String get configStorage => 'SDK storage directory';

  @override
  String get configStorageHint => 'Inside ~/.vfox';

  @override
  String get configLegacyEnable => 'Read legacy version files';

  @override
  String get configLegacyEnableHelp => 'Such as .nvmrc, .node-version, .python-version and .sdkmanrc.';

  @override
  String get configLegacyStrategy => 'Version used from legacy files';

  @override
  String get strategySpecified => 'As written';

  @override
  String get strategyLatestInstalled => 'Latest installed';

  @override
  String get strategyLatestAvailable => 'Latest available';

  @override
  String get configCache => 'Cache version lists for';

  @override
  String get configCacheHelp => 'A duration such as 12h. -1 keeps the lists forever, 0 disables the cache.';

  @override
  String get configHttpTimeout => 'Request timeout';

  @override
  String get configDownloadTimeout => 'Download timeout';

  @override
  String get configTimeoutHelp => 'A duration such as 30s or 10m. 0s uses the default.';

  @override
  String get configGitignore => 'Add .vfox/ to .gitignore';

  @override
  String get configGitignoreHelp => 'When a version is pinned in a project.';

  @override
  String get configOther => 'Other settings';

  @override
  String get settingsAbout => 'About';

  @override
  String aboutVersion(String version) {
    return 'Version $version';
  }

  @override
  String get aboutLicense => 'Licensed under the Apache License 2.0.';

  @override
  String get aboutSource => 'Source code';

  @override
  String get aboutLicenses => 'Open-source licenses';

  @override
  String get aboutDocs => 'vfox documentation';
}
