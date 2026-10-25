# 内存占用

> [English](../memory.md)

低内存占用是项目目标（见[架构](architecture.md#内存)）。
本页记录应用在 Linux 上的占用，同一方法测量，便于发现回归。数字是快照：行是沙箱
`measure` 任务单次运行的结果，同样数字见
[mygo README](../../mygo/README.zh-CN.md#内存)。

## 结果

单位 MB，应用及其全部子进程之和，Docker 内 Ubuntu 26.04，Xvfb（1280×800，无 GPU）。
"Versions" 是列出 Node.js 872 个可用版本的 SDK 页。

| 实现 | 技术 | 进程 | Home PSS | Home RSS | Versions PSS | Versions RSS | 列表增加 (PSS) |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| [mygo](../../mygo/README.zh-CN.md#内存) | mygo 0.6.3、Go 1.27.3、GTK 3（purego）、CPU 渲染 | 1 | 45 | 58 | 53 | 67 | +8 |

mygo（Home 页 PSS 45 MB，单进程）用处理器从 12 MB 的 Go 可执行文件绘制，从系统加载
GTK 3。版本列表只构建可见行，增加 8 MB PSS。

## 如何测量

`vfxtool measure` 测量应用：

1. 在当前显示器启动应用，家目录全新为空，vfox 用
   [`shared/fixtures`](../../shared/fixtures) 的假 vfox（回放捕获输出，不跑真实 vfox），
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

- 实现的工作流（[`mygo.yml`](../../.github/workflows/mygo.yml)）
  跑[如何测量](#如何测量)的 `measure` 场景，把打印的行作为构件 `memory-mygo` 上传。
  `vfxtool measure` 也把它们写入 `VFOX_PLUS_MEASURE_OUTPUT` 指定文件。
- [`memory.yml`](../../.github/workflows/memory.yml) 在该工作流结束时启动，跑
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
