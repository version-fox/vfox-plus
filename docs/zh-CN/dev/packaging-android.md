# Android 打包

> [English](../../en/dev/packaging-android.md)

本目录是安卓远端客户端：包住配对服务（`vfox-plus serve`）的 WebView，原生
Material 3 Expressive 配对流程。它按[远端协议](remote-protocol.md)编程，不直连
Go 代码。用户装 `.apk` 后与 PC 配对；见[远端与安卓](../user/remote.md)。

## 布局

| 路径 | 内容 |
| --- | --- |
| `settings.gradle.kts`、`build.gradle.kts`、`gradle.properties` | 工程配置；插件版本 pin 在此。 |
| `versions.env` | CI 读取的工具 pin（Gradle、Java、AGP、Kotlin、Material）。 |
| `app/build.gradle.kts` | 应用 ID `dev.vfox.vfoxplus`、`minSdk 26`、`targetSdk 36`；`versionName` 从 `mygo/VERSION` 读取，手机与桌面显示同一版本。CameraX 1.6.2 与 zxing core 3.5.3（均为 Apache-2.0）扫二维码。 |
| `app/src/main/` | Kotlin 源码（`dev.vfox.vfoxplus`）、布局、Material 3 Expressive 主题、英/中字符串、启动图标。 |

## App 行为

- **配对**（`PairingFragment`）是原生的：扫描终端二维码、粘贴 PC 打印的一次性
  链接，或重连上次服务器；核对 TLS 指纹、连接。shared-axis 转场，连接中波浪
  不定进度。
- **扫码**（`ScanFragment`）是 CameraX 预览加 zxing 解码：解出的链接填入配对框
  并立即连接。`CAMERA` 按非必需申请——无相机或拒绝权限时粘贴链接即可。
- **会话**（`ViewerFragment`）是 `/app` 的 WebView，会话 cookie 跨启动持久化，
  打开文件上传（供插件 `.zip`/`.lua` 文件）、下拉刷新、溢出菜单（刷新、配对新
  链接、断开）。一次性链接种下 cookie 后客户端自行进入 `/app`。WebView 内
  无法下载文件，失败任务完整日志的网页下载按钮只在浏览器里出现。
- **TLS**（`onReceivedSslError`）在信任自签名证书前弹出 SHA-256 指纹对话框——
  与 PC 终端打印的是同一个。服务端自带证书可免去警告。
- **返回键**先走 WebView 历史，再退出会话。

## 构建

需要 JDK 17 与 Android SDK（platform 36、build-tools 36）。CI 会装；本地把
`ANDROID_HOME` 指向 SDK：

```sh
cd android
gradle :app:assembleRelease
```

未签名 release APK 落在
`app/build/outputs/apk/release/app-release-unsigned.apk`（约 3.1 MB）。发布时按
版本改名：

```sh
cp app/build/outputs/apk/release/app-release-unsigned.apk \
  vfox-plus_$(tr -d '[:space:]' < ../mygo/VERSION).apk
```

装到手机（提示时允许安装未知应用），按[远端与安卓](../user/remote.md)配对。

## 签名状态

`.apk` 目前**未签名**，经未知来源提示安装。正式签名版后续提供。签名只改签名
配置；CI 上传的 `vfox-plus-android` 构件即此未签名 APK。

## 维护者注记

- `usesCleartextTraffic` 开着：可信局域网里服务端可能是纯 HTTP（PC 侧非回环
  监听一律 HTTPS，但用户可能在别处终结 TLS）。会话 cookie 都是凭证。
- 不提交 Gradle wrapper jar（仓库拒绝二进制文件）；CI 从 `versions.env` 装
  `GRADLE_VERSION`。
