# 内存占用

> [English](../../en/dev/memory.md)

低内存占用是项目目标（见[架构](architecture.md#内存)）。
本页记录应用在 Linux 上的占用，同一方法测量，便于发现回归。数字是快照：行是沙箱
`measure` 任务单次运行的结果，同样数字见
[mygo README](implementation-mygo.md#内存)。

## 结果

单位 MB，应用及其全部子进程之和，Docker 内 Ubuntu 26.04，Xvfb（1280×800，无 GPU）。
"Versions" 是列出 Node.js 872 个可用版本的 SDK 页。

| 实现 | 技术 | 进程 | Home PSS | Home RSS | Versions PSS | Versions RSS | 列表增加 (PSS) |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| [mygo](../../../mygo/README.zh-CN.md) | mygo 0.7.3、Go 1.27.3、GTK 3（purego）、CPU 渲染 | 1 | 51 | 63 | 59 | 73 | +8 |

mygo（Home 页 PSS 51 MB，单进程）用处理器从 18 MB 的 Go 可执行文件绘制，从系统加载
GTK 3。版本列表只构建可见行，增加 8 MB PSS。

被测的是端到端构建：发布构建加原生用户界面的 inspector 与 mygo 的自动化插件（测量借它驱动应用）。
它们多占至多 1 MB PSS，发布构建没有。

Android 客户端（`android/`）不在此表：它是包住网页界面的 WebView，内存为系统
WebView 加页面，在设备上衡量，不用本方法。它的工具包要付出什么，见
[Android 应用用 View 还是 Compose](#android-应用用-view-还是-compose)。

## 轻量模式

开启网页界面和轻量模式后，关闭窗口时应用不退出，只是没有窗口（见
[架构](architecture.md#窗口与轻量模式)）。`vfxtool measure-lightweight` 的测量方法与上表相同——
单进程、smaps_rollup、带假 vfox 的新家目录、启动后静置——在同一台 WSL2 主机上，2026-11-04：

| 状态 | RSS | PSS |
| --- | ---: | ---: |
| 窗口打开，主页，网页界面开启 | 62 MB | 48 MB |
| 窗口已关闭（`lightweight-idle`） | 46 MB | 33 MB |
| 窗口已关闭，一个浏览器已配对并走完首次清单（`lightweight-loaded`） | 50 MB | 37 MB |

在应用告诉系统它不要什么（见下）之前，同一台主机上这两行是 RSS 57 与 58 MB、PSS 44 与 45 MB，
每个构建三次测量都一致到 MB。

关闭窗口会释放窗口占用的东西：控件、它绘制所用的共享内存缓冲、页面、列表和监听（约 5 MB）。
不会卸载界面库，而且留下的常驻内存大多并不是应用持有的内存。窗口关闭后留下的 57 MB 常驻内存里，
约 15.6 MB 是私有内存，即堆和栈。其余是文件的干净页，系统需要内存时随时可以收回：被触及过的
可执行文件代码（17 MB：Go 的可执行文件整个被映射，系统每执行一页就把它周围十六页都映射进来）、
界面库映射的字体（10 MB）、GTK 3 与它加载的库。桌面应用只要在运行就一直留着它们，所以轻量模式
只省了几 MB，它的意义不在内存：没有窗口时网页界面、它的会话和运行中的任务都继续，从托盘图标或
再次启动应用就能恢复窗口。

### 应用对文件页做了什么

窗口关闭满两秒后，应用告诉系统这些页暂时不要了（`App.scheduleTrim`，`ui/memory_linux.go` 的
`pageOutFiles`）：对文件的代码与只读数据、且进程没有写过任何一页的那些映射调用 `madvise` 的
`MADV_PAGEOUT`（Linux 5.4 起）。系统立刻收回它们，而不是等别的程序需要内存时再收，于是常驻内存就是
应用实际在运行的部分。别的程序也映射着的页，比如桌面各库的页，由系统原样留下，它只换出单个进程
独占映射的页。两秒内窗口又打开了（它需要这些页）就取消；页在有东西运行它们时回来：关闭一分钟后网页
界面收到的第一个请求，一次测量花 14 ms，下一次不到 1 ms，此前是 1 ms；从托盘图标打开窗口花 0.15 到
0.25 s，和以前（0.19 s）一样。连续采样三分钟，常驻内存没有涨回去。

上表就是这么得来的：静置时 RSS 与 PSS 少 11 到 12 MB，浏览器用过应用之后少 8 MB。私有内存不因此变化，
差别不超过同一构建各次运行之间的差异：找出映射的 `/proc/self/smaps` 读取是逐行读、就地拆分的，不留垃圾，
之后还有一次回收，测量显示有它时私有内存多 0.2 MB。系统没有这个调用、或程序不在 Linux 上时，什么也不发生。

Go 的调度器最多用四个处理器（`main.go` 的 `limitProcessors`），除非用户设置了 `GOMAXPROCS`：应用的
工作是等待，等 vfox，等窗口，而调度器用的每个处理器都带着自己的内存缓存。在 16 线程的机器上，这让私有内存
少 0.5 MB，15.2 对 15.7 MB；四线程或更少的机器没有变化。窗口关闭时才设 `runtime.GOMAXPROCS` 没有帮助
（反而差 0.2 MB）：它只在开头才有用，而在 `main` 开头调用已经得到了变量所能做到的大部分。

### 它没有改变什么

- **界面库的字形图集。**mygo 的文字系统把窗口画过的字形遮罩放在一张图集里，图集满了就翻倍，上限 4096
  乘 4096 字节，也就是 16 MB，之后不再缩小。画过很多种缩放的文字（缩放对话框的弹簧动画，或者尺寸
  异常的第一帧）的窗口会把它留在最大，没有窗口的应用就一直占着这 16 MB。它是界面库的，是进程范围的
  状态，应用够不到。端到端场景读窗口比人频繁得多，大约每五次运行里有一次会这样，而不用自动化插件的
  约四十次运行里一次也没见到。所以托盘图标的开销是用 `vfxtool measure-lightweight --tray` 测的，它不用
  自动化插件。
- **系统自己的回收。**在这台 WSL2 主机上，空闲进程的常驻内存两分钟后会自己下降，从 64 MB 到 40 再到
  36，私有内存不变：系统收回了没有程序在运行的干净页。上表是在十秒后测的，早于这个时刻；而在不做
  这种回收的机器上，换出的页就是系统更早收回的那些。

此前应用有的去头服务器，一个去掉用户界面、从命令行启动的二进制，约占 11 MB。它已经没有了：应用是
桌面应用，没有命令行。没有界面库的进程要么是第二个程序，要么是同一个程序不带窗口再启动一次，而那会
在接手的那一刻丢掉已配对的会话和运行中的任务。

## 托盘图标

应用没有窗口时代表它的托盘图标（见[架构](architecture.md#托盘)）在窗口关闭时创建，窗口打开时
移除。所以上面各行，在没有东西显示图标的地方（如 CI 的运行器），就是没有它的数字：有窗口时它什么也不占。

有托盘的地方，它的开销很小。在同一台主机上、文件页换出之前，用一个注册为
`org.kde.StatusNotifierWatcher` 的替身 StatusNotifier 宿主（工具链的 `trayhost` 包），同一份构建，窗口已关闭，
五次的平均值。那次会话对同一状态读出的数字比上表高约 6 MB，所以只有各行之间的差才有意义：

| 状态 | RSS | 私有内存 |
| --- | ---: | ---: |
| 没有图标，画图标的库已加载（系统有这个库时 mygo 会加载） | 64.1 MB | 15.6 MB |
| 托盘里有图标 | 64.5 MB | 15.7 MB |
| 没有这个库，也就没有图标 | 63.8 MB | 15.5 MB |
| 在没有宿主显示它的地方创建图标，即 `libayatana-appindicator3` 回退到 `GtkStatusIcon` | 71.2 MB | 17.4 MB |

- **显示图标**多占 0.4 MB RSS 与 80 KB 私有内存。这 0.4 MB 是画图标的库的干净页，与同一构建两次运行
  之间的差别相当。
- **这个库**在系统有它时由 mygo 启动时加载，不论是否创建图标，每个状态下约占 0.3 MB RSS 与 70 KB
  私有内存。Debian 包推荐安装它，所以默认安装会有。
- **回退**才是内存会流失的地方：7 MB RSS（其中 1.8 MB 是私有内存），是桌面的图像加载器，GTK 为一个
  没有面板会显示的图标解码图片而启动它们（在 Ubuntu 26.04 上仅 `glycin` 就是 3 MB）。所以
  `trayHostAvailable` 在应用请求图标之前先问会话总线有没有宿主，没有宿主就没有图标。

文件页换出、处理器受限之后，图标远在应用省下的范围之内。托盘里有图标时（在替身宿主的会话里），这台主机上
应用持有 15.2 MB 私有内存，此前没有图标时是 15.6 MB，常驻内存小 11 MB。

CI 让这笔开销保持原样。`vfxtool measure-lightweight --tray`（`measure-remote` 任务的最后一步）用同样的方式
把应用运行两次，一次没有东西显示图标，一次对着替身宿主，图标多占超过 5 MB 常驻内存或 1.5 MB 私有内存就让任务
失败：这留出了运行器的噪声，又小于回退的开销，所以像回退那样创建的图标会被抓住。它不用自动化插件，原因见上面的字形
图集。回退本身，也就是没有宿主时它花掉的那些内存，由图标的场景（`vfxtool e2e-run tray`）把关：没有宿主时应用若创建
了图标，场景就失败。测过的其他把这 0.4 MB 找回来的办法，除了上面两个都不在代码里：窗口关闭后调用 `malloc_trim`
没有变化，`MALLOC_ARENA_MAX=1` 省 0.1 MB，第一次 `debug.FreeOSMemory` 几秒后再来一次，私有内存省 0.2 MB，
RSS 没有变化（换出之后的那次回收就是这个调用）。窗口关闭后 Go 堆里留下的东西，有 mygo 的文字引擎（段落缓存、
字形图集与 SVG 路径），应用没有办法释放它们。

CI 的运行器，即 GitHub 上的 Ubuntu 26.04，与这些数字吻合，同一份代码的两次任务运行都是如此：轻量模式静置在
RSS 45 MB、PSS 31 到 32 MB，配对浏览器之后是 47 到 48 MB 与 34 到 35 MB，表里是 46 与 33、50 与 37；托盘里的图标在那里
多占 0.5 MB RSS 与 0.06 到 0.18 MB 私有内存，本机上是 0.4 MB 与 0.08 MB。

## 桌面之外

Pull Request 评论还跟踪桌面表格没有的两处：轻量模式、Web 客户端堆、Android APK。各有自己的方法、单位与噪声；与上表一样，只比较同评论各行之间。

- **轻量模式。**`vfxtool measure-lightweight` 在显示环境（`xvfb-run`）下启动应用的端到端构建，
  用全新家目录、假 vfox，以及开启网页界面与轻量模式的设置。端到端构建把配对链接写进
  `VFOX_PLUS_E2E_LINK` 指定的文件，并在 `SIGUSR1` 时关闭窗口，release 构建没有这些代码。工具在两个静置点
  打印进程树的 RSS 与 PSS：`lightweight-idle` 窗口已关、无浏览器，`lightweight-loaded` 配对浏览器走完首次
  清单（SSE）。与桌面一样，超过 5 MB 且 5% 才算变化。加 `--tray` 则改为测量托盘图标的开销（见[托盘图标](#托盘图标)）。
- **Android 客户端。**`android.yml` 的 `measure` 任务把 release APK 装到模拟器
  （API 36，x86_64），经 launcher 冷启动、静置后读 `dumpsys meminfo`：
  `apk-cold`，单位 MB。模拟器启动波动大于 GitHub runner，超过 5 MB 且 10%
  才算变化。

CI 中 `mygo.yml` 的 `measure-remote` 任务上传 `memory-lightweight`，
`android.yml` 的 `measure` 任务上传 `memory-android`，`memory.yml`
在任一工作流结束时报告，Android 仍在跑时与其余一样等待。章节渲染在桌面表格
下方，Pull Request 数字与 `main` 并列。

覆盖缺口：`apk-cold` 是未配对首屏不加载任何会话。`vfxtool webheap`
仍可本地看配对后 `/app` 页的 JavaScript 堆（截图附带），只做诊断，不进 CI 报告：
浏览器进程本身的内存不在测量范围内，堆数字不能代表浏览器或 Android 客户端的占用。

### 配对后的会话与无 WebView 对照组

客户端曾经是包住 `/app` 的 WebView。为看清系统 WebView 占多少曾把同一个 APK
配对后在 `webview` 构建与从不构造 WebView 的 `nowebview` 对照构建上各测一遍，
包名与入口相同（两轮之间先卸载）。`vfxtool measure-apk --pair-url`
经显式 `VIEW` intent 打开配对会话（`MainActivity` 的测量入口，
与粘贴链接后点连接是同一个会话），再沉降一次，追加 `memory (apk-paired)` 行。
配对行只做本地诊断：两行的文件不要上传，CI 只测冷启动。
复现配对数字时，用带 `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2`
的应用端到端构建配对，从 `VFOX_PLUS_E2E_LINK`
指定的文件里读链接，每轮重启应用领一条新链接。

| 构建 | `apk-cold` PSS | `apk-paired` PSS |
| --- | ---: | ---: |
| `webview` | 37 MB | 84 MB |
| `nowebview` | 38 MB | 39 MB |

冷启动从不加载 WebView，所以两构建在那里一致。配对会话里系统 WebView 在这里约占
47 MB PSS；原生会话只比自己的冷启动多约 1 MB。全原生客户端把配对内存砍到一半
以下，是带着整个会话实测的，不是占位：主页、带版本与选项的 SDK、插件、任务、设置，
连着实时事件流。结论既出：整个会话用原生重写了一遍（代码在
`dev.vfox.vfoxplus.session`），`webview` 构建已删除，发出的是原生客户端。

### Android 应用用 View 还是 Compose

Android 应用用 View 写，配 Material Components。为了看清 Compose 要付出什么，把它的配对屏用 Compose
又写了一遍（字符串、颜色、图标、形状都相同），2026-10-30 把这几个应用在同一个模拟器（API 36、x86_64、
无 GPU）上以 R8 的 release 构建依次运行：安装、经 launcher 启动、静置 12 秒，从 `dumpsys meminfo` 读 PSS，
每个应用轮流三轮。“新装”指刚安装完，代码在解释器与即时编译中运行；“已编译”指执行
`cmd package compile -m speed -f` 之后，即整个应用已提前编译。

| 应用 | APK（字节） | 新装 PSS（MB） | 已编译 PSS（MB） |
| --- | ---: | ---: | ---: |
| View，用 Material 3 Expressive 重做之前 | 3,176,011 | 40.8 | 44.5 |
| View，用 Material 3 Expressive 重做后 | 3,228,463 | 43.5 | 46.9 |
| Compose，只有配对屏 | 3,589,067 | 44.4 | 47.4 |

- **内存分不出高下。**Compose 屏比 View 应用多 0.9 MB（新装）与 0.5 MB（已编译），与噪声同量级：六次新装它读到
  43.1 到 45.1 MB（原生堆在 9.8 与 11.5 MB 之间跳），View 应用读到 43.4 到 43.6 MB。View 应用这次重做带来的变化更大
  （+2.4 到 +2.7 MB）。内存不是选择任一工具包的理由。
- **APK 体积分得出。**在 View 应用上加 Compose，APK 变成 4,434,858 字节，多 1.2 MB（37%）；只有配对屏、没有扫码、
  会话屏、CameraX 与 zxing 的 Compose 应用，就已经比整个 View 应用大。
- **比较时只带要运行的东西。**第一个 Compose 构建里还留着从未运行的 View 屏幕，读数比 View 应用多约 3 MB（新装）与
  约 6 MB（已编译）：APK 里的代码一旦被系统编译，就会占内存。
- **Compose 屏**用的是 Compose BOM 2026.06.01（Compose 1.11.4、Material 3 1.4.0，只用稳定 API）、
  activity-compose 1.13.0，以及与锁定的 Kotlin 2.1.20 对应的 Compose 编译器插件。
- **绝对数值属于这一次测量**：更早的一次，View 应用读到 30 到 34 MB。只比较表中各行。
- **没有测：**带 Expressive 组件的 Compose（锁定的工具链构建不了，见[为什么用 View 而不是
  Compose](packaging-android.md#为什么用-view-而不是-compose)）、带 WebView 的会话屏、真机。

## 如何测量

`vfxtool measure` 测量应用：

1. 在当前显示器启动应用的端到端构建（`vfxtool build --e2e`，见[上文](#结果)），家目录全新为空，
   vfox 用 [`shared/fixtures`](../../../shared/fixtures) 的假 vfox（回放捕获输出，不跑真实 vfox）。
2. 等 Home 页出现，再等 3 秒沉降，然后从 `/proc/<pid>/smaps_rollup` 累加应用及以下每个进程的
   `Rss` 与 `Pss`。
3. 经窗口元素（mygo 的自动化插件）打开 Node.js 可用版本，等 872 个版本出现，再等 3 秒，再测一次。

大小为 MiB，工具称 MB。沙箱 `measure` 任务在 Docker 中跑全套（`vfxtool sandbox measure`，
只需 Docker）；上表数字来自 16 逻辑 CPU 的 WSL2 主机。

## 解读数字

- **RSS 与 PSS。**RSS 统计进程映射的每页，含共享页，多进程求和会重复计算共享页。
  PSS 把共享页分摊到映射它的进程，求和即应用占用。看 PSS。
- **无 GPU。**虚拟显示器无 GPU，用 OpenGL 绘制的一切走 Mesa 软件光栅（llvmpipe），其内存
  与编译器（LLVM）内存都计入被测进程。应用走处理器，并阻止 GTK 打开 OpenGL 连接：GDK
  打开 OpenGL 连接即映射 Mesa，软件驱动再加编译器 LLVM，比应用本身还大（同构建 Home 页
  GDK 打开时 PSS 约 78 MB，关闭约 43 MB，其它条件相同）。这种体量的界面不需要 GPU，
  内存不依赖本机驱动。真机有 GPU 时数字可能不同，重要时现测。
- **单次测量。**每行只测一次，不做平均，小差异无意义。mygo 在 CI 中反复测量（见下）：
  相同代码 Home 与 Versions 页 PSS 相差至多 1 MB。
- **一次快照。**同样代码隔次测量，Ubuntu 归档更新后的沙箱镜像重建，同样构建可能差几 MB。
  只比较同快照片（本表各行）之间。
- **不含什么。**只测应用进程树：X 服务器、会话总线、桌面门户不在内，vfox 也不在内
  （这里是假 vfox，真实命令会另起 vfox 进程）。

## Pull Request 中的内存

重建应用的 Pull Request 会收到一条内存评论：上表加与 `main` 的 PSS 差。新提交替换
评论，不追加。

- 实现的工作流（[`mygo.yml`](../../../.github/workflows/mygo.yml)）
  跑 `vfxtool measure`（[如何测量](#如何测量)），把打印的行作为构件 `memory-mygo` 上传。
  `vfxtool measure` 也把它们写入 `VFOX_PLUS_MEASURE_OUTPUT` 指定文件。同一工作流的
  `measure-remote` 任务上传 `memory-lightweight`，
  [`android.yml`](../../../.github/workflows/android.yml) 的 `measure` 任务上传
  `memory-android`；评论把它们渲染为表格下方的[桌面之外](#桌面之外)章节。
- [`memory.yml`](../../../.github/workflows/memory.yml) 在该工作流结束时启动，跑
  `vfxtool memory-report`。比较 Pull Request 构件与 `main` 上同工作流最近运行，
  写评论。工作流跑在 `main` 上下文，才能评论 fork 的 Pull Request，只读构件两行数字。
  令牌无评论权限时仅警告，报告进摘要。
- 数字不是上表的数字。它们来自 GitHub runners 单次运行，而非 Docker WSL2 主机。比较
  同评论各行之间及与 `main`，不与本页比。

### 噪声

相同代码每次 CI 内存不完全相同。比较相同代码的运行，Home 与 Versions PSS 相差从未超过
1 MB（Home 最低值的 2.4%；Versions 列表 0 MB）。

评论把超过 5 MB 且超过 `main` 上数字 5% 的差异标为变化并加粗；进程数不同也算变化。
样本小：若提交明显带不动报告的变化，重新测量相同代码的波动并调整 `vfxtool memory-report`
的 `Tolerances`。

### 使用

- 构件 90 天过期。`main` 上 that long 无运行则报告"无基线"，等工作流在 `main` 上再跑。
- `gh workflow run memory.yml -f pull_request=<编号>` 重写某 open Pull Request 的报告。只看不写加
  `--dry-run`：

  ```sh
  GH_TOKEN="$(gh auth token)" vfxtool memory-report --dry-run \
    --repository version-fox/vfox-plus --pull-request <编号> \
    --implementations "mygo"
  ```

- 工具测试跑在 `Repository` 工作流：
  `go -C shared/tooling test ./...`。

## 更新本页

内存明显变化时，跑 `vfxtool sandbox measure`，把
`dist/sandbox/mygo/measure/memory.txt` 的行抄入上表（PSS 在前、RSS 在后），同步抄入
实现的 README 内存节。沙箱镜像重建后重测，保持一快照片（见
[解读数字](#解读数字)）。
