# 界面规约

> [English](../../en/dev/ui-spec.md)

本文描述 vfox-plus 展示什么、做什么。
视觉细节——颜色、形状、字体与动效——见
[设计系统](design-system.md)；文本见
[`shared/i18n`](../../../shared/i18n)，下文以键引用，如 `navHome`。图标以
[`shared/icons.json`](../../../shared/icons.json) 中的名字引用，如 `sdks`。

## 窗口与导航

应用有 5 个顶级目的地：

| 目的地 | 标签键 | 图标 | 用途 |
| --- | --- | --- | --- |
| Home | `navHome` | `home` | 总览与快捷入口。 |
| SDKs | `navSdks` | `sdks` | 已安装插件及其版本。 |
| Plugins | `navPlugins` | `plugins` | 插件仓库。 |
| Tasks | `navTasks` | `tasks` | 运行中与已完成的操作。徽标显示排队与运行数。 |
| Settings | `navSettings` | `settings` | 外观、语言、vfox 可执行文件、vfox 配置、关于。 |

布局按窗口宽度的 Material 3 窗口尺寸类：

- **Compact**（< 600 dp）：底部导航条。
- **Medium 及以上**（≥ 600 dp）：起始边导航栏。菜单按钮在折叠栏（图标下标签）与
  展开栏（图标旁标签）间切换；选择记住（`railExpanded`）。栏上有 FAB
  `navAddPlugin`，打开 Plugins。

页面置于圆角面板，距窗口边缘 16 dp。目的地切换用 fade through。选中目的地图标填充，
其余描边。

最小窗口 360×480 dp。默认 1280×820 dp。

## 通用状态

每个加载数据的视图有 4 种状态：

- **加载中**：居中加载指示器，等待可能较长时配短文本（`availableLoading`）。
- **错误**：soft-burst 形上的错误图标、失败标题（如 `sdksLoadError`）、vfox 错误信息与
  重试按钮（`actionRetry`）。
- **空**：expressive 形上的图标、标题、说明与解决动作（如 `sdksEmptyTitle` 配
  Plugins 按钮）。
- **内容**。重载时旧内容保留。

