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
  `surfaceContainer`，突出卡片 `surfaceContainerHighest`。主、次、第三容器标记统计块
  与头像。
- 危险操作用 `error`。

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
| extra large | 28 dp | 页面板、对话框、统计块 |
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
- **统计块**大形状半藏边后。
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
字阶。Material 3 Expressive 为每档加 **emphasized** 重字，用于页面标题、统计块数字、
需注意的列表标题（当前版本）与卡片标题。

| 档 | 常规字重 | Emphasized 字重 |
| --- | ---: | ---: |
| Display、headline、title L | 400 | 500 |
| Title M、title S | 500 | 700 |
| Body L、body M | 400 | 500 |
| Label L、label M | 500 | 700 |

mygo：`mygo/ui/kit/theme.go` 的 `Emph`。

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

mygo：`mygo/ui/kit/motion.go` 的弹簧与 `AnimateTo`，只在动时绘制，静止时窗口空闲。

### 转场

- 顶级目的地之间：**fade through**。
- 窄窗口 SDK 列表到 SDK：**shared axis** 横向。
- 详情面板 SDK 之间：**fade through**。
- 应用内路由打开的页面（许可证）：**fade forwards**。

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
| 搜索栏、卡片、对话框、菜单、snack bar、开关、文本框 | Material 3 | `ui/kit` 的 `SearchBar`、`Card`、`Menu`、对话框与 snack bar，`ui/kit/theme.go` 定制 |

### 波浪进度指示器

- 轨道与指示器 4 dp 粗、圆端；间距 4 dp。
- 指示器 `primary`，轨道 `secondaryContainer`。
- 波幅 3 dp，每秒走一波长。确定 40 dp 波长，不定 20 dp。
- 确定：10% 以下与 95% 以上波形压平，轨道末端 2 dp 停指示。

### 加载指示器

- 默认 48 dp；指示器占 38/48。
- 裸式：`primary`。含式：`primaryContainer` 圆上的 `onPrimaryContainer`。

## 图标

图标为 [Material Symbols](https://fonts.google.com/icons) **Rounded** 风格可变字体，
沿坐标轴做动画：选中导航目的地图标填充（`FILL` 1），其余描边（`FILL` 0）。
图标与含义见 [`shared/icons.json`](../../../shared/icons.json)。应用只打包所用字形：
构建时 `vfxtool iconfont`（`shared/tooling/internal/iconfont`）从锁定版本字体按
ligature 裁剪；锁定字重、grade 与 optical size，只留 fill 轴，得 9 KB WOFF2 字体。
mygo 不能设字体 fill 轴，故同时打包描边与填充（改家族名，`vfxtool renamefont`
改名）两套，交叉淡入淡出（共 36 KB）。

应用图标为 vfox 橙 12 边 cookie 形加白 "v+"（
[`shared/packaging/linux/dev.vfox.VfoxPlus.svg`](../../../shared/packaging/linux/dev.vfox.VfoxPlus.svg)）。
