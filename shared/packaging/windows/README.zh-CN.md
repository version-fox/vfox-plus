# Windows 打包（MSIX）

> [English](README.md)

本目录是 vfox+ Windows 打包的输入：
`vfxtool build-msix` 把暂存的 Windows 可执行文件打包为未签名 `.msix`
（每架构一包）与 `.msixbundle`，目标 **Windows 11**。

| 文件 | 用途 |
| --- | --- |
| `README.md` | 本文档（英文）。 |
| `README.zh-CN.md` | 中文译文。 |
| [`Sign-Msix.ps1`](Sign-Msix.ps1) | Windows 11 一键自签名安装：创建自签名代码签名证书、用匹配的发布者重建包、用 osslsigncode 签名并安装 bundle。 |

| 文件 | 用途 |
| --- | --- |
| `README.md` | 本文档（英文）。 |
| `README.zh-CN.md` | 中文译文。 |

包标识、清单、徽标与 bundle 布局由 `vfxtool build-msix`
（`shared/tooling/internal/buildmsix`）生成；此处不提交任何二进制。

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

不需要第三方证书。解压 `vfox-plus-windows` 构件（内含未签名包、松散可执行文件、
Windows 版 `vfxtool.exe` 与 [`Sign-Msix.ps1`](Sign-Msix.ps1)），在 PowerShell 跑脚本。
脚本自动提权，一次做完所有事：创建（或复用）自签名代码签名证书、用匹配的发布者
重建 `.msix`/`.msixbundle`、用 osslsigncode 签名（自动下载，无需 Windows SDK）、
信任证书并安装 bundle。

```powershell
powershell -ExecutionPolicy Bypass -File Sign-Msix.ps1
```

重复跑安全：复用已有证书并重新安装。`Sign-Msix.ps1 -SkipInstall` 只重建签名不安装；
`-PublisherCN <名称>` 换发布者名；`-Force` 生成新证书。私钥只存在你自己的证书存储中，
只有你能签名。

### 手动安装（未签名）

不签名时，未签名包用管理员 PowerShell `Add-AppxPackage -AllowUnsigned` 安装
（可执行内容为所有用户安装）：

```powershell
Add-AppxPackage -Path .\vfox-plus_0.1.0.msixbundle -AllowUnsigned
```

不打包直接从源码跑，在 Windows 上构建后直接运行 exe 即可。

## 签名说明（维护者）

- 已签名包的发布者必须与证书主题完全一致、不带 OID——脚本因此重建包，而非直接签
  未签名包（其发布者带未签名 OID）。
- 自签名证书只在信任它的机器上有效，脚本恰好在本地建立该信任。公开发布且不希望每台
  机器单独信任，可上架 Microsoft Store（由微软签名），或用公开受信的代码签名证书 /
  Azure Trusted Signing 签名；构建相同，只换 `--publisher` 与签名步骤。
