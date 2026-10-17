// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appTitle => 'vfox+';

  @override
  String get actionRefresh => '刷新';

  @override
  String get actionCancel => '取消';

  @override
  String get actionRetry => '重试';

  @override
  String get actionClose => '关闭';

  @override
  String get actionSave => '保存';

  @override
  String get actionReset => '恢复默认值';

  @override
  String get actionBrowse => '浏览…';

  @override
  String get actionCopy => '复制';

  @override
  String get actionCopied => '已复制到剪贴板';

  @override
  String get actionOpenHomepage => '打开主页';

  @override
  String get actionShowDetails => '详情';

  @override
  String get loading => '正在加载…';

  @override
  String get errorTitle => '出错了';

  @override
  String get navHome => '首页';

  @override
  String get navSdks => 'SDK';

  @override
  String get navPlugins => '插件';

  @override
  String get navTasks => '任务';

  @override
  String get navSettings => '设置';

  @override
  String get navExpand => '显示标签';

  @override
  String get navCollapse => '隐藏标签';

  @override
  String get navAddPlugin => '添加插件';

  @override
  String get vfoxCheckingTitle => '正在查找 vfox…';

  @override
  String get vfoxMissingTitle => '未安装 vfox';

  @override
  String get vfoxMissingBody => 'vfox+ 通过 vfox 命令行工具管理 SDK。请先安装 vfox，或告诉 vfox+ 它在哪里。';

  @override
  String get vfoxBrokenTitle => 'vfox 没有响应';

  @override
  String vfoxBrokenBody(String path) {
    return '可执行文件 $path 未能报告 vfox 版本。';
  }

  @override
  String get vfoxInstallGuide => '安装指南';

  @override
  String get vfoxLocate => '指定 vfox…';

  @override
  String homeSubtitle(String version) {
    return 'vfox $version';
  }

  @override
  String homeStatPlugins(int count) {
    String _temp0 = intl.Intl.pluralLogic(count, locale: localeName, other: '插件');
    return '$_temp0';
  }

  @override
  String homeStatVersions(int count) {
    String _temp0 = intl.Intl.pluralLogic(count, locale: localeName, other: '已安装版本');
    return '$_temp0';
  }

  @override
  String get homeStatActive => '使用中';

  @override
  String get homeActiveTitle => '正在使用的版本';

  @override
  String get homeActiveEmpty => '还没有正在使用的 SDK 版本。请在 SDK 页面中选择一个。';

  @override
  String get homeShellTitle => '在终端中使用这些版本';

  @override
  String get homeShellBody => '将下面这行添加到 ~/.bashrc，新的 Shell 就会使用这里选择的版本。其他 Shell 的配置方法请参阅 vfox 文档。';

  @override
  String get homeExploreTitle => '添加更多 SDK';

  @override
  String get homeExploreBody => '通过插件为 vfox 添加 Node.js、Java、Python、Go 等众多 SDK。';

  @override
  String get homeDocsTitle => 'vfox 文档';

  @override
  String get homeDocsBody => '了解作用域、插件和配置文件。';

  @override
  String get sdksSearchHint => '搜索已安装的插件';

  @override
  String get sdksEmptyTitle => '还没有插件';

  @override
  String get sdksEmptyBody => '插件告诉 vfox 如何安装 SDK。添加一个插件即可开始。';

  @override
  String sdksNoMatch(String query) {
    return '没有与“$query”匹配的插件。';
  }

  @override
  String sdksCurrent(String version) {
    return '使用中：$version';
  }

  @override
  String get sdksNoCurrent => '未使用任何版本';

  @override
  String sdksVersionCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(count, locale: localeName, other: '已安装 $count 个版本', zero: '未安装任何版本');
    return '$_temp0';
  }

  @override
  String get sdksSelectPrompt => '选择一个 SDK 以管理其版本。';

  @override
  String get sdksLoadError => '无法列出已安装的 SDK。';

  @override
  String sdkPluginVersion(String version) {
    return '插件 $version';
  }

  @override
  String get sdkTabInstalled => '已安装';

  @override
  String get sdkTabAvailable => '可安装';

  @override
  String get sdkUpdatePlugin => '更新插件';

  @override
  String get sdkRemovePlugin => '移除插件';

  @override
  String get sdkMoreActions => '更多操作';

  @override
  String sdkRemoveTitle(String name) {
    return '移除 $name？';
  }

  @override
  String sdkRemoveBody(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '将删除该插件及其已安装的 $count 个版本。',
      zero: '将删除该插件。',
    );
    return '$_temp0';
  }

  @override
  String get sdkRemoveConfirm => '移除';

  @override
  String sdkInstalledEmpty(String name) {
    return '尚未安装任何版本的 $name。';
  }

  @override
  String get sdkBrowseVersions => '浏览版本';

  @override
  String get sdkInfoError => '无法读取插件信息。';

  @override
  String get versionInUse => '使用中';

  @override
  String get versionUse => '使用';

  @override
  String get versionUseOptions => '更多使用方式';

  @override
  String get versionUseGlobally => '设为全局默认';

  @override
  String get versionUseInProject => '在项目中使用…';

  @override
  String get versionUnuse => '停止使用';

  @override
  String get versionCopyPath => '复制安装路径';

  @override
  String get versionUninstall => '卸载';

  @override
  String versionUninstallTitle(String name, String version) {
    return '卸载 $name $version？';
  }

  @override
  String get versionUninstallBody => '该版本的文件将被删除。';

  @override
  String get versionUninstallCurrentBody => '该版本正在使用中。删除后，vfox 会切换到其他已安装的版本（如有）。';

  @override
  String get versionInstall => '安装';

  @override
  String get versionInstalled => '已安装';

  @override
  String versionMoreActions(String version) {
    return '$version 的更多操作';
  }

  @override
  String get availableSearchHint => '筛选版本';

  @override
  String get availableFilterAll => '全部';

  @override
  String get availableFilterTagged => '带标签';

  @override
  String get availableFilterInstalled => '已安装';

  @override
  String get availableLoading => '正在获取可安装的版本…';

  @override
  String get availableEmpty => '没有匹配的版本。';

  @override
  String availableError(String name) {
    return '无法列出 $name 的版本。';
  }

  @override
  String availableCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(count, locale: localeName, other: '$count 个版本');
    return '$_temp0';
  }

  @override
  String useProjectTitle(String name, String version) {
    return '在项目中使用 $name $version';
  }

  @override
  String get useProjectBody => 'vfox 会将版本记录到项目目录的 .vfox.toml 中，并将 SDK 链接到 .vfox/sdks。';

  @override
  String get useProjectDirectory => '项目目录';

  @override
  String get useProjectInvalid => '请选择一个已存在的目录。';

  @override
  String get useProjectConfirm => '在项目中使用';

  @override
  String get pluginsTitle => '插件';

  @override
  String get pluginsSubtitle => '每个插件都为 vfox 添加一种语言或工具。';

  @override
  String get pluginsSearchHint => '搜索插件';

  @override
  String get pluginsFilterAll => '全部';

  @override
  String get pluginsFilterOfficial => '官方';

  @override
  String get pluginsFilterCommunity => '社区';

  @override
  String get pluginsOfficial => '官方插件';

  @override
  String get pluginsAdd => '添加';

  @override
  String get pluginsAdded => '已添加';

  @override
  String get pluginsAddCustom => '从来源添加…';

  @override
  String get pluginsUpdateAll => '更新全部插件';

  @override
  String get pluginsLoadError => '无法加载插件仓库。';

  @override
  String get pluginsNoMatch => '没有与搜索匹配的插件。';

  @override
  String get pluginsCustomTitle => '从来源添加插件';

  @override
  String get pluginsCustomSource => '来源';

  @override
  String get pluginsCustomSourceHelp => '插件压缩包（.zip）或脚本（.lua）的 URL 或路径。';

  @override
  String get pluginsCustomFileType => 'vfox 插件';

  @override
  String get pluginsCustomAlias => '名称（可选）';

  @override
  String get pluginsCustomAliasHelp => '以此名称而非插件自身的名称安装插件。';

  @override
  String get pluginsCustomInvalidName => '只能使用字母、数字、点、连字符和下划线。';

  @override
  String get pluginsCustomConfirm => '添加插件';

  @override
  String get pluginsCustomRequired => '请输入 URL 或路径。';

  @override
  String get tasksEmptyTitle => '没有正在进行的任务';

  @override
  String get tasksEmptyBody => '安装和其他更改在 vfox 处理期间会显示在这里。';

  @override
  String get tasksClearFinished => '清除已完成';

  @override
  String get taskStatusQueued => '等待中';

  @override
  String get taskStatusRunning => '进行中';

  @override
  String get taskStatusSucceeded => '已完成';

  @override
  String get taskStatusFailed => '失败';

  @override
  String get taskStatusCancelled => '已取消';

  @override
  String taskDownloading(int percent) {
    return '正在下载… $percent%';
  }

  @override
  String get taskShowOutput => '显示输出';

  @override
  String get taskHideOutput => '隐藏输出';

  @override
  String get taskCopyOutput => '复制输出';

  @override
  String get taskNoOutput => '暂无输出。';

  @override
  String taskInstall(String subject) {
    return '安装 $subject';
  }

  @override
  String taskUninstall(String subject) {
    return '卸载 $subject';
  }

  @override
  String taskUse(String subject) {
    return '使用 $subject';
  }

  @override
  String taskUnuse(String subject) {
    return '停止使用 $subject';
  }

  @override
  String taskAddPlugin(String subject) {
    return '添加插件 $subject';
  }

  @override
  String get taskAddPluginFromSource => '从来源添加插件';

  @override
  String taskRemovePlugin(String subject) {
    return '移除插件 $subject';
  }

  @override
  String taskUpdatePlugin(String subject) {
    return '更新插件 $subject';
  }

  @override
  String get taskUpdateAllPlugins => '更新全部插件';

  @override
  String taskSucceeded(String title) {
    return '$title：已完成';
  }

  @override
  String taskFailed(String title) {
    return '$title：失败';
  }

  @override
  String get taskView => '查看';

  @override
  String get settingsAppearance => '外观';

  @override
  String get settingsTheme => '主题';

  @override
  String get themeSystem => '跟随系统';

  @override
  String get themeLight => '浅色';

  @override
  String get themeDark => '深色';

  @override
  String get settingsColor => '颜色';

  @override
  String get settingsSystemColor => '使用桌面强调色';

  @override
  String get settingsSystemColorUnavailable => '桌面未提供强调色，因此使用下方的颜色。';

  @override
  String settingsColorOption(int index) {
    return '强调色 $index';
  }

  @override
  String get settingsLanguage => '语言';

  @override
  String get languageSystem => '跟随系统';

  @override
  String get settingsVfox => 'vfox';

  @override
  String get settingsExecutable => '可执行文件';

  @override
  String get settingsExecutableDetected => '自动检测';

  @override
  String get settingsExecutableCustom => '手动指定';

  @override
  String get settingsExecutableMissing => '未找到';

  @override
  String get settingsExecutableDetect => '自动检测';

  @override
  String settingsVfoxVersion(String version) {
    return '版本 $version';
  }

  @override
  String get settingsEnvironment => '环境变量';

  @override
  String get settingsEnvironmentHelp =>
      '传递给 vfox 及其插件，每行一个 KEY=VALUE。适用于 VFOX_NODEJS_MIRROR 等下载镜像设置——从桌面启动器启动时，不会读取 Shell 配置文件中的这些变量。';

  @override
  String settingsEnvironmentInvalid(int count, String lines) {
    String _temp0 = intl.Intl.pluralLogic(count, locale: localeName, other: '第 $lines 行不是 KEY=VALUE 格式。');
    return '$_temp0';
  }

  @override
  String get settingsSaved => '已保存';

  @override
  String get settingsVfoxConfig => 'vfox 配置';

  @override
  String get settingsVfoxConfigHelp => '由 vfox 保存在 ~/.vfox/config.yaml 中，与命令行共享。';

  @override
  String get settingsVfoxConfigError => '无法读取 vfox 配置。';

  @override
  String get configProxyEnable => '通过代理下载';

  @override
  String get configProxyUrl => '代理 URL';

  @override
  String get configRegistry => '插件仓库';

  @override
  String get configRegistryHint => '官方仓库';

  @override
  String get configStorage => 'SDK 存储目录';

  @override
  String get configStorageHint => '位于 ~/.vfox 中';

  @override
  String get configLegacyEnable => '读取旧版版本文件';

  @override
  String get configLegacyEnableHelp => '例如 .nvmrc、.node-version、.python-version 和 .sdkmanrc。';

  @override
  String get configLegacyStrategy => '旧版版本文件中的版本';

  @override
  String get strategySpecified => '按文件所写';

  @override
  String get strategyLatestInstalled => '最新已安装';

  @override
  String get strategyLatestAvailable => '最新可用';

  @override
  String get configCache => '版本列表缓存时长';

  @override
  String get configCacheHelp => '一段时长，例如 12h。-1 表示永久缓存，0 表示禁用缓存。';

  @override
  String get configHttpTimeout => '请求超时';

  @override
  String get configDownloadTimeout => '下载超时';

  @override
  String get configTimeoutHelp => '一段时长，例如 30s 或 10m。0s 表示使用默认值。';

  @override
  String get configGitignore => '将 .vfox/ 添加到 .gitignore';

  @override
  String get configGitignoreHelp => '在项目中固定版本时生效。';

  @override
  String get configOther => '其他设置';

  @override
  String get settingsAbout => '关于';

  @override
  String aboutVersion(String version) {
    return '版本 $version';
  }

  @override
  String get aboutLicense => '基于 Apache License 2.0 授权。';

  @override
  String get aboutSource => '源代码';

  @override
  String get aboutLicenses => '开源许可';

  @override
  String get aboutDocs => 'vfox 文档';
}