长耗时操作永不阻塞界面：都成为任务（见[任务](#任务)）。

## vfox 不可用

定位 vfox 时各页（设置除外）显示加载指示器与 `vfoxCheckingTitle`。找不到可执行文件
显示 `vfoxMissingTitle` / `vfoxMissingBody`，三个动作：打开安装指南
（<https://vfox.dev/guides/quick-start.html>）、选择可执行文件（`vfoxLocate`）、重试。
可执行文件报不出版本显示 `vfoxBrokenTitle` / `vfoxBrokenBody`，动作相同。
设置页始终可达。

## Home

- 页眉：应用名、`vfox <版本> · <可执行文件路径>` 与刷新按钮。
- 三个统计块：插件数、已安装版本数、有使用中版本的插件数。每块自有容器色
  （primary、secondary、tertiary）与半藏边后的大装饰形。点统计块打开 SDKs。
- `homeActiveTitle`：有使用中版本的插件分段列表，头像加版本。点项打开 SDKs 页对应项。
  无则显示 `homeActiveEmpty`。
- 终端使用说明卡（`homeShellTitle`、`homeShellBody`），含
  `eval "$(vfox activate bash)"` 行与复制按钮。
- 两个快捷卡：`homeExploreTitle`（打开 Plugins）、`homeDocsTitle`
  （打开 <https://vfox.dev/>）。

## SDKs

列表-详情布局。页宽 ≥ 840 dp 时列表（360 dp）与详情并排；详情无选中显示
`sdksSelectPrompt`，切换选中 fade through。窄页列表与详情沿横向 shared axis 互换，
详情有返回按钮。

### 列表

- 页眉刷新按钮与搜索栏（`sdksSearchHint`），按名过滤。
- 已安装插件分段列表：头像、名称、当前版本（`sdksCurrent` 或 `sdksNoCurrent`）与已安装
  版本数（`sdksVersionCount`）。选中项高亮。
- 空：`sdksEmptyTitle` / `sdksEmptyBody` 配 `navAddPlugin`。

### 详情

- 页眉：72 dp 头像、名称、当前版本标签与插件版本标签（`sdkPluginVersion`）、插件描述。
  动作：更新插件（`sdkUpdatePlugin`，运行时变加载指示器）与菜单
  `actionOpenHomepage`、`sdkRemovePlugin`。窄面板动作移至头像上行。
- 连接按钮组在 `sdkTabInstalled`（带计数）与 `sdkTabAvailable` 间切换；有
  [配置档案](architecture.md#插件配置档案)的插件还有 `sdkTabOptions`（图标 `options`）。页签淡入淡出切换。

**已安装**列出已安装版本，新版在前：

- 当前版本有填充 `current` 图标、强调标签与 `versionInUse` 标记；菜单提供
  `versionUseInProject`、`versionUnuse`、`versionCopyPath` 与 `versionUninstall`。
- 其它版本为分割按钮：`versionUse` 设全局默认；菜单提供 `versionUseGlobally` 与
  `versionUseInProject`。溢出菜单提供 `versionCopyPath` 与 `versionUninstall`。
- 版本有任务运行时动作变加载指示器。
- 空：`sdkInstalledEmpty` 配 `sdkBrowseVersions`。
- 有配置档案的插件，版本名称会被拆开，标题下第二行显示发行版（`Temurin`、`OpenJDK`）与
  类型（`JavaFX`、`x64`、`OpenHarmony`）。

**可用**首次打开跑 `vfox search`。有配置档案的插件，第一行放：

- **搜索按钮** `availableChoiceButton`（“发行版：Temurin”；选择项的标签来自档案，图标 `registry`）。
  档案里有这样的搜索时（`java`），它会打开[选择对话框](#选择与名称)来决定搜索列出什么；所选值成为
  `vfox search` 的参数，按插件保存在设置里（`searches`），所以重启后仍在，Web 界面里也一样，之前选择的列表立即消失；
- `availableInstallByName`，描边按钮，打开[按名称安装对话框](#选择与名称)。每个插件都有。

列表加载中或失败时这一行仍在，方便改选。随后是：

- 过滤框（`availableSearchHint`，匹配版本与备注）、连接按钮组
  `availableFilterAll` / `availableFilterTagged`（有备注如 LTS）/
  `availableFilterInstalled`，与版本数（`availableCount`）。
- **筛选 Chip**：档案里每个维度只要有一个已列出的版本具有其值，就占一行，行首是维度标签
  （`Support`：LTS、JavaFX；`Channel`：stable、beta、dev、OpenHarmony；`Architecture`：x64、arm64），
  有 Chip 开启时显示 `availableClearFilters`。Chip 是 Material 3 的筛选 Chip
  （见[设计系统](design-system.md#组件)）：同一维度内的值是“或”，维度之间、过滤框与按钮组互相收窄。
- 懒加载分段列表。每版备注成标签，附带工具（`npm 11.19.1`）；搜索列出多个发行版时，还有一个
  发行版标签。尾动作为 `versionInstall`；已安装为禁用 `versionInstalled`；安装中为波浪进度配
  `taskDownloading` 与取消按钮。

**选项**按插件的档案用分段列表构建：

- 该插件版本名称的说明（`optionsGuideTitle`）；
- `optionsVariables`：插件读取的每个环境变量一项（Java 与 Flutter 的 GitHub 镜像、Node.js 的下载镜像、
  Flutter 的存储镜像、Node.js 的 C 库，后者带 `optionsVariableAdvanced` 标签），显示变量名、值（若是
  已检查过的镜像则带镜像名）或 `optionsVariableUnset`，以及 `optionsVariableChange`，打开
  [变量对话框](#选择与名称)；
- `optionsLegacy`：插件让 vfox 读取的版本文件，以标签显示（`.sdkmanrc`；`.node-version` 与 `.nvmrc`），
  没有时为 `optionsLegacyNone`；vfox 配置里的开关 `configLegacyEnable`；策略有影响时（Node.js）
  显示 `configLegacyStrategy`，没有影响时（Java）显示 `optionsLegacyStrategyIgnored`；
- `optionsShorthands`：插件除列出的版本外接受的名称，每个带含义与 `optionsShorthandTry`，后者用该名称
  打开按名称安装对话框；
- `optionsTips`，档案里的须知；
- `optionsDocs`：用应用的语言打开该插件文档的链接。

### 确认与对话框

- 移除插件：`sdkRemoveTitle` / `sdkRemoveBody`（含将删除的版本数），危险确认
  `sdkRemoveConfirm`。
- 卸载：`versionUninstallTitle` 配 `versionUninstallBody`，当前版本用
  `versionUninstallCurrentBody`。
- 项目中使用：`useProjectTitle` / `useProjectBody`，目录框配浏览按钮；目录必须存在
  （`useProjectInvalid`）。

对话框按弹簧缩放进入。

#### 选择与名称

- **选择**（`searchChoose…`）：标题为选择项的标签（`Distribution`）的对话框，含过滤框
  （`searchChooseHint`，匹配简称、全称、名称与厂商，两种语言都可）和按档案顺序排列的选择列表，
  常用的在前；每项显示名称、厂商与简称，当前项带对勾，vfox 不带参数时列出的那一项带 `searchChoiceDefault`。
  选中即关闭对话框并列出该项。
- **按名称安装**（`installName…`）：字段 `installNameField`，接受 `vfox install` 中 `<插件>@` 后面的部分
  （辅助文字 `installNameHelp`）；有档案的插件还有 `installNameExamples`，每个简写一个筛选 Chip，点击
  填入字段，并显示当前所输简写的含义。空名称（`installNameRequired`）、含空格或以短横线开头的名称
  （`installNameInvalid`）会被拒绝。确认后排队 `vfox install --yes <插件>@<名称>` 并切到任务页。
- **变量**（`variable…`）：变量的说明与名称；地址类变量有等宽字段 `variableAddress`（地址必须以 `http://`
  或 `https://` 开头，`variableInvalidAddress`）；`variablePresets` 下每个已检查的值一个 Chip，再加一个
  取消变量的 Chip `variableDefaultOption`。选择类变量（`musl`、`glibc`）只能取这些值
  （`variableInvalidChoice`）。值会写入设置里的额外环境变量，设置页也会显示。

## Plugins

- 页眉：`pluginsTitle`、`pluginsSubtitle`；动作：更新全部插件、刷新、
  `pluginsAddCustom`。
- 搜索栏（`pluginsSearchHint`，匹配名称与主页）与连接按钮组
  `pluginsFilterAll` / `pluginsFilterOfficial` / `pluginsFilterCommunity`。
- 响应式网格卡：头像、名称、官方 `verified` 图标（tooltip `pluginsOfficial`）、仓库路径、
  打开主页按钮、`pluginsAdd`——已安装显示 `pluginsAdded`，点开 SDKs 页对应插件。
- `pluginsAddCustom` 要来源（URL 或 `.zip` / `.lua` 文件路径，文件选择器限类型，
  标签 `pluginsCustomFileType`）与可选别名。vfox 会搬走本地 `.lua` 文件，故应用传
  临时拷贝（见 [CLI 契约](vfox-cli-contract.md#已知问题)）。

## 任务

任务分段列表，新在前，页眉 `tasksClearFinished`。每任务展示：

- 状态：加载指示器（运行中）、`queued`、`succeeded`、`error` 或 `cancel`（已取消）；
- 标题（`taskInstall`、`taskUse` 等）与状态文本；失败显示 vfox 错误信息；
- 安装中显示波浪进度；
- 取消（排队或运行中）、重试（失败或取消）与展开输出开关：命令行与等宽可选输出
  块，配复制按钮。已完成任务只保留最新 500 行，更早的只计数不显示
  （`taskDroppedLines`）。

其它页时任务完成弹 snack bar（`taskSucceeded`）。失败（`taskFailed` 加 vfox 消息）
常驻至手动关闭，提供 `taskView` 打开 Tasks。

## 设置

分段列表分组：

- **外观**：主题（`themeSystem` / `themeLight` / `themeDark` 连接按钮组）、
  `settingsSystemColor`（开关）、8 个颜色 swatch（选其一即关桌面色；选中 swatch 变成
  花形）与语言（`languageSystem` / `English` / `简体中文`；语言名永远用本语言书写）。
- **vfox**：可执行文件及其来源（`settingsExecutableDetected`、
  `settingsExecutableCustom` 或 `settingsExecutableMissing`）与版本、浏览按钮，自定义
  可执行文件配 `settingsExecutableDetect`；额外环境变量 `KEY=VALUE` 行，校验
  （`settingsEnvironmentInvalid`）。
- **vfox 配置**：[CLI 契约](vfox-cli-contract.md#配置)每键一编辑器——布尔开关、
  `legacyVersionFile.strategy` 菜单、其余文本框（回车或失焦保存；清空重置），未知键
  `configOther`。
- **关于**：应用名、版本与许可证，源码与 vfox 文档链接，依赖开源许可证含打包的
  Material Symbols 图标（Apache-2.0，Google）。

## 无障碍

- 纯图标按钮都有 tooltip，同时作为无障碍名。
- 选中项、开关、按钮组暴露选中状态。
- 标题标记为标题。
- 平台要求减少动态效果时全部动效瞬时或静止。
- 文本跟平台字号；布局换行不裁剪。
