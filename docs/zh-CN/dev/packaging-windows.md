# Windows 打包（MSIX）

> [English](../../en/dev/packaging-windows.md)

vfox+ 的 Windows 打包是 `vfxtool` 的几个命令。`vfxtool build-msix` 把暂存的 Windows 可执行文件打包为
`.msix`（每架构一包）与 `.msixbundle`，目标 **Windows 11**。发布时，CI 用 Artifact Signing 给它们签名，签名证书由 Windows
信任（见[发布的签名](#发布的签名)）：`vfxtool rebundle` 把签好名的包重新放进 bundle，
`vfxtool install-check --trusted` 则像用户那样安装它。`vfxtool sign-msix` 用它自己做的证书签名，用来测试
签名的路径（见[自己签名](#自己签名)）。`shared/packaging/windows/` 只放一个指向本页的入口。

包标识、清单、徽标与 bundle 布局由 `vfxtool build-msix`（`shared/tooling/internal/buildmsix`）生成；
此处不提交任何二进制。徽标就是应用图标，构建时由 `shared/tooling/internal/appicon` 按其 SVG 的路径绘制
（见[设计系统](design-system.md#应用图标)）。

## 产出什么

每架构（`x64` 对 AMD64，`arm64` 对 ARM64）：

- `vfox-plus_<版本>_<架构>.msix`：OPC ZIP，含 `AppxManifest.xml`、`[Content_Types].xml`、
  `AppxBlockMap.xml`、可执行文件（如 `vfox-plus.exe`）与 `Assets/` 徽标。
- `vfox-plus_<版本>.msixbundle`：含上述 `.msix` 与 `AppxMetadata/AppxBundleManifest.xml` 的常规 bundle。

标识默认 `dev.vfox.VfoxPlus`（与 Linux 桌面入口一致）。发布者由 `--publisher` 设定：发布时传入签名证书的
主题。不传则发布者为 `CN=vfox-plus` 加未签名包的 OID（`OID.2.25.311729368913984317654407730594956997722=1`），
这是 Windows 11 `-AllowUnsigned` 安装的要求；CI 构建的就是这种未签名包，用来测试。目标设备族
`Windows.Desktop`，`MinVersion 10.0.22000.0`（Windows 11）。

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

只需单架构则只传一个 `--stage-*`，bundle 含单包。输出可复现：ZIP 条目 Store（不压缩）、排序、时间戳取
`SOURCE_DATE_EPOCH`。例外是 `[Content_Types].xml`：它是由存储块组成的 DEFLATE 条目（仍未压缩，所以没有压缩器
会改变字节）。压缩包是 ZIP64，与 MakeAppx 的写法一致。原因见[签名说明](#签名说明维护者)。

## 在 Windows 11 安装

从 GitHub release 下载 `vfox-plus_<版本>.msixbundle`，在管理员 PowerShell 中安装：

```powershell
Add-AppxPackage -Path .\vfox-plus_<版本>.msixbundle
```

发布的 bundle 由 Windows 信任的证书签名，所以不需要 `-AllowUnsigned`，也不需要在机器上做任何步骤。部署平台
自动选择与本机匹配的架构切片，包为所有用户安装。卸载：

```powershell
Remove-AppxPackage -Package dev.vfox.VfoxPlus_<版本>_neutral_~_<发布者标识>
```

（用 `Get-AppxPackage -Name dev.vfox.VfoxPlus` 查看完整包名。）

签名发布之前的版本（未签名或自签名的）发布者不同，因而是另一个包，会与已签名的包并存。用上面的命令把它卸载一次。

CI 运行的 `vfox-plus-windows` 构件里是未签名的 bundle，旁边是暂存的可执行文件与 Windows 版 `vfxtool.exe`。
其中的包供测试用：只能用 `Add-AppxPackage -AllowUnsigned` 安装。

不打包直接从源码跑，在 Windows 上构建后直接运行 exe 即可。

## 发布的签名

[发布工作流](contributing.md#发布)的 `sign-windows` 任务用 [Artifact Signing](https://learn.microsoft.com/azure/artifact-signing/)
（原名 Trusted Signing）给 `.msix` 文件与 `.msixbundle` 签名，调用的是
[`azure/artifact-signing-action`](https://github.com/Azure/artifact-signing-action)。任务通过 OpenID Connect
登录 Azure，所以仓库与它的 secret 里都不存证书，也不存私钥。它依次做这些步骤：

1. 用证书配置文件的证书给每个 `.msix` 签名。
2. 把签好名的包重新放进 bundle（`vfxtool rebundle`）：签名改变了包的大小与偏移，而 bundle 记着它们。
3. 给 bundle 签名。
4. 用 `vfxtool install-check --trusted` 安装 bundle（不带 `-AllowUnsigned`），然后卸载。只有签名与发布者都成立，
   Windows 才接受这次安装；运行器的证书库不受影响。

同一个任务之后还用发布密钥签名手机端的 APK：`zipalign` 与 `apksigner` 替换它的 debug 签名（见[签名状态](packaging-android.md#签名状态)）。

服务给包加了时间戳，所以证书过期后签名依然有效；这些证书只有几天的寿命。

设置只做一次，由掌握 Azure 订阅的维护者完成：

1. 一个 Artifact Signing 账户，加一个公开信任类型的证书配置文件。服务要求的身份验证先做；证书记着它验证过的身份。
2. 为发布建一个 Microsoft Entra ID 中的身份（应用注册），并为它加一条 GitHub Actions 的联合凭据：组织 `version-fox`，
   仓库 `vfox-plus`，实体类型 **Environment**，环境 `release`。主体因此是
   `repo:version-fox/vfox-plus:environment:release`，只有 `release` 环境的任务符合。
3. 在账户上给该身份授予角色 **Artifact Signing Certificate Profile Signer**。
4. `release` 环境的 secret 是该身份的客户端 ID、租户 ID 与订阅 ID：

   ```sh
   gh secret set AZURE_CLIENT_ID --env release -R version-fox/vfox-plus
   gh secret set AZURE_TENANT_ID --env release -R version-fox/vfox-plus
   gh secret set AZURE_SUBSCRIPTION_ID --env release -R version-fox/vfox-plus
   ```

5. 仓库的变量（`gh variable set NAME -R version-fox/vfox-plus`）：
   - `ARTIFACT_SIGNING_ENDPOINT`：账户所在区域的端点，例如 `https://eus.codesigning.azure.net/`；
   - `ARTIFACT_SIGNING_ACCOUNT`：账户名；
   - `ARTIFACT_SIGNING_PROFILE`：证书配置文件名；
   - `WINDOWS_PUBLISHER`：证书的主题，须与证书完全一致，形如 `CN=..., O=..., L=..., S=..., C=...`。它为空时构建失败。
     `sign-windows` 任务会打印已签名 bundle 的主题（步骤「Show the publisher of the signed bundle」）：安装随后拒绝
     bundle 时，日志会说明该填什么。

一次发布要在 `release` 环境里等一次批准，在签名 Windows 包与 APK 之前；发布紧接着进行，不再等第二次。先跑一次**试运行**，即在 Actions 的 Release
工作流里勾选 `dry-run` 输入：它构建、签名、安装 bundle，但不发布任何东西。只有不勾选 `dry-run` 的运行才会推送、
打标签并发布。

说明：

- 第一个签名版本的发布者与之前的包不同，所以会与它们并存：见[在 Windows 11 安装](#在-windows-11-安装)。
- 浏览器下载的新文件有它自己的 SmartScreen 检查。新版本可能在一段时间里遇到警告，即使它的签名是有效的。

## 自己签名

`vfxtool sign-msix` 用它自己做的证书给 bundle 签名并安装，适合希望不带 `-AllowUnsigned` 就能安装的机器，以及测试签名路径的人。
它取 CI 运行（`mygo.yml`）的 `vfox-plus-windows` 构件：里面是未签名的包、`stage-x64/` 与 `stage-arm64/`
里的暂存可执行文件，以及 Windows 版 `vfxtool.exe`。在该目录里运行：

```powershell
.\vfxtool.exe sign-msix
```

它依次做这些：

1. 把 osslsigncode 2.16 下载到 `tools/`，版本固定并核对 SHA-256，因此不需要 Windows SDK。`--osslsigncode PATH`
   改用你已有的，`--force` 重新下载。
2. 用发布者 `CN=vfox-plus`（`--publisher-cn`）而不是未签名包的 OID 把包重建到 `signed/`，因为已签名包的发布者必须与证书主题一致。
3. 为该主题做一张自签名代码签名证书（RSA 2048，有效期五年），给每个 `.msix` 签名，再像 SignTool 那样把签好名的包放回
   bundle，并给 bundle 签名。
4. 在本机的 `TrustedPeople` 证书库中信任这张证书——旁加载已签名包看的就是它——并删掉此前某次运行添加的证书。
5. 用 `Add-AppxPackage` 安装 bundle（`--skip-install` 则在此之前停下）。

私钥只活一次运行：写到临时文件交给 osslsigncode，签完即删，从不保留。留在本机受信任的证书因此再也签不了任何东西，
也没有需要保护的密钥。代价是每次运行都做新证书，所以此前某次签的 bundle 不能再安装：重新签名，安装新的。

信任证书与安装包需要管理员权限。在未提权的 shell 里，命令请 Windows 以管理员权限启动自己的一个副本：你回答 UAC
提示，副本在自己的窗口里工作，结束时等你按 Enter，好让你读完它做了什么。在已提权的 shell 里直接运行；
`--no-relaunch` 让它失败而不是询问，供脚本使用。其他系统没有证书库也没有安装，所以命令在那里只签名：传 `--skip-install`
与该系统的 `--osslsigncode`。

要撤销，删除包（见上）与证书：

```powershell
Get-ChildItem Cert:\LocalMachine\TrustedPeople |
  Where-Object Subject -eq "CN=vfox-plus" | Remove-Item
```

[`mygo.yml`](../../../.github/workflows/mygo.yml) 的 `validate-windows` 任务在 Windows Server 2025（x64）与 Windows 11 on ARM
（arm64）上运行该命令：签名、不带 `-AllowUnsigned` 安装、检查架构、卸载、再运行一次，并检查恰有一张证书受信任。
提权副本是 CI 走不到的唯一路径，因为运行器是关了 UAC 的管理员；改动之后需要在有 UAC 的机器上跑一次。

`run-windows` 任务在同样两个运行器上运行应用本身，用 `build` 任务交叉编译出的 Windows 端到端构建：对假 vfox 的场景，
以及托盘图标（见[托盘](architecture.md#托盘)；ARM 的运行器的 shell 不接受任何图标，所以图标的位置只在 Windows Server 2025 上检查）。

## 签名说明（维护者）

- 已签名包的发布者必须与证书主题完全一致、不带 OID。发布时从一开始就用 `--publisher` 构建包；`sign-msix` 则要重建包，
  因为它的包带着未签名的 OID。
- bundle 的签名只覆盖它的清单（bundle 的块映射不列别的文件），所以 bundle 里的包带着自己的签名，清单按偏移与大小指向它们。
  装着未签名包的 bundle 即使自己签了名也会被拒绝。因此顺序是：给包签名，用它们重新生成 bundle（`vfxtool rebundle`，
  它拒绝未签名的包），然后给 bundle 签名。
- osslsigncode 为加入签名部分会改写 `[Content_Types].xml`，它总是把新内容压缩后写入，却沿用该条目原来的方法。Store 的条目因此变成了损坏的条目
  （方法是 Store，压缩后大小小于原大小），osslsigncode 自己的 `verify` 接受它，Windows 的部署平台不接受：`Add-AppxPackage`
  以 `0x8007000D`、“Opening the package ... failed” 失败。所以 `build-msix` 像 MakeAppx 那样，只把这一个部分写成由存储块组成的
  DEFLATE 条目；`sign-msix` 的测试也要把 osslsigncode 签出的每个条目读到末尾，只检查签名是查不出这个的。
- Windows 会拒绝已签名的纯 ZIP32 包，而同一个包不签名却能安装：`Add-AppxPackage` 以 `0x8007000B`、“Opening the package ... failed”
  失败。MakeAppx 与 SignTool 把每个包都写成 ZIP64，不管多小：经典的结尾记录里只有哨兵值 `0xFFFF` 与 `0xFFFFFFFF`，ZIP64 结尾记录
  及其定位器在它前面，中央目录里每个条目都写“由 4.5 创建”，而条目只需要 2.0，也没有 ZIP64 扩展字段（带扩展字段的包同样会被拒绝）。
  `build-msix` 写出这种形状，`CheckZip64` 检查它（`validate-msix` 也检查）。这种形状来自
  [microsoft/msix-packaging](https://github.com/microsoft/msix-packaging) 里微软签名的测试包，以及
  [openappx](https://github.com/gianlucamazza/openappx) 在 Xbox 上的实测：签名的 ZIP32 包被拒绝，签名的 ZIP64 包能安装。
