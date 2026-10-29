# Windows 打包（MSIX）

> [English](../../en/dev/packaging-windows.md)

vfox+ 的 Windows 打包是 `vfxtool` 的两个命令：
`vfxtool build-msix` 把暂存的 Windows 可执行文件打包为未签名 `.msix`
（每架构一包）与 `.msixbundle`，目标 **Windows 11**；`vfxtool sign-msix` 用它自己做的证书给
它们签名并安装 bundle（见[自己签名](#自己签名)）。
`shared/packaging/windows/` 只放一个指向本页的入口。

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
（不压缩）、排序、时间戳取 `SOURCE_DATE_EPOCH`。例外是 `[Content_Types].xml`：它是由存储块
组成的 DEFLATE 条目（仍未压缩，所以没有压缩器会改变字节）。压缩包是 ZIP64，与 MakeAppx 的写法一致。
原因见[签名说明](#签名说明维护者)。

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

## 自己签名

`vfxtool sign-msix` 用它自己做的证书给 bundle 签名并安装，适合希望不带 `-AllowUnsigned`
就能安装的机器，以及测试签名路径的人。它取代了原来的 `Sign-Msix.ps1`。在解压后的
`vfox-plus-windows` 构件目录里运行，该目录含 `stage-x64/`、`stage-arm64/`、
`vfxtool.exe` 与未签名包：

```powershell
.\vfxtool.exe sign-msix
```

它依次做：

1. 把 osslsigncode 2.14 下载到 `tools/`，版本固定并核对 SHA-256，因此不需要
   Windows SDK。`--osslsigncode PATH` 改用你已有的，`--force` 重新下载。
2. 用发布者 `CN=vfox-plus`（`--publisher-cn`）而不是未签名包的 OID 把包重建到 `signed/`，
   因为已签名包的发布者必须与证书主题一致。
3. 为该主题做一张自签名代码签名证书（RSA 2048，有效期五年），给每个 `.msix` 签名，
   再像 SignTool 那样把签好名的包放回 bundle，并给 bundle 签名。
4. 在本机的 `TrustedPeople` 证书库中信任这张证书——旁加载已签名包看的就是它——
   并删掉此前某次运行添加的证书。
5. 用 `Add-AppxPackage` 安装 bundle（`--skip-install` 则在此之前停下）。

私钥只活一次运行：写到临时文件交给 osslsigncode，签完即删，从不保留。留在本机受信任
的证书因此再也签不了任何东西，也没有需要保护的密钥。代价是每次运行都做新证书，
所以此前某次签的 bundle 不能再安装：重新签名，安装新的。

信任证书与安装包需要管理员权限。在未提权的 shell 里，命令请 Windows 以管理员权限启动自己的
一个副本：你回答 UAC 提示，副本在自己的窗口里工作，结束时等你按 Enter，好让你读完它做了什么。
在已提权的 shell 里直接运行；`--no-relaunch` 让它失败而不是询问，供脚本使用。其他系统没有证书库
也没有安装，所以命令在那里只签名：传 `--skip-install` 与该系统的 `--osslsigncode`。

要撤销，删除包（见上）与证书：

```powershell
Get-ChildItem Cert:\LocalMachine\TrustedPeople |
  Where-Object Subject -eq "CN=vfox-plus" | Remove-Item
```

[`mygo.yml`](../../../.github/workflows/mygo.yml) 的 `validate-windows` 任务在 Windows Server 2025
（x64）与 Windows 11 on ARM（arm64）上运行该命令：签名、不带 `-AllowUnsigned` 安装、检查架构、
卸载、再运行一次，并检查恰有一张证书受信任。提权副本是 CI 走不到的唯一路径，因为 runner 是关了
UAC 的管理员；改动之后需要在有 UAC 的机器上跑一次。

## 签名说明（维护者）

- 已签名包的发布者必须与证书主题完全一致、不带 OID——签名因此重建包，而非直接签
  未签名包（其发布者带未签名 OID）。
- bundle 的签名只覆盖它的清单（bundle 的块映射不列别的文件），所以 bundle 里的包带着自己的签名，
  清单按偏移与大小指向它们。装着未签名包的 bundle 即使自己签了名也会被拒绝。SignTool 先给
  bundle 里的包签名再放进去；osslsigncode 一次只签一个文件，所以 `sign-msix` 先签 `.msix`，
  再用它们重新写 bundle（新的偏移、大小与块映射），然后签它（`buildmsix.Rebundle`）。
- osslsigncode 为加入签名部分会改写 `[Content_Types].xml`，它总是把新内容压缩后写入，却沿用该条目原来
  的方法。Store 的条目因此变成了损坏的条目（方法是 Store，压缩后大小小于原大小），osslsigncode 自己的
  `verify` 接受它，Windows 的部署平台不接受：`Add-AppxPackage` 以 `0x8007000D`
  “Opening the package ... failed” 失败。所以 `build-msix` 像 MakeAppx 那样，只把这一个部分写成由存储块组成的
  DEFLATE 条目；`sign-msix` 的测试也要把 osslsigncode 签出的每个条目读到末尾，只检查签名是查不出这个的。
- Windows 会拒绝已签名的纯 ZIP32 包，而同一个包不签名却能安装：`Add-AppxPackage` 以 `0x8007000B`
  “Opening the package ... failed” 失败。MakeAppx 与 SignTool 把每个包都写成 ZIP64，不管多小：经典的结尾记录里只有
  哨兵值 `0xFFFF` 与 `0xFFFFFFFF`，ZIP64 结尾记录及其定位器在它前面，中央目录里每个条目都写“由 4.5 创建”，
  而条目只需要 2.0，也没有 ZIP64 扩展字段（带扩展字段的包同样会被拒绝）。`build-msix` 写出这种形状，
  `CheckZip64` 检查它（`validate-msix` 也检查），osslsigncode 会保持它。这种形状来自
  [microsoft/msix-packaging](https://github.com/microsoft/msix-packaging) 里微软签名的测试包，以及
  [openappx](https://github.com/gianlucamazza/openappx) 在 Xbox 上的实测：签名的 ZIP32 包被拒绝，签名的 ZIP64 包能安装。
- 公开发布且不希望每台机器单独信任，可上架 Microsoft Store（由微软签名），或用
  公开受信的代码签名证书 / Azure Trusted Signing 签名；构建相同，只换 `--publisher`
  与签名步骤。
