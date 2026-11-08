# 开发环境

> [English](../../en/dev/environment.md)

如何配置一台参与 vfox-plus 开发的机器：Ubuntu 26.04 LTS，单独使用或在 Windows 11 的
WSL2 中，或者用装齐了这一切的开发容器。要装什么取决于你做什么：

| 你想 | 你需要 |
| --- | --- |
| 在编辑器里一次拥有全部工具，且不改动你的机器 | Docker，以及能打开开发容器的编辑器（见[开发容器](#开发容器)） |
| 跑全部检查、端到端测试与打包，且不改动你的机器 | Docker，以及构建 `vfxtool` 用的 Go（见[沙箱](#沙箱)） |
| 快速编辑并测试 Go 代码 | Go、几个软件包与 golangci-lint（见[Go 与本机工具](#go-与本机工具)） |
| 构建 Android 客户端 | JDK 17 与 Android SDK（见 [Android](#android)） |
| 在模拟器上运行 Android 客户端 | KVM、一台 adb 设备，或 CI（见[模拟器与 KVM](#模拟器与-kvm)） |

固定的版本在 [`mygo/versions.env`](../../../mygo/versions.env) 与
[`android/versions.env`](../../../android/versions.env)。下面的命令直接读取它们，
所以某个版本改了，本页不会过时；开发容器的镜像在构建时也读取它们。

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

网络慢时，设置 `SANDBOX_UBUNTU_MIRROR`、`SANDBOX_GOPROXY` 与 `SANDBOX_NPM_REGISTRY`（见
[mygo 指南](implementation-mygo.md#沙箱)）。

## 开发容器

[`.devcontainer`](../../../.devcontainer) 描述了一个容器，下面各节要装的东西它都有，所以你什么都不用装：
Ubuntu 26.04，固定版本的 Go、golangci-lint 与 vfox，CI 安装的软件包，一块虚拟显示，JDK 与 Android SDK，
以及 Docker。VS Code（装有 Dev Containers 扩展）、GitHub Codespaces 与
[开发容器 CLI](https://github.com/devcontainers/cli) 都能打开它。镜像面向 x86-64，与它所装的压缩包一致；
在别的处理器上，Docker 以模拟方式运行它。

在 VS Code 里打开仓库，运行 *Dev Containers: Reopen in Container*（在容器中重新打开）。在终端里，Docker 运行时：

```sh
npm install --global @devcontainers/cli
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . bash
```

首次启动要构建镜像，需要几分钟，下载约 1 GB，占用 3.6 GB 磁盘，然后运行
[`devcontainer.json`](../../../.devcontainer/devcontainer.json) 的 `postCreateCommand`：把 `vfxtool`
构建到 `/home/vscode/go/bin`（用户的 `~/go/bin`，在 `PATH` 上），下载 Go 模块，并准备应用内嵌的资源
（不含 Go 模块的许可证，那只有打包才需要；不带 `MYGO_SKIP_LICENSES` 的 `vfxtool prepare` 会收集它们）。
之后，[mygo 指南](implementation-mygo.md#日常命令)与[参与贡献](contributing.md#工作流)里的命令照写的样子就能运行：

```sh
cd mygo
golangci-lint run && CGO_ENABLED=1 go test -race ./...
vfxtool build --e2e build/vfox-plus-e2e
xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- vfxtool e2e-run real build/vfox-plus-e2e
```

`VFOX_PLUS_E2E_VFOX` 指向镜像装的 vfox，版本就是 `mygo/versions.env` 固定的，所以对真实 vfox 的场景不用加前缀，
也没有你的 `~/.vfox` 可碰。`ANDROID_HOME` 已设好，`cd android && ./gradlew :app:assembleRelease`
即可构建 Android 客户端（见 [Android](#android)）。

容器带 init 进程运行（`init`），用来回收测试留下的进程；并带 1 GB 共享内存（`--shm-size`），因为软件渲染和
内嵌的测试服务器需要的，比 Docker 默认给容器的 64 MB 多。沙箱的容器也是这样运行的。

### 重建

Dockerfile 在构建镜像时读取 `mygo/versions.env` 与 `android/versions.env`，CI 与沙箱也这样做，所以那里改了
某个版本（比如一次 `git pull`），重建容器时它就会进入容器：*Dev Containers: Rebuild Container*（重建容器），或
`devcontainer up --workspace-folder . --remove-existing-container`。工具在 `/usr/local` 与 `/opt`，供用户使用而不是修改。
用户下载的东西放在 `/home/vscode` 下，两个 Docker 卷让它们在容器重建后得以保留：`vfox-plus-devcontainer-go`
（Go 的模块与构建缓存）与 `vfox-plus-devcontainer-gradle`（Gradle 与 Android 应用的库）。`docker volume rm` 可删除它们。

### 网络慢时

沙箱的那些变量在这里同样有效。在启动编辑器或 CLI 的环境里设置 `SANDBOX_UBUNTU_MIRROR`（构建镜像时用）、
`SANDBOX_GOPROXY` 与 `SANDBOX_NPM_REGISTRY`（它们以 `GOPROXY` 与 `VFOX_PLUS_NPM_REGISTRY` 进入容器）。
Go、golangci-lint、vfox 与 Android 命令行工具分别来自 go.dev、GitHub 与 dl.google.com，Docker feature 来自 ghcr.io，
它们都没有可指定的镜像（连不上 ghcr.io 时怎么办，见[容器里的 Docker](#容器里的-docker)）。
你的 `~/.gradle` 不是容器的：Gradle wrapper 从 `gradle-wrapper-neo.json` 读取的下载镜像
（见[构建](packaging-android.md#构建)）要放在容器的 Gradle 用户目录 `/home/vscode/.gradle`（`$GRADLE_USER_HOME`）里。

### Rootless Docker

Rootless Docker 把容器的 root 映射为你，把其余用户都映射为从属用户，而从属用户不能写你的文件：在以 `vscode`
运行的容器里，仓库是只读的。在启动编辑器或 CLI 的环境里设置 `VFOX_PLUS_DEVCONTAINER_USER=root`（写在
`~/.profile` 里，先于 `code .` 或 `devcontainer up`），容器就以 root 运行，也就是容器外的你：它写的文件归你。
用 Docker Desktop、以 root 运行的 Docker Engine 或 Codespaces 时不要动它。此时用户是 `vscode`，有 `sudo` 且无需密码，
在 Linux 上它会取得你的 uid（`updateRemoteUserUID`）。

### 容器里的 Docker

Docker Engine 在容器里运行，有自己的镜像，靠
[`docker-in-docker`](https://github.com/devcontainers/features/tree/main/src/docker-in-docker)
feature，它让容器成为特权容器。本仓库用到 Docker 的三件事在里面照写的样子就能工作：`vfxtool sandbox`，
以及[参与贡献](contributing.md#markdown)里的许可证头检查与 Markdown 检查。容器本身已经是带有沙箱工具的用后即弃环境，
所以只有沙箱固定下来的那些才需要在里面再用沙箱：装进干净 Ubuntu 的安装测试（`package`）、内存（`measure`）与桌面应用的截图
（`screenshots`）。不允许特权容器、或连不上 ghcr.io（feature 的来源）的地方，从 `devcontainer.json` 里删掉
`features`；其余一切都不需要它。

### 限制

- 没有模拟器，也没有 KVM（见[模拟器与 KVM](#模拟器与-kvm)）：用 CI 的 `measure` 任务，或像[没有 KVM 时](#没有-kvm-时)那样用一台 adb 设备。
- 没有显示。应用的窗口需要显示：测试在 `xvfb` 里运行，要使用应用，运行 `vfox-plus-serve`，在容器外配对一个浏览器。
- 没有 Chrome，`vfxtool webheap` 驱动的那个（见 [Go 与本机工具](#go-与本机工具)）。自己装一个，用 `--chrome` 指定。
- git worktree 是这样一个文件夹：它的 `.git` 是个文件，指向它所属的克隆，而那在容器挂载的文件夹之外，所以 git
  在容器里用不了（Go 仍能构建）。请打开那个克隆。

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
sudo apt install build-essential libgtk-3-0t64 libayatana-appindicator3-1 fontconfig fonts-ubuntu fonts-noto-cjk \
  xvfb xauth dbus-x11
```

`build-essential` 只为单元测试的竞态检测器，它需要 C 编译器；应用本身构建时不需要。
GTK 3 与字体是应用启动时要加载的。`xvfb`、`xauth` 与 `dbus-x11` 给端到端测试一块虚拟显示。
要像 CI 那样检查 `.deb`，再装 `lintian desktop-file-utils`。`libayatana-appindicator3-1` 画出应用没有窗口时代表它的托盘图标，端到端场景 `tray` 需要它，并且要在
`dbus-run-session` 给的会话总线里运行，因为它用一个替身托盘宿主（见[托盘](architecture.md#托盘)）。要在自己的桌面上
看到这个图标，桌面需要是 GNOME（带它的 AppIndicator 扩展）或 Windows。场景 `tray-gnome` 在
`shared/sandbox/gnome.Dockerfile` 的容器里运行，所以需要 Docker。构建、测试与打包的命令见
[mygo 指南](implementation-mygo.md#日常命令)。

`vfxtool webheap` 给 Web 界面截图（拍一个页面，或 Java 发行版之类的场景），附带看一眼它的 JavaScript 堆做本地诊断，
`vfxtool gateway-shots` 拍网关各页的截图，它们都驱动一个讲 DevTools Protocol 的 Chrome：用
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
  `wsl --shutdown`，它也会结束 WSL2 里运行的一切）。想省掉新会话，就用 CI 用的那条规则把设备开放给所有人：
  它立刻生效，已经在运行的程序也包括在内，重启之后仍然有效；只有一个人使用的电脑上，没有谁需要被挡在外面：

  ```sh
  echo 'KERNEL=="kvm", GROUP="kvm", MODE="0666", OPTIONS+="static_node=kvm"' | sudo tee /etc/udev/rules.d/99-kvm4all.rules
  sudo udevadm control --reload-rules
  sudo udevadm trigger --name-match=kvm
  ```

  这时 `ls -l /dev/kvm` 显示 `crw-rw-rw-`。下面这条直接问内核 KVM 能不能用，能用就打印 12
  （`PermissionError` 说明你还不能使用这个设备）：

  ```sh
  python3 -I -c 'import fcntl,os; print(fcntl.ioctl(os.open("/dev/kvm", os.O_RDWR), 0xAE00))'
  ```

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

### 自己的模拟器

KVM 能用之后，模拟器需要 SDK 的命令行工具，即 `android/versions.env` 固定的那个版本（dl.google.com 上的
`commandlinetools-linux-<build>_latest.zip`，按它的校验和核对，解压到 `$ANDROID_HOME/cmdline-tools/latest`），以及三个包。
下载约 3 GB，`sdkmanager` 会要求你接受 Android SDK 的许可：

```sh
export ANDROID_HOME=$HOME/Android/Sdk
export PATH=$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$PATH
sdkmanager --licenses
sdkmanager platform-tools emulator "system-images;android-36;google_apis;x86_64"
avdmanager create avd -n pixel6 -k "system-images;android-36;google_apis;x86_64" -d pixel_6
emulator -avd pixel6 -no-window -no-audio -no-boot-anim -no-snapshot -gpu swiftshader_indirect &
adb wait-for-device && adb shell getprop sys.boot_completed   # 启动完成后是 1，约一分钟
```

`pixel_6` 是 1080×2400、420 dpi，和 CI 的模拟器一样。`~/.android/avd/pixel6.avd/config.ini` 里有
`hw.camera.back=virtualscene` 时，扫码界面的后置摄像头显示一间客厅（`avdmanager` 写的是 `emulated`，只是一段
普通动画）；对 WebView 来说 `hw.ramSize=3072M` 比默认的 1536M 合适。去掉 `-no-window` 就能在显示器上
（WSLg）看到模拟器。不想自己构建、直接装 CI 构建好的 release APK，可以从该 Pull Request 的运行里下载：
`gh run download RUN -n vfox-plus-android`。

### 运行 App

模拟器启动后（`adb devices` 能列出它），构建 App、安装，并与你机器上的服务器配对：

```sh
cd android && ./gradlew :app:assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell pm grant dev.vfox.vfoxplus android.permission.CAMERA   # 供扫码页使用
```

模拟器把你的机器当作 `10.0.2.2`。服务器的会话 cookie 带 `Secure`，WebView 只为 `https://` 与回环地址保留它，所以
用 `http://10.0.2.2:PORT` 配对行不通；给其他设备的服务器本来也只讲 HTTPS。做一份写有 `10.0.2.2` 的自签名证书，
交给应用：

```sh
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -days 30 \
  -subj /CN=vfox-plus-emulator -addext subjectAltName=IP:10.0.2.2,IP:127.0.0.1 \
  -keyout key.pem -out cert.pem
```

启动应用（`vfxtool build build/vfox-plus && build/vfox-plus`），在它的设置里开启网页界面和*其他设备*，再选择
*证书 → 浏览…* 和文件 `cert.pem`、`key.pem`。（也可以在启动前把这些设置写进
`~/.config/vfox-plus/settings.json`：`"webInterface": true`、`"webRemote": true`、
`"webCertificate": "/path/cert.pem"` 与 `"webKey": "/path/key.pem"`。）

端到端构建（`vfxtool build --e2e`）既不需要证书，也不用替换地址：环境里有 `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2` 时，它
为自己生成的证书写的就是 `10.0.2.2`，设置里和*显示二维码*给出的链接也是。指南里的图就是这样拍的
（[截图](contributing.md#截图)）。

点*配对手机*旁的*显示二维码*，复制链接，输入到 App 里，并把其中你机器的地址换成 `10.0.2.2`。App 会显示证书的指纹，它必须与设置里显示的、以及
`openssl x509 -in cert.pem -noout -fingerprint -sha256` 打印的相同。

`vfxtool measure-apk` 在同一个模拟器上运行。它安装时不带 `-r`，所以要先卸载 App（`adb uninstall dev.vfox.vfoxplus`）。

## WSL2

- **systemd** 是上面几步与 rootless Docker 都需要的。在 `/etc/wsl.conf` 里 `[boot]` 下写
  `systemd=true`，然后 `wsl --shutdown`。
- **内存**默认是机器的一半，除非 `%UserProfile%\.wslconfig` 在 `[wsl2]` 下写了 `memory=<大小>`。
  Go 代码的竞态测试、Docker 构建与 Chrome 是吃内存的。
- 图形程序的**显示**来自 WSLg（`.wslconfig` 里的 `guiApplications` 除非另有说明为 `true`）。需要显示但不要窗口的
  测试用 `xvfb` 或沙箱。
- Linux 文件系统里（`~/...`）的**文件**比 `/mnt/c` 下的快得多：把仓库克隆到 Linux 家目录里。
- **开发容器**可与 Docker Desktop 的 WSL 集成配合，也可与发行版里的 Docker Engine 配合。在 VS Code 的 WSL
  窗口里打开克隆（在发行版里运行 `code .`），再在容器中重新打开。Engine 是 rootless 的，就先设置
  `VFOX_PLUS_DEVCONTAINER_USER=root`（见 [Rootless Docker](#rootless-docker)）。
