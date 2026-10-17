import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale) : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate = _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates = <LocalizationsDelegate<dynamic>>[
    delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
  ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[Locale('en'), Locale('zh')];

  /// Name of the application.
  ///
  /// In en, this message translates to:
  /// **'vfox+'**
  String get appTitle;

  /// Reloads data from vfox.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get actionRefresh;

  /// Dismisses a dialog or stops a task.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get actionCancel;

  /// Runs a failed operation again.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get actionRetry;

  /// Closes a dialog or panel.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get actionClose;

  /// Stores an edited value.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get actionSave;

  /// Restores the default value of a setting.
  ///
  /// In en, this message translates to:
  /// **'Reset to default'**
  String get actionReset;

  /// Opens a file or folder chooser.
  ///
  /// In en, this message translates to:
  /// **'Browse…'**
  String get actionBrowse;

  /// Copies text to the clipboard.
  ///
  /// In en, this message translates to:
  /// **'Copy'**
  String get actionCopy;

  /// Confirmation after copying text.
  ///
  /// In en, this message translates to:
  /// **'Copied to the clipboard'**
  String get actionCopied;

  /// Opens the homepage of a plugin in the web browser.
  ///
  /// In en, this message translates to:
  /// **'Open homepage'**
  String get actionOpenHomepage;

  /// Shows more information about an error.
  ///
  /// In en, this message translates to:
  /// **'Details'**
  String get actionShowDetails;

  /// Accessibility label of loading indicators.
  ///
  /// In en, this message translates to:
  /// **'Loading…'**
  String get loading;

  /// Title of an error message without more specific information.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong'**
  String get errorTitle;

  /// Navigation destination: overview page.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get navHome;

  /// Navigation destination: installed SDKs and their versions.
  ///
  /// In en, this message translates to:
  /// **'SDKs'**
  String get navSdks;

  /// Navigation destination: the plugin registry.
  ///
  /// In en, this message translates to:
  /// **'Plugins'**
  String get navPlugins;

  /// Navigation destination: running and finished operations.
  ///
  /// In en, this message translates to:
  /// **'Tasks'**
  String get navTasks;

  /// Navigation destination: settings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get navSettings;

  /// Tooltip of the button that expands the navigation rail.
  ///
  /// In en, this message translates to:
  /// **'Show labels'**
  String get navExpand;

  /// Tooltip of the button that collapses the navigation rail.
  ///
  /// In en, this message translates to:
  /// **'Hide labels'**
  String get navCollapse;

  /// Floating action button that opens the plugin registry.
  ///
  /// In en, this message translates to:
  /// **'Add plugin'**
  String get navAddPlugin;

  /// Shown while the application checks the vfox executable.
  ///
  /// In en, this message translates to:
  /// **'Looking for vfox…'**
  String get vfoxCheckingTitle;

  /// Title shown when no vfox executable was found.
  ///
  /// In en, this message translates to:
  /// **'vfox is not installed'**
  String get vfoxMissingTitle;

  /// Explanation shown when no vfox executable was found.
  ///
  /// In en, this message translates to:
  /// **'vfox+ manages your SDKs through the vfox command line tool. Install vfox, or show vfox+ where it is.'**
  String get vfoxMissingBody;

  /// Title shown when the vfox executable does not work.
  ///
  /// In en, this message translates to:
  /// **'vfox does not respond'**
  String get vfoxBrokenTitle;

  /// Explanation shown when the vfox executable does not work.
  ///
  /// In en, this message translates to:
  /// **'The executable {path} did not report a vfox version.'**
  String vfoxBrokenBody(String path);

  /// Opens the vfox installation documentation.
  ///
  /// In en, this message translates to:
  /// **'Installation guide'**
  String get vfoxInstallGuide;

  /// Lets the user choose the vfox executable.
  ///
  /// In en, this message translates to:
  /// **'Locate vfox…'**
  String get vfoxLocate;

  /// Subtitle of the home page.
  ///
  /// In en, this message translates to:
  /// **'vfox {version}'**
  String homeSubtitle(String version);

  /// Label below the number of installed plugins.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{Plugin} other{Plugins}}'**
  String homeStatPlugins(int count);

  /// Label below the number of installed SDK versions.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{Installed version} other{Installed versions}}'**
  String homeStatVersions(int count);

  /// Label below the number of SDKs that have a current version.
  ///
  /// In en, this message translates to:
  /// **'In use'**
  String get homeStatActive;

  /// Heading of the list of current SDK versions.
  ///
  /// In en, this message translates to:
  /// **'Versions in use'**
  String get homeActiveTitle;

  /// Shown when no SDK has a current version.
  ///
  /// In en, this message translates to:
  /// **'No SDK version is in use yet. Pick one on the SDKs page.'**
  String get homeActiveEmpty;

  /// Heading of the tip about shell integration.
  ///
  /// In en, this message translates to:
  /// **'Use the versions in a terminal'**
  String get homeShellTitle;

  /// Explains how to activate vfox in a shell.
  ///
  /// In en, this message translates to:
  /// **'Add this line to ~/.bashrc so that new shells pick up the versions chosen here. Other shells are covered in the vfox documentation.'**
  String get homeShellBody;

  /// Heading of the shortcut to the plugin registry.
  ///
  /// In en, this message translates to:
  /// **'Add more SDKs'**
  String get homeExploreTitle;

  /// Text of the shortcut to the plugin registry.
  ///
  /// In en, this message translates to:
  /// **'Plugins add Node.js, Java, Python, Go and many more to vfox.'**
  String get homeExploreBody;

  /// Heading of the shortcut to the vfox documentation.
  ///
  /// In en, this message translates to:
  /// **'vfox documentation'**
  String get homeDocsTitle;

  /// Text of the shortcut to the vfox documentation.
  ///
  /// In en, this message translates to:
  /// **'Learn about scopes, plugins and configuration files.'**
  String get homeDocsBody;

  /// Hint of the search field on the SDKs page.
  ///
  /// In en, this message translates to:
  /// **'Search installed plugins'**
  String get sdksSearchHint;

  /// Shown when no plugin is installed.
  ///
  /// In en, this message translates to:
  /// **'No plugins yet'**
  String get sdksEmptyTitle;

  /// Shown when no plugin is installed.
  ///
  /// In en, this message translates to:
  /// **'A plugin teaches vfox how to install an SDK. Add one to get started.'**
  String get sdksEmptyBody;

  /// Shown when the search on the SDKs page has no result.
  ///
  /// In en, this message translates to:
  /// **'No plugin matches “{query}”.'**
  String sdksNoMatch(String query);

  /// The current version of an SDK.
  ///
  /// In en, this message translates to:
  /// **'In use: {version}'**
  String sdksCurrent(String version);

  /// Shown when an SDK has no current version.
  ///
  /// In en, this message translates to:
  /// **'No version in use'**
  String get sdksNoCurrent;

  /// Number of installed versions of an SDK.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =0{No version installed} =1{1 version installed} other{{count} versions installed}}'**
  String sdksVersionCount(int count);

  /// Placeholder of the detail pane when no SDK is selected.
  ///
  /// In en, this message translates to:
  /// **'Select an SDK to manage its versions.'**
  String get sdksSelectPrompt;

  /// Shown when vfox list or vfox current fails.
  ///
  /// In en, this message translates to:
  /// **'Could not list the installed SDKs.'**
  String get sdksLoadError;

  /// Version of the plugin itself, not of the SDK.
  ///
  /// In en, this message translates to:
  /// **'Plugin {version}'**
  String sdkPluginVersion(String version);

  /// Tab with the installed versions of an SDK.
  ///
  /// In en, this message translates to:
  /// **'Installed'**
  String get sdkTabInstalled;

  /// Tab with the versions that can be installed.
  ///
  /// In en, this message translates to:
  /// **'Available'**
  String get sdkTabAvailable;

  /// Updates the plugin of an SDK.
  ///
  /// In en, this message translates to:
  /// **'Update plugin'**
  String get sdkUpdatePlugin;

  /// Removes the plugin of an SDK and all its versions.
  ///
  /// In en, this message translates to:
  /// **'Remove plugin'**
  String get sdkRemovePlugin;

  /// Tooltip of the overflow menu of an SDK.
  ///
  /// In en, this message translates to:
  /// **'More actions'**
  String get sdkMoreActions;

  /// Title of the confirmation before removing a plugin.
  ///
  /// In en, this message translates to:
  /// **'Remove {name}?'**
  String sdkRemoveTitle(String name);

  /// Explanation in the confirmation before removing a plugin.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =0{The plugin will be deleted.} =1{The plugin and its installed version will be deleted.} other{The plugin and its {count} installed versions will be deleted.}}'**
  String sdkRemoveBody(int count);

  /// Confirms removing a plugin.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get sdkRemoveConfirm;

  /// Shown when an SDK has no installed version.
  ///
  /// In en, this message translates to:
  /// **'No version of {name} is installed yet.'**
  String sdkInstalledEmpty(String name);

  /// Switches to the available versions.
  ///
  /// In en, this message translates to:
  /// **'Browse versions'**
  String get sdkBrowseVersions;

  /// Shown when vfox info fails.
  ///
  /// In en, this message translates to:
  /// **'Could not read the plugin information.'**
  String get sdkInfoError;

  /// Badge of the current version.
  ///
  /// In en, this message translates to:
  /// **'In use'**
  String get versionInUse;

  /// Makes a version the global default.
  ///
  /// In en, this message translates to:
  /// **'Use'**
  String get versionUse;

  /// Tooltip of the menu button of the use split button.
  ///
  /// In en, this message translates to:
  /// **'More ways to use this version'**
  String get versionUseOptions;

  /// Makes a version the global default.
  ///
  /// In en, this message translates to:
  /// **'Use as global default'**
  String get versionUseGlobally;

  /// Pins a version in a project directory.
  ///
  /// In en, this message translates to:
  /// **'Use in a project…'**
  String get versionUseInProject;

  /// Removes the global default version.
  ///
  /// In en, this message translates to:
  /// **'Stop using'**
  String get versionUnuse;

  /// Copies the directory of a version.
  ///
  /// In en, this message translates to:
  /// **'Copy installation path'**
  String get versionCopyPath;

  /// Uninstalls a version.
  ///
  /// In en, this message translates to:
  /// **'Uninstall'**
  String get versionUninstall;

  /// Title of the confirmation before uninstalling a version.
  ///
  /// In en, this message translates to:
  /// **'Uninstall {name} {version}?'**
  String versionUninstallTitle(String name, String version);

  /// Explanation in the confirmation before uninstalling a version.
  ///
  /// In en, this message translates to:
  /// **'The files of this version will be deleted.'**
  String get versionUninstallBody;

  /// Explanation in the confirmation before uninstalling the current version.
  ///
  /// In en, this message translates to:
  /// **'This version is in use. After it is deleted, vfox switches to another installed version, if there is one.'**
  String get versionUninstallCurrentBody;

  /// Installs a version.
  ///
  /// In en, this message translates to:
  /// **'Install'**
  String get versionInstall;

  /// Shown instead of the install button for installed versions.
  ///
  /// In en, this message translates to:
  /// **'Installed'**
  String get versionInstalled;

  /// Tooltip of the overflow menu of a version.
  ///
  /// In en, this message translates to:
  /// **'More actions for {version}'**
  String versionMoreActions(String version);

  /// Hint of the filter field of the available versions.
  ///
  /// In en, this message translates to:
  /// **'Filter versions'**
  String get availableSearchHint;

  /// Shows every available version.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get availableFilterAll;

  /// Shows only versions with a remark such as LTS.
  ///
  /// In en, this message translates to:
  /// **'Tagged'**
  String get availableFilterTagged;

  /// Shows only installed versions.
  ///
  /// In en, this message translates to:
  /// **'Installed'**
  String get availableFilterInstalled;

  /// Shown while vfox search runs.
  ///
  /// In en, this message translates to:
  /// **'Fetching the available versions…'**
  String get availableLoading;

  /// Shown when the version filter has no result.
  ///
  /// In en, this message translates to:
  /// **'No version matches.'**
  String get availableEmpty;

  /// Shown when vfox search fails.
  ///
  /// In en, this message translates to:
  /// **'Could not list the versions of {name}.'**
  String availableError(String name);

  /// Number of listed versions.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 version} other{{count} versions}}'**
  String availableCount(int count);

  /// Title of the dialog that pins a version in a project.
  ///
  /// In en, this message translates to:
  /// **'Use {name} {version} in a project'**
  String useProjectTitle(String name, String version);

  /// Explains what pinning a version in a project does.
  ///
  /// In en, this message translates to:
  /// **'vfox records the version in .vfox.toml in the project directory and links the SDK into .vfox/sdks.'**
  String get useProjectBody;

  /// Label of the project directory field.
  ///
  /// In en, this message translates to:
  /// **'Project directory'**
  String get useProjectDirectory;

  /// Validation error of the project directory field.
  ///
  /// In en, this message translates to:
  /// **'Choose an existing directory.'**
  String get useProjectInvalid;

  /// Confirms pinning a version in a project.
  ///
  /// In en, this message translates to:
  /// **'Use in project'**
  String get useProjectConfirm;

  /// Title of the plugin registry page.
  ///
  /// In en, this message translates to:
  /// **'Plugins'**
  String get pluginsTitle;

  /// Subtitle of the plugin registry page.
  ///
  /// In en, this message translates to:
  /// **'Every plugin adds a language or tool to vfox.'**
  String get pluginsSubtitle;

  /// Hint of the search field of the plugin registry.
  ///
  /// In en, this message translates to:
  /// **'Search plugins'**
  String get pluginsSearchHint;

  /// Shows all plugins.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get pluginsFilterAll;

  /// Shows plugins maintained by the vfox team.
  ///
  /// In en, this message translates to:
  /// **'Official'**
  String get pluginsFilterOfficial;

  /// Shows plugins maintained by the community.
  ///
  /// In en, this message translates to:
  /// **'Community'**
  String get pluginsFilterCommunity;

  /// Tooltip of the badge of official plugins.
  ///
  /// In en, this message translates to:
  /// **'Official plugin'**
  String get pluginsOfficial;

  /// Adds a plugin.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get pluginsAdd;

  /// Shown for plugins that are already added.
  ///
  /// In en, this message translates to:
  /// **'Added'**
  String get pluginsAdded;

  /// Adds a plugin from a URL or a local archive.
  ///
  /// In en, this message translates to:
  /// **'Add from source…'**
  String get pluginsAddCustom;

  /// Updates every installed plugin.
  ///
  /// In en, this message translates to:
  /// **'Update all plugins'**
  String get pluginsUpdateAll;

  /// Shown when vfox available fails.
  ///
  /// In en, this message translates to:
  /// **'Could not load the plugin registry.'**
  String get pluginsLoadError;

  /// Shown when the plugin search has no result.
  ///
  /// In en, this message translates to:
  /// **'No plugin matches your search.'**
  String get pluginsNoMatch;

  /// Title of the dialog that adds a plugin from a source.
  ///
  /// In en, this message translates to:
  /// **'Add a plugin from source'**
  String get pluginsCustomTitle;

  /// Label of the source field.
  ///
  /// In en, this message translates to:
  /// **'Source'**
  String get pluginsCustomSource;

  /// Help text of the source field.
  ///
  /// In en, this message translates to:
  /// **'URL or path of a plugin archive (.zip) or script (.lua).'**
  String get pluginsCustomSourceHelp;

  /// Name of the file type in the file chooser for plugin sources.
  ///
  /// In en, this message translates to:
  /// **'vfox plugins'**
  String get pluginsCustomFileType;

  /// Label of the alias field.
  ///
  /// In en, this message translates to:
  /// **'Name (optional)'**
  String get pluginsCustomAlias;

  /// Help text of the alias field.
  ///
  /// In en, this message translates to:
  /// **'Install the plugin under this name instead of its own.'**
  String get pluginsCustomAliasHelp;

  /// Validation error of the alias field.
  ///
  /// In en, this message translates to:
  /// **'Use letters, digits, dots, dashes and underscores only.'**
  String get pluginsCustomInvalidName;

  /// Confirms adding a plugin from a source.
  ///
  /// In en, this message translates to:
  /// **'Add plugin'**
  String get pluginsCustomConfirm;

  /// Validation error of the source field.
  ///
  /// In en, this message translates to:
  /// **'Enter a URL or a path.'**
  String get pluginsCustomRequired;

  /// Shown when there is no task.
  ///
  /// In en, this message translates to:
  /// **'Nothing is running'**
  String get tasksEmptyTitle;

  /// Shown when there is no task.
  ///
  /// In en, this message translates to:
  /// **'Installations and other changes appear here while vfox works on them.'**
  String get tasksEmptyBody;

  /// Removes finished tasks from the list.
  ///
  /// In en, this message translates to:
  /// **'Clear finished'**
  String get tasksClearFinished;

  /// Status of a task that waits for other tasks.
  ///
  /// In en, this message translates to:
  /// **'Waiting'**
  String get taskStatusQueued;

  /// Status of a running task.
  ///
  /// In en, this message translates to:
  /// **'Running'**
  String get taskStatusRunning;

  /// Status of a successful task.
  ///
  /// In en, this message translates to:
  /// **'Done'**
  String get taskStatusSucceeded;

  /// Status of a failed task.
  ///
  /// In en, this message translates to:
  /// **'Failed'**
  String get taskStatusFailed;

  /// Status of a cancelled task.
  ///
  /// In en, this message translates to:
  /// **'Cancelled'**
  String get taskStatusCancelled;

  /// Download progress of a task.
  ///
  /// In en, this message translates to:
  /// **'Downloading… {percent}%'**
  String taskDownloading(int percent);

  /// Expands the output of a task.
  ///
  /// In en, this message translates to:
  /// **'Show output'**
  String get taskShowOutput;

  /// Collapses the output of a task.
  ///
  /// In en, this message translates to:
  /// **'Hide output'**
  String get taskHideOutput;

  /// Copies the output of a task.
  ///
  /// In en, this message translates to:
  /// **'Copy output'**
  String get taskCopyOutput;

  /// Shown when a task has not printed anything.
  ///
  /// In en, this message translates to:
  /// **'No output yet.'**
  String get taskNoOutput;

  /// Title of an install task.
  ///
  /// In en, this message translates to:
  /// **'Install {subject}'**
  String taskInstall(String subject);

  /// Title of an uninstall task.
  ///
  /// In en, this message translates to:
  /// **'Uninstall {subject}'**
  String taskUninstall(String subject);

  /// Title of a task that selects a version.
  ///
  /// In en, this message translates to:
  /// **'Use {subject}'**
  String taskUse(String subject);

  /// Title of a task that clears the current version.
  ///
  /// In en, this message translates to:
  /// **'Stop using {subject}'**
  String taskUnuse(String subject);

  /// Title of a task that adds a plugin.
  ///
  /// In en, this message translates to:
  /// **'Add plugin {subject}'**
  String taskAddPlugin(String subject);

  /// Title of a task that adds a plugin from a URL without a name.
  ///
  /// In en, this message translates to:
  /// **'Add plugin from source'**
  String get taskAddPluginFromSource;

  /// Title of a task that removes a plugin.
  ///
  /// In en, this message translates to:
  /// **'Remove plugin {subject}'**
  String taskRemovePlugin(String subject);

  /// Title of a task that updates a plugin.
  ///
  /// In en, this message translates to:
  /// **'Update plugin {subject}'**
  String taskUpdatePlugin(String subject);

  /// Title of a task that updates every plugin.
  ///
  /// In en, this message translates to:
  /// **'Update all plugins'**
  String get taskUpdateAllPlugins;

  /// Message after a task finished.
  ///
  /// In en, this message translates to:
  /// **'{title}: done'**
  String taskSucceeded(String title);

  /// Message after a task failed.
  ///
  /// In en, this message translates to:
  /// **'{title}: failed'**
  String taskFailed(String title);

  /// Opens the task list from a message.
  ///
  /// In en, this message translates to:
  /// **'View'**
  String get taskView;

  /// Heading of the appearance settings.
  ///
  /// In en, this message translates to:
  /// **'Appearance'**
  String get settingsAppearance;

  /// Light or dark appearance.
  ///
  /// In en, this message translates to:
  /// **'Theme'**
  String get settingsTheme;

  /// Follow the desktop appearance.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get themeSystem;

  /// Light appearance.
  ///
  /// In en, this message translates to:
  /// **'Light'**
  String get themeLight;

  /// Dark appearance.
  ///
  /// In en, this message translates to:
  /// **'Dark'**
  String get themeDark;

  /// Accent color setting.
  ///
  /// In en, this message translates to:
  /// **'Color'**
  String get settingsColor;

  /// Derive the colors from the accent color of the desktop.
  ///
  /// In en, this message translates to:
  /// **'Use the desktop accent color'**
  String get settingsSystemColor;

  /// Shown when no desktop accent color is available.
  ///
  /// In en, this message translates to:
  /// **'The desktop does not share an accent color, so the color below is used.'**
  String get settingsSystemColorUnavailable;

  /// Accessibility label of a color swatch.
  ///
  /// In en, this message translates to:
  /// **'Accent color {index}'**
  String settingsColorOption(int index);

  /// User interface language setting.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get settingsLanguage;

  /// Follow the desktop language.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get languageSystem;

  /// Heading of the settings about the vfox executable.
  ///
  /// In en, this message translates to:
  /// **'vfox'**
  String get settingsVfox;

  /// Path of the vfox executable.
  ///
  /// In en, this message translates to:
  /// **'Executable'**
  String get settingsExecutable;

  /// The executable was found in PATH.
  ///
  /// In en, this message translates to:
  /// **'Detected automatically'**
  String get settingsExecutableDetected;

  /// The executable was chosen by the user.
  ///
  /// In en, this message translates to:
  /// **'Chosen manually'**
  String get settingsExecutableCustom;

  /// No executable was found.
  ///
  /// In en, this message translates to:
  /// **'Not found'**
  String get settingsExecutableMissing;

  /// Forgets the manually chosen executable.
  ///
  /// In en, this message translates to:
  /// **'Detect automatically'**
  String get settingsExecutableDetect;

  /// Version of vfox.
  ///
  /// In en, this message translates to:
  /// **'Version {version}'**
  String settingsVfoxVersion(String version);

  /// Extra environment variables for vfox.
  ///
  /// In en, this message translates to:
  /// **'Environment variables'**
  String get settingsEnvironment;

  /// Help text of the environment variables field.
  ///
  /// In en, this message translates to:
  /// **'Passed to vfox and its plugins, one KEY=VALUE per line. Useful for download mirrors such as VFOX_NODEJS_MIRROR, which a desktop launcher does not read from your shell profile.'**
  String get settingsEnvironmentHelp;

  /// Validation error of the environment variables field.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{Line {lines} is not KEY=VALUE.} other{Lines {lines} are not KEY=VALUE.}}'**
  String settingsEnvironmentInvalid(int count, String lines);

  /// Confirmation after saving a setting.
  ///
  /// In en, this message translates to:
  /// **'Saved'**
  String get settingsSaved;

  /// Heading of the settings stored by vfox.
  ///
  /// In en, this message translates to:
  /// **'vfox configuration'**
  String get settingsVfoxConfig;

  /// Explains where the vfox configuration lives.
  ///
  /// In en, this message translates to:
  /// **'Stored by vfox in ~/.vfox/config.yaml and shared with the command line.'**
  String get settingsVfoxConfigHelp;

  /// Shown when vfox config --list fails.
  ///
  /// In en, this message translates to:
  /// **'Could not read the vfox configuration.'**
  String get settingsVfoxConfigError;

  /// vfox setting proxy.enable.
  ///
  /// In en, this message translates to:
  /// **'Download through a proxy'**
  String get configProxyEnable;

  /// vfox setting proxy.url.
  ///
  /// In en, this message translates to:
  /// **'Proxy URL'**
  String get configProxyUrl;

  /// vfox setting registry.address.
  ///
  /// In en, this message translates to:
  /// **'Plugin registry'**
  String get configRegistry;

  /// Placeholder when registry.address is empty.
  ///
  /// In en, this message translates to:
  /// **'Official registry'**
  String get configRegistryHint;

  /// vfox setting storage.sdkPath.
  ///
  /// In en, this message translates to:
  /// **'SDK storage directory'**
  String get configStorage;

  /// Placeholder when storage.sdkPath is empty.
  ///
  /// In en, this message translates to:
  /// **'Inside ~/.vfox'**
  String get configStorageHint;

  /// vfox setting legacyVersionFile.enable.
  ///
  /// In en, this message translates to:
  /// **'Read legacy version files'**
  String get configLegacyEnable;

  /// Help text of legacyVersionFile.enable.
  ///
  /// In en, this message translates to:
  /// **'Such as .nvmrc, .node-version, .python-version and .sdkmanrc.'**
  String get configLegacyEnableHelp;

  /// vfox setting legacyVersionFile.strategy.
  ///
  /// In en, this message translates to:
  /// **'Version used from legacy files'**
  String get configLegacyStrategy;

  /// legacyVersionFile.strategy = specified.
  ///
  /// In en, this message translates to:
  /// **'As written'**
  String get strategySpecified;

  /// legacyVersionFile.strategy = latest_installed.
  ///
  /// In en, this message translates to:
  /// **'Latest installed'**
  String get strategyLatestInstalled;

  /// legacyVersionFile.strategy = latest_available.
  ///
  /// In en, this message translates to:
  /// **'Latest available'**
  String get strategyLatestAvailable;

  /// vfox setting cache.availableHookDuration.
  ///
  /// In en, this message translates to:
  /// **'Cache version lists for'**
  String get configCache;

  /// Help text of cache.availableHookDuration.
  ///
  /// In en, this message translates to:
  /// **'A duration such as 12h. -1 keeps the lists forever, 0 disables the cache.'**
  String get configCacheHelp;

  /// vfox setting plugin.http.timeout.
  ///
  /// In en, this message translates to:
  /// **'Request timeout'**
  String get configHttpTimeout;

  /// vfox setting plugin.http.downloadTimeout.
  ///
  /// In en, this message translates to:
  /// **'Download timeout'**
  String get configDownloadTimeout;

  /// Help text of the timeout settings.
  ///
  /// In en, this message translates to:
  /// **'A duration such as 30s or 10m. 0s uses the default.'**
  String get configTimeoutHelp;

  /// vfox setting gitignore.enable.
  ///
  /// In en, this message translates to:
  /// **'Add .vfox/ to .gitignore'**
  String get configGitignore;

  /// Help text of gitignore.enable.
  ///
  /// In en, this message translates to:
  /// **'When a version is pinned in a project.'**
  String get configGitignoreHelp;

  /// Heading of vfox settings without a dedicated editor.
  ///
  /// In en, this message translates to:
  /// **'Other settings'**
  String get configOther;

  /// Heading of the about section.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get settingsAbout;

  /// Version of the application.
  ///
  /// In en, this message translates to:
  /// **'Version {version}'**
  String aboutVersion(String version);

  /// License of the application.
  ///
  /// In en, this message translates to:
  /// **'Licensed under the Apache License 2.0.'**
  String get aboutLicense;

  /// Opens the source code repository.
  ///
  /// In en, this message translates to:
  /// **'Source code'**
  String get aboutSource;

  /// Shows the licenses of the dependencies.
  ///
  /// In en, this message translates to:
  /// **'Open-source licenses'**
  String get aboutLicenses;

  /// Opens the vfox documentation.
  ///
  /// In en, this message translates to:
  /// **'vfox documentation'**
  String get aboutDocs;
}

class _AppLocalizationsDelegate extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) => <String>['en', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
