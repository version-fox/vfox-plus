# 设计系统

> [English](../../en/dev/design-system.md)

vfox-plus 遵循 [Material 3 Expressive](https://m3.material.io/)——2025 年 Material Design
演进：弹簧、更丰富的形状库、强调字体与新组件。本页列出应用使用的标记与组件，
及 mygo 实现的位置。

## 颜色

- 配色由**种子颜色**经 Material 颜色工具 *tonal spot* 变体、**2021** 版颜色工具规约，
  为浅色与深色主题生成，与规约一致。
- 种子为用户允许且桌面可给出的**桌面强调色**（XDG 桌面门户
  `org.freedesktop.appearance accent-color`，回退 GTK 主题），否则为设置中所选颜色。
  默认 vfox 徽标橙 `#F28C1B`。
- 大面积用 surface 容器角色：窗口 `surface`，页面板 `surfaceContainerLow`，列表项
  `surfaceContainer`，突出卡片 `surfaceContainerHighest`。主、次、第三容器标记
  头像。
- 危险操作用 `error`。
- Web 界面（`mygo/pair/webapp.html` 的 `--m3-*` 变量）与 Android 应用（`values/colors.xml`、`values-night/colors.xml`）以字面值存放
  方案的 48 个角色，它们与 `mygo/ui/mcu` 按种子算出的角色不同时，`mygo` 里的 `go test ./pair` 会失败。

mygo：`mygo/ui/kit/theme.go` 的 `PaletteFor`，由 `mygo/ui/mcu`（Material 颜色工具的 Go
移植，tonal spot）计算。

## 形状

### 圆角 scale

| 标记 | 半径 | 用途 |
| --- | ---: | --- |
| none | 0 dp | |
| extra small | 4 dp | 分段列表内角、tooltip |
| small | 8 dp | 按下按钮、按钮组内角、chip |
| medium | 12 dp | 文本框、snack bar |
| large | 16 dp | 卡片、菜单、浮动按钮 |
| large increased | 20 dp | 分段列表外角、插件卡片 |
| extra large | 28 dp | 页面板、对话框 |
| extra large increased | 32 dp | |
| extra extra large | 48 dp | |
| full | 50% | 静息按钮、搜索栏、标签 |

mygo：`mygo/ui/kit/draw.go` 的 `Corner*` 常量。

### 形状库与形变

[Material 形状库](https://m3.material.io/styles/shape/overview-principles)的 expressive
形状赋予界面个性：

- **插件头像**按插件名稳定哈希取形（9 边 cookie、4 叶草、sunny、soft burst、
  6 边 cookie、gem、五边形、花、puffy diamond、4 边 cookie、arch、12 叶草），配三组
  颜色之一：primary fixed dim、secondary 与 tertiary 容器。不用 secondary 容器（选中列表项）
  与 primary 容器（vfox 橙等种子下太浅）。悬停旋转 30°，fast spatial 弹簧。
- **空与错误状态**图标置于大形状上。
- **颜色 swatch**选中从圆变成花，微转。
- **加载指示器**在 soft burst、9 边 cookie、五边形、pill、sunny、4 边 cookie、oval 间形变。

形状来自 `mygo/ui/shapes`（`androidx.graphics.shapes` 的 Go 移植）；
`Polygon` 与 `Morph` 把形状或形变变成 `mygo/ui/kit` 绘制的路径。

### 交互形状形变

交互元素随使用改变形状：

- 按钮静息全圆，按下角缩至 8 dp。
- 连接按钮组外角全圆、内角 8 dp；选中按钮全圆，按下按钮内角缩至 4 dp。
- 分段列表项外角 20 dp、内角 4 dp；悬停、按下、选中时内角长至 12 dp。
- 分割按钮菜单打开时尾按钮变圆。

## 字体

应用使用平台无衬线字体（Ubuntu 上 Ubuntu Sans，中文 Noto Sans CJK）与 Material 3
字阶。Material 3 Expressive 为每档加 **emphasized** 重字，用于页面标题、
需注意的列表标题（当前版本）与卡片标题。

| 档 | 常规字重 | Emphasized 字重 |
| --- | ---: | ---: |
| Display、headline、title L | 400 | 500 |
| Title M、title S | 500 | 700 |
| Body L、body M | 400 | 500 |
| Label L、label M | 500 | 700 |

mygo：`mygo/ui/kit/theme.go` 的 `Emph`。

## 文案与密度

三端都尽量少说话。一段文字有用，是因为它帮人理解、判断或操作；布局已经说明的，不再重复。

- **标签不加解释。** 标题和它的控件说明这一行做什么。下面那一行只说控件说不了的：值（`默认`）、
  需要权衡的后果，或之后出错时才会指出的限制。设置是什么、存在哪里、镜像是什么，这些属于文档。
- **格式是空字段里的示例**（`j17`、`17-system`、`KEY=value`），不是它下面的说明。出错时由字段指出错在哪。
- **空列表没有句子。** 它有能填满它的东西：命名别名的按钮，或者一个图标、一个标题和那个操作。
- **页面不用卡片来宣布。** 没有统计下方列表条数的块，也没有通往导航已能到达的页面的快捷入口。
- **没有窗格在等人选择。** 并排时列表已打开第一项，也就是在用的那个；列表之外的添加方式是一个菜单，
  不是每种一个按钮。
- **一节只有标题，没有说明；一页的标题在栏里，没有副标题。**
- **后果只说一次**，在发问的确认框里，不在打开它的那一行里再说。
- **名称在输入的地方解释。** 插件的指南在“按名称安装”对话框里，不在选项里。档案里的须知是没有控件能说的
  几句话，最多三条。
- **图标用来区分行。** 一列行都以同一个图标开头，这个图标什么也没说；设置列表里的每一行各有自己的图标。
- **一行一个操作。** 其他的放进它的菜单：拆分按钮，或溢出按钮。分段控件的各段算一个操作。
- **字号取字号阶梯上的值**（11、12、14、16、22、24、28、32、36、45、57 px）；代码的等宽字体 13 px 按它
  自己的字号算。

### 预算

一个屏幕的普查，网页界面（安卓应用显示的也是它）由 `webheap -audit` 做，桌面应用由
`TestTheScreensAreWithinTheBudgets` 做（[怎么做](contributing.md#审查界面)）。这个测试在十一种窗口里画 21 个屏幕：
1280 dp 宽的浅色和深色，700、412 和 360 dp 宽（应用能打开的最窄窗口），以及系统把文字放大之后的六种：
360 dp 宽的 130 % 和 150 %，412 dp 宽的 150 %，以及 360、700 和 1280 dp 宽的 200 %。任何一个超出预算，测试就失败。
桌面的 1 dp 对应网页界面的 1 px。一个屏幕要满足：

| 普查项 | 预算 |
| ------ | -----: |
| 对比度低于 WCAG AA 的文字（4.5:1；大字 3:1） | 0 |
| 页面比窗口宽 | 否 |
| 小于 24 px 的控件 | 0 |
| 不在字号阶梯上的字号 | 0 |
| 一行里有两个带文字的按钮的列表行 | 0 |
| 三行或更多行以同一图标开头的列表 | 0 |
| 8 个词以上的句子 | 3 |
| 4 个词以上的说明小字 | 3 |
| 被省略号截断的标题和控件文字（一行的，或允许的几行的） | 0 |
| 列表行里画到行内边距之外的控件和文字 | 0 |
| 图标表里没有的图标（仅桌面） | 0 |

**更大的文字。** 系统的文字可以放大，最大到 200 %，每种窗口、每种字号下都要满足预算。装文字的东西跟着文字变宽：
放不下时，原本在文字旁边的控件移到文字下面（列表项末尾的控件、页头和对话框的操作），分段按钮组先让出内边距和图标，
再退成下拉框（标签页、主题、语言、过滤、配对的地址），芯片、标签、选项名，以及下拉框里显示的那个选项，多出第二行。
位置本身不随文字变大的东西最多读到 130 %（底栏的标签）。

**长词。** 一个没有空格可断的词，名字、地址、路径或链接，比它的空间长时，在任意字符处断开（`kit.Breakable`，
列表项对它的字符串自动这样做）：不用省略号截断，不伸出装它的东西，旁边的东西先移到它下面再考虑断它，
所以只有单独一行也放不下的词才会被断。装这种词的盒子要明确告诉它可以比这个词窄（`MinWidth(0)`）：行里的盒子
最窄不能窄过它最长的词，这个限制会沿着每一层装它的盒子往上传。

跟着文字的宽度要么是量出来的（分段、按钮、指纹的每一行、列表项的空间），要么随文字缩放增长（输入框、仓库的卡片），
没有一个是拿某种字体量一次就写死的宽度。普查在最窄的窗口、带导航栏的窗口和最宽的窗口取 200 %，这是三种布局，
并在空间最紧的地方取 130 % 和 150 %。布局并不保证 200 % 下满足的，更小的字号下也满足，所以中间的字号是看出来的，
不是假设出来的：`VFOX_PLUS_CENSUS_SWEEP=1` 时，`TestTheScreensAreWithinTheBudgetsAtEverySizeOfText` 在四种宽度下取 115、130、150
和 175 %，改了布局之后要跑一遍。

**字体。** 布局没有哪一处是照 Ubuntu Sans（图是用它拍的）的宽度写的：宽度跟着字体排出来的文字走，而 200 % 的文字
比 100 % 的宽一倍，比系统里各种字体之间的差别大得多。`TestTheScreensAreWithinTheBudgetsInTheFontsThatTheSystemChooses` 用系统给 `sans-serif`
和 `monospace` 的字体画这些场景（日志里写着跑它的机器上那是哪种字体），取最窄窗口的 100 % 和 200 %，以及最宽的窗口。

## 动效

Material 3 Expressive 用**弹簧**代替时长与缓动曲线。*spatial* 弹簧移动、缩放、重塑，
可超调；*effects* 弹簧变色与透明度，永不超调。各三档：小组件 fast、局部中页 default、
全屏 slow。质量均为 1。

| 弹簧 | Expressive（阻尼比 / 刚度） | Standard |
| --- | ---: | ---: |
| Fast spatial | 0.6 / 800 | 0.9 / 1400 |
| Default spatial | 0.8 / 380 | 0.9 / 700 |
| Slow spatial | 0.8 / 200 | 0.9 / 300 |
| Fast effects | 1 / 3800 | 1 / 3800 |
| Default effects | 1 / 1600 | 1 / 1600 |
| Slow effects | 1 / 800 | 1 / 800 |

vfox-plus 使用 **expressive** 方案：

| 元素 | 弹簧 |
| --- | --- |
| 按钮、按钮组、分割按钮与列表形状 | fast spatial |
| 头像悬停旋转、颜色 swatch 选中 | fast spatial |
| 导航栏展开收起 | default spatial |
| 导航指示器生长、图标填充 | fast spatial / fast effects |
| 对话框缩放进入 | default spatial（缩放）、default effects（淡入） |
| 页面切换（fade through、shared axis） | slow effects |
| 波浪进度、波幅 | default spatial / default effects |
| 加载指示器形变 | 0.6 / 200，每 650 ms 一次，加 4666 ms 旋转 |

弹簧中途变目标保持速度， interrupted 动画平滑。只接受时长曲线的 API（对话框）把
弹簧采样为 settle 时间内的曲线；自带曲线的页面切换持续 slow effects 弹簧 settle
时长。平台要求减少动态效果时弹簧直达目标，不定指示器静止。

Web 界面就是这样的 API：CSS 只有时长和曲线，没有弹簧。它的 `--sp-*` 变量就是六个弹簧：`--sp-NAME-t` 是 settle 所需的毫秒数，
`--sp-NAME` 是阶跃响应采样成的 `linear()` 曲线，取尽量少的采样点，使折线与曲线相差不超过 0.6 %。每个 `transition` 与
`animation` 的时间和曲线都取自它们；对话框在淡入（default effects）的同时缩放进入（default spatial），所以是两个动画。
`mygo/pair` 的一个测试按 `kit.Expressive` 重新计算这些变量，它们不同、或某条声明带着自己的时长或曲线时失败，并打印该粘贴
的行（一直运行到工作结束的波浪进度与加载指示器除外）。没有 `linear()` 的浏览器沿用此前页面的贝塞尔曲线。Android 应用从主题
（`motionSpring*`）取弹簧：会动的东西用 `SpringAnimation`，屏幕之间的转场用 `SpringInterpolator`。

mygo：`mygo/ui/kit/motion.go` 的弹簧与 `AnimateTo`，只在动时绘制，静止时窗口空闲。

### 转场

- 顶级目的地之间：**fade through**。
- 窄窗口 SDK 列表到 SDK：**shared axis** 横向。
- 详情面板 SDK 之间：**fade through**。
- 应用内路由打开的页面（许可证）：**fade forwards**。

### 刷新率

120 Hz 下一帧只有 8.3 ms，各端都要在这个预算内：

- mygo：工具包按显示器的刷新率出帧。窗口由 CPU 绘制时（没有 GPU，如虚拟机和 WSL），
  带缩放的转场每帧要重绘整页的每个像素，耗时 10 到 40 ms，所以页面、对话框和菜单
  在那里只做淡入淡出（`kit.Env.Software`，由 `Painter.Accelerated` 设置）。
- Web 界面与用 WebView 显示它的 Android 应用：动画用 `transform` 和 `opacity`，由合成器
  执行、不占页面主线程；不要动画 `left`、`top`、`height`。长列表设置
  `content-visibility: auto`，打开页面时只布局屏幕上的部分（插件卡片在比桌面慢六倍的
  手机上布局要 58 ms，设置后 16 ms）。
- Android：很多手机让普通窗口停在 60 Hz，所以应用请求当前分辨率下刷新率最高的显示
  模式（`HighRefreshRate.kt`）。

## 组件

| 组件 | 规约 | mygo |
| --- | --- | --- |
| 导航栏（折叠 96 dp、展开 240 dp，带菜单按钮与 FAB） | [Navigation rail](https://m3.material.io/components/navigation-rail) | `ui/kit/nav.go` 的 `NavRail` |
| 导航条（窄窗口） | [Navigation bar](https://m3.material.io/components/navigation-bar) | `ui/kit/nav.go` 的 `NavBar` |
| 连接按钮组（单选） | [Button groups](https://m3.material.io/components/button-groups) | `ui/kit/group.go` 的 `ButtonGroup` |
| 分割按钮 | [Split button](https://m3.material.io/components/split-button) | `ui/kit/button.go` 的 `Button` |
| 图标按钮（expressive 尺寸形状） | [Icon buttons](https://m3.material.io/components/icon-buttons) | `ui/kit/button.go` 的 `IconButton` |
| 常规按钮（按下形状形变） | [Buttons](https://m3.material.io/components/buttons) | `ui/kit/button.go` 的 `Button` |
| 加载指示器（含式与裸式） | [Loading indicator](https://m3.material.io/components/loading-indicator) | `ui/kit/loading.go` 的 `LoadingIndicator` |
| 波浪线性进度（确定与不定） | [Progress indicators](https://m3.material.io/components/progress-indicators) | `ui/kit/loading.go` 的 `WavyProgress` |
| 分段列表 | [Lists](https://m3.material.io/components/lists) | `ui/kit/list.go` 的 `ListGroup`、`ListItem` |
| 筛选 Chip | [Chips](https://m3.material.io/components/chips) | `ui/kit/chip.go` 的 `FilterChip` |
| 搜索栏、卡片、对话框、菜单、snack bar、开关、文本框 | Material 3 | `ui/kit` 的 `SearchBar`、`Card`、`Menu`、对话框与 snack bar，`ui/kit/theme.go` 定制 |

### Android 应用

App 的屏幕由 Material Components 1.14.0 的组件搭成（用 View 而不是 Compose：[原因](packaging-android.md#为什么用-view-而不是-compose)），
其 Expressive 样式是主题的默认值。

| 组件 | 在 App 里 |
| --- | --- |
| 按钮（filled、tonal、text）与图标按钮 | `Widget.Material3Expressive.Button*`，自带形状形变；布局只设尺寸（`materialSizeOverlay`，Medium） |
| 浮动工具栏 | 扫码页的 `FloatingToolbarLayout`，样式 `Widget.Material3.FloatingToolbar.Vibrant`：手电筒与“手动输入” |
| 卡片 | 28 dp 圆角的 filled `MaterialCardView`：命令在 `surfaceContainerHigh`，上次的服务器在 `secondaryContainer` |
| 文本框 | outlined `TextInputLayout`，圆角取规约的 4 dp；框尾图标粘贴与清除 |
| 加载指示器 | 取代“连接”按钮位置的 `LoadingIndicator` |
| 波浪进度指示器 | `Widget.Material3Expressive.LinearProgressIndicator.Wavy`，会话页面加载时 |
| 对话框 | `MaterialAlertDialogBuilder`，Expressive 主题的样式；证书指纹放在 `surfaceContainerHighest` 卡片里 |
| 形状 | 配对屏狐狸背后的 cookie 与错误状态图标背后的 soft burst（`ExpressiveShapes`） |

### 波浪进度指示器

- 轨道与指示器 4 dp 粗、圆端；间距 4 dp。
- 指示器 `primary`，轨道 `secondaryContainer`。
- 波幅 3 dp，每秒走一波长。确定 40 dp 波长，不定 20 dp。
- 确定：10% 以下与 95% 以上波形压平，轨道末端 2 dp 停指示。

### 筛选 Chip

- 高 32 dp，内边距 12 dp，标签 `labelLarge`。关：8 dp 胶囊，1 dp `outlineVariant` 描边，内容
  `onSurfaceVariant`。开：`secondaryContainer` 胶囊，内容 `onSecondaryContainer`，无描边，前面有勾号。
- 状态变化时 Chip 用快速空间弹簧形变：筛选开启时圆角从 8 dp 长到整个高度，描边淡出，填充淡入，18 dp 的勾号
  通过撑宽 Chip 滑入；按下时圆角收到 4 dp。
- 一行 Chip 可换行；同一行的 Chip 互为“或”。Web 界面用 CSS（`.chip`）画同样的 Chip，圆角相同，弹簧相同
  （`--sp-fast-spatial`）。

### 加载指示器

- 默认 48 dp；指示器占 38/48。
- 裸式：`primary`。含式：`primaryContainer` 圆上的 `onPrimaryContainer`。

## 图标

图标为 [Material Symbols](https://fonts.google.com/icons) **Rounded** 风格，字重 400、
grade 0、optical size 24：选中导航目的地图标填充（`FILL` 1），其余描边（`FILL` 0）。
图标与含义见 [`shared/icons.json`](../../../shared/icons.json)，它同时锁定图标的来源：npm 包
[`@iconify-json/material-symbols`](https://www.npmjs.com/package/@iconify-json/material-symbols)
（Apache-2.0，1.3 MB；锁定其版本与 tarball 的 SHA-512）。`vfxtool prepare` 首次把它下载到缓存、
校验后切出所用符号的 SVG（`SYMBOL.svg` 描边、`SYMBOL-fill.svg` 填充），所以仓库里不提交任何图标。
mygo 以文本颜色绘制 SVG（`ui.Icon`），应用因此不带图标字体；导航与按钮的填充动画是描边与填充两个图标的
交叉淡入淡出，SVG（共约 60 KB）在首次绘制某图标时才解析。

Android 应用把同样的符号画成矢量图，仓库里没有任何东西保存或生成它们：它们是
[google/material-design-icons](https://github.com/google/material-design-icons)
为每个符号发布的 Android XML（Apache-2.0，与本仓库相同），由构建导入。那个仓库把它们放在 git 里，不当作库发布，所以
`settings.gradle.kts` 有一个指向其 raw 文件的 Ivy 仓库，只服务组 `symbols`，别的一概不经过它；`app/build.gradle.kts`
列出符号与提交（`10ecc14790612ca8fd164e3f4ffe9c16fc50d5b7`，2026-10-23），
每个符号是一个依赖 `symbols:SYMBOL:COMMIT@xml`，填充的加分类器 `fill1`：对应文件
`symbols/android/SYMBOL/materialsymbolsrounded/SYMBOL_24px.xml` 与 `SYMBOL_fill1_24px.xml`。Gradle 把它们取到自己的缓存里
一次，之后离线构建也能用；每个变体一个任务（`copyMaterialSymbolsRelease`、`copyMaterialSymbolsDebug`）把它们原样写成
`ic_NAME.xml`，作为生成的资源交给 Android 插件与其它资源合并（`app/build/generated/res/copyMaterialSymbolsRelease/drawable`）。文件的颜色取主题的 `colorControlNormal`，除非视图另外着色；在从右到左的
语言里要翻转的符号由 Google 标出（`autoMirrored`，这里是 `copy`、`forward` 与 `help`）。应用用到 `ic_close`（`close`）、`ic_copy`（`content_copy`）、`ic_forward`（`arrow_forward`）、`ic_help`（`help`）、`ic_history`（`history`）、`ic_keyboard`（`keyboard`）、`ic_link`（`link`）、`ic_lock`（`lock`）、`ic_paste`（`content_paste`）、`ic_retry`（`replay`）、`ic_scan_qr`（`qr_code_scanner`）、`ic_torch_off`（`flashlight_off`）、`ic_torch_on_fill`（`flashlight_on`，填充）与 `ic_unreachable`（`cloud_off`）。
它们与上面的 npm 包是同一套图标的两份拷贝，即 Material Symbols Rounded、字重 400、24 px，所以各界面看起来一样。

### 应用图标

应用图标是 vfox 标志里的狐狸，用 Material 3 Expressive 的形状语言重新绘制
（[`shared/packaging/linux/dev.vfox.VfoxPlus.svg`](../../../shared/packaging/linux/dev.vfox.VfoxPlus.svg)）。

- **形象。** 与 version-fox 组织头像是同一只狐狸：橙色的脸，奶油色脸颊围成一个
  "V"，奶油色内耳，深色鼻子和带高光的圆形深色眼睛。没有描边，转角像形状库的
  形状一样用大圆角。两只耳朵与两侧脸颊的毛簇构成的轮廓，在 16 px 下仍是狐狸。
- **底板与光晕。** 靛蓝底板（皮毛的对比色），狐狸身后是形状库的 9 边 cookie，
  作柔和的光晕。
- **颜色。** 皮毛自上而下 `#FFB63F` 至 `#FF8419`，脸颊 `#FFFFFF` 至 `#FFE7CB`，
  内耳 `#FFE3C4`，眼睛和鼻子 `#17123F`；底板自左上至右下 `#5444E4` 至
  `#1B1268`；光晕为白色，不透明度 12% 至 3%。
- **易辨认。** 一个大而简单的形状，没有文字。在成品图上实测，皮毛与身后背景的
  对比度不低于 3.2:1（WCAG 对图形要求 3:1），脸颊不低于 5.9:1，眼睛与皮毛不低于
  7.1:1。桌面上狐狸占底板三分之二；安卓上它位于自适应图标 66 dp 的安全区内，
  圆形、方圆与圆角方形遮罩都能完整显示。

一幅图服务所有平台：

| 平台 | 文件 | 说明 |
| --- | --- | --- |
| Linux | [`shared/packaging/linux/dev.vfox.VfoxPlus.svg`](../../../shared/packaging/linux/dev.vfox.VfoxPlus.svg) | 母版，方圆底板。`vfxtool build-deb` 装为可缩放图标。旁边的 AppStream 元数据写明软件中心垫在图标后面的颜色：浅色主题 `#FFDCC1`，深色主题 `#6B3B04`，即 vfox 橙配色的主容器色（Web 界面的 `--m3-primary-container`）。 |
| 安卓启动器 | [`android/app/src/main/res/drawable`](../../../android/app/src/main/res/drawable) 里的 `ic_launcher_background.xml`、`ic_launcher_foreground.xml` 与 `ic_launcher_monochrome.xml` | 自适应图标：启动器给铺满的底板和狐狸加遮罩。单色层供安卓 13 的主题图标，脸颊半强度，眼睛和鼻子镂空。 |
| 安卓配对界面 | 同目录的 `ic_mark.xml` | 母版的矢量绘制版。 |
| Windows | `.msix` 里的 `Assets/*.png` | 不提交任何图片，故由 `vfxtool build-msix` 用同样的路径（[`shared/tooling/internal/appicon`](../../../shared/tooling/internal/appicon)）绘制。 |
| Web 界面 | [`mygo/pair/favicon.svg`](../../../mygo/pair/favicon.svg) | 底板与狐狸，去掉光晕，狐狸更大，16 px 下仍可辨认。配对服务的每个页面都把它以 data URI 内联（`favicon.go`），页面不为图标另发请求；Web 界面用它作导航栏的品牌标志，Web 应用清单把它写成已安装应用的图标（尺寸为 `any` 的 SVG 图标是 Chromium 接受用于安装的图标）。 |

SVG 是母版：矢量绘制、`appicon` 与 favicon 重复它的路径，不一致时 `shared/tooling`
（前两者）与 `mygo/pair`（favicon）的 `go test` 失败。
