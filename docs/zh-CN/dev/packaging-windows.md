# Windows 打包（MSIX）

> [English](../../en/dev/packaging-windows.md)

本目录是 vfox+ Windows 打包的输入：
`vfxtool build-msix` 把暂存的 Windows 可执行文件打包为未签名 `.msix`
（每架构一包）与 `.msixbundle`，目标 **Windows 11**。

| 文件 | 用途 |
| --- | --- |
| `README.md` | 本文档（英文）。 |
| `README.zh-CN.md` | 中文译文。 |
| [`Sign-Msix.ps1`(../../../shared/packaging/windows/Sign-Msix.ps1) | 维护者本地重建与签名包的辅助脚本；用户不需要使用（见下）。 |

包标识、清单、徽标与 bundle 布局由 `vfxtool build-msix`
（`shared/tooling/internal/buildmsix`）生成；此处不提交任何二进制。徽标就是应用图标，
构建时由 `shared/tooling/internal/appicon` 按其 SVG 的路径绘制（见
[设计系统](design-system.md#应用图标)）。

## 产出什么

每架构（`x64` 对 AMD64、`arm64` 对 ARM64）：

- `vfox-plus_<版本>_<架构>.msix`：未签名 OPC ZIP，含 `AppxManifest.xml`、
  `[Content_Types].xml`、`AppxBlockMap.xml`、可执行文件（如 `vfox-plus.exe`）与
  `Assets/` 徽标。
- `vfox-plus_<版本>.msixbundle`：含上述 `.msix` 与
  `AppxMetadata/AppxBundleManifest.xml` 的常规 bundle。

标识默认 `dev.vfox.VfoxPlus`（与 Linux 桌面入口一致），发布者默认 `CN=vfox-plus`
加未签名包 OID（`OID.2.25.311729368913984317654407730594956997722=1`），Windows 11
`-AllowUnsigned` 安装要求。目标设备族 `Windows.Desktop`，
`MinVersion 10.0.22000.0`（Windows 11）。

## 构建

任意系统交叉编译（mygo 无需 C 编译器），再打包：

```sh
vfxtool build --goos windows --goarch amd64 --root mygo stage-x64/vfox-plus.exe
vfxtool build --goos windows --goarch arm64 --root mygo stage-arm64/vfox-plus.exe
SOURCE_DATE_EPOCH="$(git log -1 --format=%ct)" vfxtool build-msix \
  --package vfox-plus --version "$(tr -d '[:space:]' < mygo/VERSION)" \
  --stage-x64 stage-x64 --stage-arm64 stage-arm64 \
  --executable vfox-plus.exe dist
```

只需单架构则只传一个 `--stage-*`，bundle 含单包。输出可复现：ZIP 条目 Store
（不压缩）、排序、时间戳取 `SOURCE_DATE_EPOCH`。

## 在 Windows 11 安装

解压 `vfox-plus-windows` 构件（内含未签名包、暂存可执行文件与 Windows 版
`vfxtool.exe`），在管理员 PowerShell 安装 bundle：

```powershell
Add-AppxPackage -Path .\vfox-plus_0.1.0.msixbundle -AllowUnsigned
```

部署平台自动选择与本机匹配的架构切片，包为所有用户安装。卸载：

```powershell
Remove-AppxPackage -Package dev.vfox.VfoxPlus_<版本>_neutral_~_<发布者标识>
```

（用 `Get-AppxPackage -Name dev.vfox.VfoxPlus` 查看完整包名。）

不打包直接从源码跑，在 Windows 上构建后直接运行 exe 即可。

## 签名状态

`.msix` 与 `.msixbundle` 目前**未签名**，按上用 `Add-AppxPackage -AllowUnsigned`
安装即可，无需自签名。日后会提供官方签名的安装包。

## 签名说明（维护者）

- 已签名包的发布者必须与证书主题完全一致、不带 OID——签名因此重建包，而非直接签
  未签名包（其发布者带未签名 OID）。
- 公开发布且不希望每台机器单独信任，可上架 Microsoft Store（由微软签名），或用
  公开受信的代码签名证书 / Azure Trusted Signing 签名；构建相同，只换 `--publisher`
  与签名步骤。
