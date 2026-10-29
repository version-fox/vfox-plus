# Android 打包

> [English](../../en/dev/packaging-android.md)

本目录是安卓远端客户端：包住配对服务（`vfox-plus serve`）的 WebView，原生
Material 3 Expressive 配对流程。它按[远端协议](remote-protocol.md)编程，不直连
Go 代码。用户装 `.apk` 后与 PC 配对；见[远端与安卓](../user/remote.md)。

## 布局

| 路径 | 内容 |
| --- | --- |
| `settings.gradle.kts`、`build.gradle.kts`、`gradle.properties` | 工程配置；插件版本 pin 在此。 |
| `gradlew`、`gradlew.bat`、`gradlew.ps1`、`gradle/wrapper/` | Gradle wrapper（[Gradle Wrapper Neo](https://github.com/Glavo/gradle-wrapper-neo)）；`gradle-wrapper.properties` 连同 SHA-256 pin 住 Gradle。 |
| `versions.env` | CI 读取的工具 pin（Java、AGP、Kotlin、Material）。 |
| `app/build.gradle.kts` | 应用 ID `dev.vfox.vfoxplus`、`minSdk 26`、`targetSdk 36`；`versionName` 从 `mygo/VERSION` 读取，手机与桌面显示同一版本。CameraX 1.6.2 与 zxing core 3.5.3（均为 Apache-2.0）扫二维码。 |
| `app/src/main/` | Kotlin 源码（`dev.vfox.vfoxplus`）、布局、Material 3 Expressive 主题、英/中字符串、启动图标（自适应，含供主题图标用的单色层；见[设计系统](design-system.md#应用图标)）。 |

## App 行为

- **配对**（`PairingFragment`）是原生的：扫描终端二维码、粘贴 PC 打印的一次性
  链接，或重连上次服务器；核对 TLS 指纹、连接。shared-axis 转场，连接中波浪
  不定进度。
- **扫码**（`ScanFragment`）是 CameraX 预览加 zxing 解码：解出的链接填入配对框
  并立即连接。`CAMERA` 按非必需申请——无相机或拒绝权限时粘贴链接即可。
- **会话**（`ViewerFragment`）是 `/app` 的 WebView，会话 cookie 跨启动持久化，
  打开文件上传（供插件 `.zip`/`.lua` 文件）、下拉刷新、溢出菜单（刷新、配对新
  链接、断开）。一次性链接种下 cookie 后客户端自行进入 `/app`。WebView 内
  无法下载文件，失败任务完整日志的网页下载按钮只在浏览器里出现。窗口始终停在已配对的服务器上：页面链到其它
  站点的地址（插件主页、文档）在手机的浏览器里打开（`shouldOverrideUrlLoading`；页面为它们点击一个链接，
  而不调用 `window.open`，后者 WebView 不会上报）。
- **TLS**（`onReceivedSslError`）在信任自签名证书前弹出 SHA-256 指纹对话框——
  与 PC 终端打印的是同一个。服务端自带证书可免去警告。
- **返回键**先走 WebView 历史，再退出会话。

## 构建

需要 JDK 17 与 Android SDK（platform 36、build-tools 35.0.0；后者是所 pin AGP
的默认值）。CI 会装；本地把 `ANDROID_HOME` 指向 SDK（或把 `sdk.dir` 写进被 git
忽略的 `local.properties`）；缺少的包要先接受其许可（`sdkmanager --licenses`）
Gradle 才会安装。wrapper 会下载 `gradle/wrapper/gradle-wrapper.properties` pin
住的 Gradle，不必另装 Gradle：

```sh
cd android
./gradlew :app:assembleRelease
```

构建不需要 KVM；在模拟器上运行 APK 才需要（见[模拟器与 KVM](environment.md#模拟器与-kvm)）。首次运行要用 JDK 编译 `GradleWrapperNeo.java`，并下载 Gradle 与依赖，需要几分钟。
`services.gradle.org` 慢的地方，请给你的用户而不是给项目配置下载镜像；见 Neo
README 的 “Configure download mirrors”。

release APK 落在 `app/build/outputs/apk/release/app-release.apk`（约 3.2 MB），
用 debug 密钥签名（见[签名状态](#签名状态)）。发布时按版本改名：

```sh
cp app/build/outputs/apk/release/app-release.apk \
  vfox-plus_$(tr -d '[:space:]' < ../mygo/VERSION).apk
```

装到手机（提示时允许安装未知应用），按[远端与安卓](../user/remote.md)配对。

## 签名状态

安卓不安装没有签名的 APK（未签名的会以 `INSTALL_PARSE_FAILED_NO_CERTIFICATES`
失败），所以 release 构建用 **debug 密钥**签名；Gradle 首次构建时在
`~/.android/debug.keystore`（或 `$ANDROID_USER_HOME`）创建它。每台构建机器各有
一把，CI 的随 runner 丢弃：不同构建的 APK 不能互相覆盖更新（要先卸载，配对也随
之丢失），签名也不表明构建者是谁。正式签名的 APK 后续提供；它只改
`app/build.gradle.kts` 里 release 构建的 `signingConfig`。

## 维护者注记

- `usesCleartextTraffic` 开着：可信局域网里服务端可能是纯 HTTP（PC 侧非回环
  监听一律 HTTPS，但用户可能在别处终结 TLS）。会话 cookie 都是凭证。
- wrapper 是 [Gradle Wrapper Neo](https://github.com/Glavo/gradle-wrapper-neo)
  0.2.0（Apache-2.0）：官方的 `gradle-wrapper.jar` 是二进制，仓库拒绝二进制文件，
  Neo 用一个 Java 源文件做同样的事，由 `gradlew` 首次使用时编译。`gradlew`、
  `gradlew.bat`、`gradlew.ps1` 与 `GradleWrapperNeo.java` 保持发布版原样（所以
  `.licenserc.yaml` 跳过它们的文件头）；升级时按其 README 从新发布版下载这四个
  文件。`.gitattributes` 让 `*.bat` 保持 CRLF，`gradlew.bat` 在 Windows 上需要。
- 换 Gradle 要改 `gradle/wrapper/gradle-wrapper.properties` 里的
  `distributionUrl` 与 `distributionSha256Sum`（校验值：
  <https://gradle.org/release-checksums/>）。pin 在 9.5.1 是因为 AGP 8.13.2 在
  Gradle 9.8.0 上失败；AGP 与 Gradle 一起升。
