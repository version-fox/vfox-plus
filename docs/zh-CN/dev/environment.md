# 开发环境

> [English](../../en/dev/environment.md)

如何配置一台参与 vfox-plus 开发的机器：Ubuntu 26.04 LTS，单独使用或在 Windows 11 的
WSL2 中。要装什么取决于你做什么：

| 你想 | 你需要 |
| --- | --- |
| 跑全部检查、端到端测试与打包，且不改动你的机器 | Docker，以及构建 `vfxtool` 用的 Go（见[沙箱](#沙箱)） |
| 快速编辑并测试 Go 代码 | Go、几个软件包与 golangci-lint（见[Go 与本机工具](#go-与本机工具)） |
| 构建 Android 客户端 | JDK 17 与 Android SDK（见 [Android](#android)） |
| 在模拟器上运行 Android 客户端 | KVM、一台 adb 设备，或 CI（见[模拟器与 KVM](#模拟器与-kvm)） |

固定的版本在 [`mygo/versions.env`](../../../mygo/versions.env) 与
[`android/versions.env`](../../../android/versions.env)。下面的命令直接读取它们，
所以某个版本改了，本页不会过时。

## 沙箱

`vfxtool sandbox` 在用后即弃的 Docker 容器里运行 CI 的检查、端到端测试与打包
（见[共享沙箱](sandbox.md)），所以只需装 Docker，机器上也什么都不会变：不动工具链，
不动家目录，不动已安装的 vfox。Docker Engine 可用，rootful 或 rootless 皆可
（`vfxtool sandbox` 会区分）。在 WSL2 中，把 Engine 装在发行版里；它需要 systemd
（见 [WSL2](#wsl2)）。

`vfxtool` 是 Go 程序：先用下一节的 Go 构建一次，之后一切都经它运行。第一次运行要构建
镜像，需要一些时间。

```sh
go -C shared/tooling build -o ~/.local/bin/vfxtool ./cmd/vfxtool
vfxtool sandbox all
```

网络慢时，设置 `SANDBOX_UBUNTU_MIRROR` 与 `SANDBOX_GOPROXY`（见
[mygo 指南](implementation-mygo.md#沙箱)）。

## Go 与本机工具

Go 用它的发布压缩包，并对照 `mygo/versions.env` 固定的校验和（`go.mod` 要求 1.27.3
或更高）。在仓库根目录运行：

```sh
. ./mygo/versions.env
curl -fsSL -o /tmp/go.tar.gz "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz"
echo "$GO_SHA256  /tmp/go.tar.gz" | sha256sum --check --strict -
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tar.gz
echo 'export PATH="$HOME/.local/bin:/usr/local/go/bin:$PATH"' >> ~/.profile
```

golangci-lint 用固定的版本，安装方式与沙箱相同：

```sh
. ./mygo/versions.env
archive="golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64"
curl -fsSL -o "/tmp/$archive.tar.gz" \
  "https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_LINT_VERSION/$archive.tar.gz"
echo "$GOLANGCI_LINT_SHA256  /tmp/$archive.tar.gz" | sha256sum --check --strict -
tar -xzf "/tmp/$archive.tar.gz" -C /tmp
install -D -m 755 "/tmp/$archive/golangci-lint" ~/.local/bin/golangci-lint
```

Ubuntu 软件包与 CI 安装的相同（[`mygo.yml`](../../../.github/workflows/mygo.yml) 中的
`LINUX_PACKAGES`）：

```sh
sudo apt install build-essential libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk \
  xvfb xauth dbus-x11
```

`build-essential` 只为单元测试的竞态检测器，它需要 C 编译器；应用本身构建时不需要。
GTK 3 与字体是应用启动时要加载的。`xvfb`、`xauth` 与 `dbus-x11` 给端到端测试一块虚拟显示。
要像 CI 那样检查 `.deb`，再装 `lintian desktop-file-utils`。构建、测试与打包的命令见
[mygo 指南](implementation-mygo.md#日常命令)。

`vfxtool webheap` 测量 Web 界面的堆并截图（拍一个页面，或 Java 发行版之类的场景），它驱动一个讲 DevTools Protocol 的 Chrome：用
`--chrome` 指定，或让工具去找 `google-chrome`、`chromium` 之类（见[截图](contributing.md#截图)，
那里还说明了哪种 Chrome 能给出你要的尺寸）。对真实 vfox 的端到端场景在沙箱里运行，沙箱自带固定版本的
vfox，所以不会碰你的 `~/.vfox`。

## Android

构建需要 JDK 17（CI 用 Temurin）与带 platform 36 和 build-tools 35.0.0 的 Android SDK，经提交
进仓库的 wrapper 运行；见 [Android 打包](packaging-android.md#构建)。构建不需要 KVM，
模拟器需要。

## 模拟器与 KVM

[`android.yml`](../../../.github/workflows/android.yml) 的 `measure` 任务把 release APK 装到模拟器
（API 36、`google_apis`、x86_64）上，并在那里运行 `vfxtool measure-apk`。没有硬件加速，模拟器不会启动
x86_64 镜像，在 Linux 上硬件加速就是 KVM，所以要在你的机器上做同样的事，`/dev/kvm` 必须可用。CI 用 runner
上的 udev 规则启用它。本仓库其他任何事都不需要 KVM。

### 检查

在 Ubuntu 里：

```sh
grep -cE '\b(vmx|svm)\b' /proc/cpuinfo   # 大于 0：处理器提供虚拟化
ls -l /dev/kvm                           # kvm_intel 或 kvm_amd 加载后才存在
id -nG | grep -w kvm                     # 你可以使用它：/dev/kvm 的所属组
journalctl -k -b | grep -E 'Hyper-V: (Host Build|Nested features)'
```

最后一行是给 WSL2 的。`Nested features: 0x0` 表示 Windows 没有把虚拟化提供给 WSL2 虚拟机，
Ubuntu 里的任何操作都改变不了；非 0 的值表示提供了（能用的那台机器上是 `0x4a0000`）。
`journalctl --list-boots` 与 `journalctl -k -b -<n>` 会对更早的几次启动显示同样的行，由此可知变化
发生在哪一天。`sudo apt install cpu-checker` 添加的 `kvm-ok` 会用文字告诉你。

### 让它工作

- 在 WSL2 上，它需要 Windows 11、固件里打开的虚拟化（Intel VT-x 或 AMD-V，SVM），以及
  `%UserProfile%\.wslconfig` 里的 `nestedVirtualization`，该项除非文件另有说明，默认为 `true`
  （[微软文档](https://learn.microsoft.com/zh-cn/windows/wsl/wsl-config)）。更改之后，在 Windows 终端里
  运行 `wsl --shutdown` 重启 WSL2。
- 下一步需要 WSL2 有 systemd：`/etc/wsl.conf` 里 `[boot]` 下的 `systemd=true`。WSL2 内核把 KVM 构建
  为模块，systemd 运行时，处理器提供该扩展，udev 就会加载 `kvm_intel` 或 `kvm_amd`。如果
  `/proc/cpuinfo` 里有 `vmx` 或 `svm` 而 `/dev/kvm` 仍然没有，运行 `sudo modprobe kvm_intel`（或
  `kvm_amd`）会告诉你原因。
- `/dev/kvm` 属于 `kvm` 组：`sudo usermod -aG kvm "$USER"`，然后开一个新会话（在 WSL2 里是
  `wsl --shutdown`）。

### 已知问题：AMD 处理器，2026 年 10 月的 Windows 更新之后

Windows 11 的累积更新 KB5127648（2026 年 10 月；版本 26100.9711，25H2 上是 26200.9711）让 AMD
处理器上的 WSL2 失去了虚拟化：`/proc/cpuinfo` 里没有 `svm`，没有 `/dev/kvm`，
`Nested features: 0x0`，加载模块时是 `kvm_amd: SVM not supported by CPU`，尽管 `.wslconfig` 没错，
固件里 SVM 也开着。WSL 询问 Windows 处理器能做什么，回答以 `ERROR_INVALID_DATA` 失败，WSL 便一声不响地
把嵌套虚拟化省掉了。报告来自 Zen 2 到 Zen 4 的 AMD 处理器，其中有一份报告说一台 Intel 笔记本没受影响。
[microsoft/WSL#41917](https://github.com/microsoft/WSL/issues/41917) 在跟踪它。在一台开发机上，内核日志
能看出是哪一天：2026-10-13 的那次启动有 `Host Build 10.0.26100.9444`、`Nested features: 0x4a0000` 和
`kvm_amd: Nested Virtualization enabled`，2026-10-14 的那次启动是 `Host Build 10.0.26100.9711` 和
`Nested features: 0x0`。

微软在 2026-10-26 写道，它相信修复会在第二天通过 Windows 更新发出。在 Windows 里安装提供的更新，重启，
运行 `wsl --shutdown`，再按上面的方法检查。在那之前，可以在管理员终端里用
`wusa /uninstall /kb:5127648` 卸载该更新（它是安全更新，而且有人用这种方法卸载不掉）。WSL 3.0.4 只是多了一条
警告 `wsl: Failed to detect nested virtualization support, error: ERROR_INVALID_DATA`，并没有解决问题。

### 没有 KVM 时

- 让 CI 来做：`android.yml` 的 `measure` 任务在每个改动 `android/` 或 `mygo/pair/` 的 Pull Request 上运行，
  并上传 `memory-android` 与 `screenshot-android`。
- `vfxtool measure-apk` 只需要 `PATH` 上有 `adb`（`sudo apt install adb`）和 `adb devices` 列得出的一台
  设备：开了无线调试的手机（`adb pair`，再 `adb connect`），或运行在 Windows 一侧、经那里的 adb 服务器连上的
  模拟器（在 WSL 里设置 `ANDROID_ADB_SERVER_ADDRESS` 与 `ANDROID_ADB_SERVER_PORT`，让那台服务器用
  `adb -a nodaemon server` 监听，并让 Windows 防火墙放行）。它的数字属于那台设备，不能与 CI 的模拟器比较。

## WSL2

- **systemd** 是上面几步与 rootless Docker 都需要的。在 `/etc/wsl.conf` 里 `[boot]` 下写
  `systemd=true`，然后 `wsl --shutdown`。
- **内存**默认是机器的一半，除非 `%UserProfile%\.wslconfig` 在 `[wsl2]` 下写了 `memory=<大小>`。
  Go 代码的竞态测试、Docker 构建与 Chrome 是吃内存的。
- 图形程序的**显示**来自 WSLg（`.wslconfig` 里的 `guiApplications` 除非另有说明为 `true`）。需要显示但不要窗口的
  测试用 `xvfb` 或沙箱。
- Linux 文件系统里（`~/...`）的**文件**比 `/mnt/c` 下的快得多：把仓库克隆到 Linux 家目录里。
