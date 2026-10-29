# 内存占用

> [English](../../en/dev/memory.md)

低内存占用是项目目标（见[架构](architecture.md#内存)）。
本页记录应用在 Linux 上的占用，同一方法测量，便于发现回归。数字是快照：行是沙箱
`measure` 任务单次运行的结果，同样数字见
[mygo README(implementation-mygo.md#内存)。

## 结果

单位 MB，应用及其全部子进程之和，Docker 内 Ubuntu 26.04，Xvfb（1280×800，无 GPU）。
"Versions" 是列出 Node.js 872 个可用版本的 SDK 页。

| 实现 | 技术 | 进程 | Home PSS | Home RSS | Versions PSS | Versions RSS | 列表增加 (PSS) |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| [mygo](../../../mygo/README.zh-CN.md) | mygo 0.7.3、Go 1.27.3、GTK 3（purego）、CPU 渲染 | 1 | 50 | 62 | 58 | 73 | +8 |

mygo（Home 页 PSS 50 MB，单进程）用处理器从 18 MB 的 Go 可执行文件绘制，从系统加载
GTK 3。版本列表只构建可见行，增加 8 MB PSS。

Android 客户端（`android/`）不在此表：它是包住配对服务的 WebView，内存为系统
WebView 加页面，在设备上衡量，不用本方法。

## 配对服务与后台服务

带模型的 `vfox-plus serve` 与去掉界面的 `vfox-plus-serve serve`
空闲时体量相近，测量方法同上：单进程、smaps_rollup、新家目录、无 vfox、启动后静置：

| 服务 | 二进制 | RSS | PSS |
| --- | --- | ---: | ---: |
| `vfox-plus serve` | 桌面二进制，18 MB | 16 MB | 16 MB |
| `vfox-plus-serve serve` | 去头二进制，12 MB | 11 MB | 11 MB |

去头二进制（`mygo/cmd/vfox-plus-serve`：同样的 `pair`、`serve`、`service` 命令，去掉 mygo 用户界面）减掉约三分之一的二进制和常驻内存；空闲模型本身只占约 1 MB，其余是 Go 运行时与触及过的二进制代码页。没有唯一稳定的数字：启动后立刻读会偏低（代码页尚未缺页），GC 时机也会让堆差出几 MB。与上表一样当一次快照看，跨机器不比，关键比较重测。桌面在打开远端访问时把同一服务内嵌在自己进程里，不多占进程，只多一个监听与空闲 goroutine；默认关闭时一无所增。

## 桌面之外

Pull Request 评论还跟踪桌面表格没有的三处：去头服务、Web 客户端堆、Android APK。各有自己的方法、单位与噪声；与上表一样，只比较同评论各行之间。

- **去头服务。**`vfxtool measure-serve` 用全新家目录与假 vfox 在回环上启动
  `vfox-plus-serve serve`，在两个静置点打印服务进程树的 RSS 与 PSS：`serve-idle`
  无浏览器，`serve-loaded` 配对浏览器走完首次清单（SSE）。与桌面一样，超过
  5 MB 且 5% 才算变化。
- **Web 客户端堆。**`vfxtool webheap`（只需要一个 Chrome）用
  Chrome DevTools Protocol 在 headless Chrome 中打开配对后的 `/app` 页，
  写两处 JavaScript 堆字节数：清单列表静置后 `app`，打开页面带实时事件空闲
  20 秒后 `app-settled`（可发现事件流泄漏）。报告按 KB 展示；超过 100 KB 且
  5% 才算变化。
- **Android 客户端。**`android.yml` 的 `measure` 任务把 release APK 装到模拟器
  （API 36，x86_64），经 launcher 冷启动、静置后读 `dumpsys meminfo`：
  `apk-cold`，单位 MB。模拟器启动波动大于 GitHub runner，超过 5 MB 且 10%
  才算变化。

CI 中 `mygo.yml` 的 `measure-remote` 任务上传 `memory-serve` 与 `memory-web`，
`android.yml` 的 `measure` 任务上传 `memory-android`，`memory.yml`
在任一工作流结束时报告，Android 仍在跑时与其余一样等待。章节渲染在桌面表格
下方，Pull Request 数字与 `main` 并列。

覆盖缺口：没有配对后 WebView APK 状态（模拟器无可配对服务，`apk-cold` 是未配对
首屏），堆在桌面 Chrome 中测量，而非 APK 内嵌的系统 WebView，其回收器行为
可能不同。

## 如何测量

`vfxtool measure` 测量应用：

1. 在当前显示器启动应用，家目录全新为空，vfox 用
   [`shared/fixtures`](../../../shared/fixtures) 的假 vfox（回放捕获输出，不跑真实 vfox），
   跑应用的 `measure` 场景。
2. 应用展示 Home 页，等待 3 秒沉降后报告 `e2e: MEASURE home`。工具读报告，从
   `/proc/<pid>/smaps_rollup` 累加应用及以下每个进程的 `Rss` 与 `Pss`；应用再等 5 秒，
   保证测量时页面不变。
3. 应用打开 Node.js 可用版本，等待 3 秒后报告 `e2e: MEASURE nodejs`，工具再测一次。

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
  跑[如何测量](#如何测量)的 `measure` 场景，把打印的行作为构件 `memory-mygo` 上传。
  `vfxtool measure` 也把它们写入 `VFOX_PLUS_MEASURE_OUTPUT` 指定文件。同一工作流的
  `measure-remote` 任务上传 `memory-serve` 与 `memory-web`，
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
