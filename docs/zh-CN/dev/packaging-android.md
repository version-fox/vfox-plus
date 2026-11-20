# 手机端打包

> [English](../../en/dev/packaging-android.md)

手机端是 [`mygo/phone`](../../../mygo/phone)：mygo 的安卓原生应用。它的屏幕是 mygo 的原生用户界面，既不是 Android View，也不是 WebView。它与运行 vfox-plus 的电脑配对，然后经[远端协议](remote-protocol.md)显示那台电脑上的 SDK、插件与任务。用户装好 `.apk` 后与电脑配对，见[安装](../user/install.md#安卓)与[安卓 App](../user/android.md)。

应用由 mygo 构建：`mygo build -platform android` 从 `mygo/phone` 写出安卓工程，运行 Gradle，并把 APK 与 AAB 写到 `mygo/phone/dist/android/`。仓库里没有安卓工程。

## 布局

| 路径 | 内容 |
| --- | --- |
| `main.go` | 入口。在 Linux 上，于 `mygo/` 目录运行 `go run ./phone`，即可在手机大小的窗口里打开应用，供开发用。 |
| `app/` | 应用的模型：屏幕所显示的页面、标签页与后端。 |
| `screens/` | 由 mygo 绘制的屏幕：配对屏，以及五个标签页：主页、SDK、插件、任务与设置。 |
| `pairing/` | 配对链接、电脑证书的指纹，以及从照片中读出承载链接的二维码。 |
| `remote/` | [远端协议](remote-protocol.md)的客户端。 |
| `store/` | 应用在两次启动之间保存的内容。 |
| `i18n/` | 文本，英文与简体中文。 |
| `resources/` | 图标的源文件。`vfxtool appicon` 在构建前写出它们（见[图标](#图标)）。 |
| `mygo.json` | 应用本身：名称、标识 `dev.vfox.vfoxplus`、slug、版本、URL 协议 `vfoxplus`，以及安卓的自适应图标。 |
| `versions.env` | 钉住的版本（见[钉住的版本](#钉住的版本)）。 |

## 配对

配对从一条配对链接开始。`vfoxplus://pair?link=<链接>`（协议名取自 `mygo.json`）打开配对屏，链接已经填好。用户核对证书的指纹，然后点*信任并配对*（英文界面为 Trust and pair）。这是唯一信任证书、并完成配对的按钮。配对成功时，应用记一行日志：`vfoxplus: paired host=<主机> fingerprint=<指纹>`。

`vfxtool measure-apk` 走的就是这条路径。它查找那行日志，以及按钮与标签页的文字（见[维护者注记](#维护者注记)）。

## 构建

构建需要 [`mygo/phone/versions.env`](../../../mygo/phone/versions.env) 钉住的工具（见[钉住的版本](#钉住的版本)）：JDK 17、平台 36 与构建工具 36.0.0 的 Android SDK、NDK 28.2.13676358（应用的 Go 代码由它的 clang 为各个 ABI 编译），以及 mygo CLI。[安卓](environment.md#android)一节说明怎样安装它们，[开发容器](environment.md#开发容器)里则全部装好了。mygo 为安卓工程运行的 Gradle，首次构建时要下载它的发行版与依赖，需要几分钟；它们留在 Gradle 的用户目录里：本机是 `~/.gradle`，容器里是 `/home/vscode/.gradle`。

在仓库根目录，用[沙箱](environment.md#沙箱)所说的方法构建 `vfxtool`，然后运行：

```sh
vfxtool appicon mygo/phone/resources
cd mygo/phone
mygo build -platform android
```

`mygo build -platform android` 生成发布所用的 APK 与 AAB。`-platform android/amd64` 只生成 x86_64 的那一对，模拟器运行的就是它，CI 构建的也是它。图标要先写，并且要在仓库根目录写：`vfxtool appicon` 的路径相对于仓库根目录，若在 `mygo/phone` 里运行，图标会写到 `mygo/phone/mygo/phone` 下面。

配置里没有 keystore，所以 `mygo build` 用本机的 debug 密钥签名（Gradle 在首次构建时生成它）。这样的 APK 只供测试：发布时会替换它的签名（见[签名状态](#签名状态)）。产物在 `mygo/phone/dist/` 中，被 git 忽略。要把 APK 装到模拟器或手机上，用 `adb install -r`，见[运行 App](environment.md#运行-app)。

在 Linux 上，于 `mygo/` 目录运行 `go run ./phone`，会在手机大小的窗口里打开同一个应用，看屏幕时比模拟器快。

## 钉住的版本

| 变量 | 钉住的内容 |
| --- | --- |
| `MYGO_VERSION` | mygo CLI，用 `go install github.com/egoist/mygo/cmd/mygo@v<版本>` 安装。它必须与 `mygo/go.mod` 中 `mygo` 模块的版本一致。 |
| `JAVA_VERSION` | JDK，17：mygo 运行的 Gradle 8.14.3 不能在 JDK 25 上运行。CI 用 Temurin 17，开发容器用 Ubuntu 软件源里的 OpenJDK 17。 |
| `ANDROID_CMDLINE_TOOLS_BUILD`、`ANDROID_CMDLINE_TOOLS_SHA256` | 开发容器安装的 SDK 命令行工具：dl.google.com 发布的 zip 文件名中的构建号，及其校验和。 |
| `ANDROID_SDK_PLATFORM`、`ANDROID_SDK_BUILD_TOOLS` | 平台 `android-36` 与构建工具 36.0.0，后者含 `aapt2` 与 `apksigner`。 |
| `ANDROID_GRADLE_BUILD_TOOLS` | 构建工具 35.0.0：Android Gradle 插件除上面那些之外还要它，缺了它会在构建中途下载，而在 SDK 不可写的地方（如开发容器）下载会失败。 |
| `ANDROID_NDK_VERSION` | NDK，28.2.13676358。 |
| `ANDROID_EMULATOR_API`、`ANDROID_SYSTEM_IMAGE_TARGET` | [CI](#持续集成) 的模拟器：API 36，x86_64 的 `google_apis`。 |

CI、开发容器及其检查（[`devcontainer.yml`](../../../.github/workflows/devcontainer.yml)）都读取这个文件，所以改动会传到它们全部。

## 图标

仓库里不放二进制文件。`vfxtool appicon mygo/phone/resources` 从应用图标的路径绘制 `icon.png` 与 `icon-foreground.png`（见[应用图标](design-system.md#应用图标)），两者都被 git 忽略。自适应图标的背景是 `#1B1268`，由 `mygo.json` 指定。CI 与[开发容器](environment.md#开发容器)在构建前写出图标。

## 签名状态

安卓只会把更新装在用同一把密钥签名的应用上，遇到别的密钥会拒绝（`INSTALL_FAILED_UPDATE_INCOMPATIBLE`）。所以 vfox-plus 的每个发布版本都用**发布密钥**签名：一把密钥，存为仓库 `release` 环境里的 secret，每次发布都用它。新版本直接装在旧版本上，应用数据与电脑的配对都保留。

APK 分两步签名，见 [`release.yml`](../../../.github/workflows/release.yml)：

- `build` 任务用 mygo 生成 APK，用运行器的 debug 密钥签名，并以 `vfox-plus-android-unsigned` 上传。这个任务没有任何 secret，所以发布密钥不会到达它。
- `sign-windows` 任务在批准之后用 [`shared/scripts/sign-apk.ps1`](../../../shared/scripts/sign-apk.ps1) 签名 APK。`zipalign -f -P 16 4` 重新对齐（原生库按 16 KB 页对齐），`apksigner sign` 用发布密钥替换 debug 签名。脚本随后检查签名，检查证书里已没有 `CN=Android Debug`，并用 `zipalign -c` 检查签名之后对齐仍然成立。任务把 APK 写到 `dist-android/vfox-plus_<版本>.apk`。`vfxtool check-assets` 在发布之前要求这样的文件恰有一个。

**演练。** 发布在合并之前无法试：它从 `main` 手动发起，发布密钥又在只有 `main` 与标签能用的环境里。所以每当手机端应用或这个脚本有改动，[`phone.yml`](../../../.github/workflows/phone.yml) 的 `signing` 任务就像发布那样在 Windows 运行器上运行同一个脚本。密钥由任务自己生成（一个口令谁也不知道的 PKCS12 keystore），APK 是 `build` 任务用 debug 密钥生成的那个。任务先检查构建出的 APK 确实是 mygo 的 debug 签名，再签名，最后检查结果带的是演练的证书，没有 debug 证书。演练够不到的是只属于发布的部分：密钥从哪里来（下面四个 secret）、给 Windows 包签名的 Azure 步骤、批准，以及版本。这些靠在 `main` 上对 Release 工作流做一次 dry run 来试：Actions，Release，Run workflow，勾选 `dry-run`（构建并签名一切，不推送任何东西）。改过签名之后做一次，设置好环境之后也做一次。

没有下面四个 secret，任务拒绝运行：`ANDROID_RELEASE_KEYSTORE_BASE64`（base64 编码的 keystore）、`ANDROID_RELEASE_STORE_PASSWORD`、`ANDROID_RELEASE_KEY_ALIAS` 与 `ANDROID_RELEASE_KEY_PASSWORD`。它们只放在 `release` 环境里，仓库其他地方没有，也只有 `sign-windows` 读取。keystore 写到运行器的临时文件中，无论结果如何都会删除。

运行要在 `sign-windows` 签名 Windows 包与 APK 之前，等 `@version-fox/vfox-maintainers` 或 `@version-fox/vfox-committers` 的成员批准一次；批准之后发布紧接着进行，不再等第二次。仓库管理员发起的运行可以跳过审批。只有来自 `main` 或 `v*` 标签的运行能使用这个环境。GitHub 不显示 secret 的值，维护者也不分享。如需这四个 secret 的实际值，请联系 <https://github.com/linghengqian>。

设置发布密钥只需一次。先在仓库设置里建 `release` 环境（Environments）：Required reviewers 选两个团队 `vfox-maintainers` 与 `vfox-committers`，勾选 "Allow administrators to bypass configured protection rules"，Deployment branches and tags 只允许 `main` 与 `v*`。然后把 keystore 与口令放进密码管理器，并另存一份：没有它们，以后的任何发布都不能升级已装的应用。

```sh
keytool -genkeypair -v -keystore vfox-plus-release.jks -alias vfox-plus \
  -keyalg RSA -keysize 4096 -validity 10000
base64 -w0 vfox-plus-release.jks | gh secret set ANDROID_RELEASE_KEYSTORE_BASE64 --env release -R version-fox/vfox-plus
gh secret set ANDROID_RELEASE_STORE_PASSWORD --env release -R version-fox/vfox-plus
gh secret set ANDROID_RELEASE_KEY_ALIAS --env release -R version-fox/vfox-plus
gh secret set ANDROID_RELEASE_KEY_PASSWORD --env release -R version-fox/vfox-plus
```

`.jks` 文件不要放进仓库：`*.jks` 与 `*.keystore` 已被忽略。

## 版本与更新

安卓只接受用同一把签名密钥、且 `versionCode` 更高的更新。发布时，版本写进带有它的文件，其中包括 `mygo/VERSION` 与 `mygo/phone/mygo.json` 的 `version`（由 `vfxtool set-version` 完成），所以手机显示的版本与桌面应用相同。`versionCode` 是 mygo 的默认值，即 `主版本 × 10000 + 次版本 × 100 + 补丁号`，1.2.3 为 10203。因此次版本号与补丁号必须小于 100，否则版本码就不再随版本递增。

由 debug 密钥签名的 APK，例如 CI 或你本机的 `mygo build` 生成的，不是发布版本：它不能升级到发布版本，发布版本也不能升级它。先卸载它，这同时也会忘掉与电脑的配对。

## 持续集成

[`phone.yml`](../../../.github/workflows/phone.yml) 构建应用，并在模拟器上驱动它。应用、它所配对的代码、共享工具与钉住的版本有改动时运行：

- `build` 任务生成 x86_64 的 APK 与 AAB，用 debug 密钥签名，以 `vfox-plus-android-amd64` 上传。
- `emulator` 任务在虚拟显示中，对着假 vfox 启动桌面应用的端到端构建，并设 `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2`（模拟器访问运行器的地址）：各标签页里有的，与真人看到的一样，是已安装的 SDK 和可安装的插件。它启动模拟器（API 36、`google_apis`、x86_64，软件 GPU），运行 `vfxtool phone-link`（与桌面应用配对，并写出给 `10.0.2.2` 的配对链接）以及 `vfxtool measure-apk`。后者安装 APK，冷启动五次并在每次启动后测量（那一行取中位数，见[手机端应用](memory.md#手机端应用)），在配对对话框开着时按返回键（对话框必须关上，应用必须仍在前台，`--check-back-key`），读配对链接的输入框向键盘要的是什么（地址键盘、不自动大写、不纠错，`--check-input-type`），然后经深链配对，逐个访问五个标签页并截图，顺带拍下[手机端 App 指南](../user/android.md)用的图（`--pictures`，见[截图](contributing.md#截图)），应用的日志里有错误则失败。配对对话框说电脑没有响应时（模拟器的网络还没起来就可能这样），它关掉对话框并重新打开链接，最多三次，并在输出里说明。失败的运行会保留屏幕、应用的日志，以及设备对自己网络的说法。冷启动的测量是构件 `memory-android`；截图、应用日志、失败的证据、配对后的测量，以及五次冷启动的读数与每次完整的 `dumpsys meminfo` 报告，是构件 `phone-e2e-android`；指南用的图是构件 `phone-pictures`。
- `signing` 任务在 Windows 运行器上用发布用的脚本和它自己生成的密钥演练发布的签名（见[签名状态](#签名状态)）。`shared/scripts/sign-apk.ps1` 有改动时它也运行。

[`release.yml`](../../../.github/workflows/release.yml) 生成发布用的 APK 并签名（见[签名状态](#签名状态)）。[`memory.yml`](../../../.github/workflows/memory.yml) 在 `mygo` 或 `phone` 结束时报告 Pull Request 的内存（见[Pull Request 中的内存](memory.md#pull-request-中的内存)）。

## 维护者注记

- `vfxtool measure-apk` 查找应用显示与记录的文字，英文与简体中文都有：确认按钮的标签、电脑没有响应的提示与关闭它的按钮的标签、五个标签页的标签，以及配对时记下的那行日志（见 `shared/tooling/internal/measureapk/measureapk.go`）。拍图的那次运行还要找它打开的那个 SDK 的行（`nodejs`），以及添加插件对话框里的按钮*从来源添加*（Add from source）与*取消*（Cancel）（在紧挨着它的 `pictures.go` 里）。改应用里的这些文字时，要与这些文件在同一次提交中一起改，否则测量失败。
- `mygo/phone/versions.env` 与 `mygo/go.mod` 钉住同一个 mygo 版本。`JAVA_VERSION` 跟随 mygo 为项目运行的 Gradle：只有连同能运行新 JDK 的 Gradle 的 mygo 一起升级时，才提高它。
- 不要手改 `mygo.json` 的 `version`：`vfxtool set-version` 会与 `mygo/VERSION` 一起写它。
