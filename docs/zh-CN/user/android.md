# 手机端 App

> [English](../../en/user/android.md)

手机端 App 用手机操控 PC 上的 vfox-plus。它是安卓原生应用，在手机上名为 vfox+：它不是窗口里的网页，
而是自己绘制界面，并直接调用 PC 上的 vfox-plus，所以 PC 上要开启*其他设备*。界面里指向其他网站的链接
（如插件主页）会在手机的浏览器里打开。

## 安装

按[安装](install.md#安卓)所说，安装最新 GitHub release 中的 `.apk`。手机端 App 的包名是
`dev.vfox.vfoxplus`，与之前在窗口里显示网页界面的安卓客户端相同。如果安卓拒绝在那个客户端上覆盖安装，
先卸载它；这会忘记配对，之后需要重新配对。

## 配对手机

手机需要能到达 PC：同一 Wi-Fi，或其他网络路由。

1. 在 PC 上打开 vfox-plus，进入*设置 → 其他设备*，打开*其他设备*，并在随后的询问中点**允许**（Allow）。
2. 在 PC 的防火墙中放行其他设备的端口（默认 8369）。
3. 在 PC 上点*配对手机*旁的**显示二维码**（Show code）。PC 会显示一个二维码和它的链接；点**复制链接**
   （Copy link）即可复制链接。
4. 在手机上打开 vfox+。配对界面有三种获取链接的方式，并排放着。

   **拍照**（Photo）用相机拍下二维码。安卓会请求一次相机权限。如果拒绝，App 会提示*相机权限未开启*，
   并提供**打开设置**（Open settings），或改为选择图片。把相机对准二维码，点**拍摄**（Take the photo）。

   **相册**（Picture）从手机的相册中选一张二维码图片。只能读取 PNG 与 JPEG 图片。

   **粘贴**（Paste）粘贴你复制的链接，然后点**连接**（Connect）。也可以把链接输入或粘贴到*配对链接*
   （Pairing link）输入框里，再点**连接**。

   拍照或选图之后不需要点**连接**：App 读到二维码后就立即检查 PC。

5. App 检查 PC 后显示*信任这台电脑吗？*（Trust this computer?），并给出 PC 证书的 SHA-256 指纹。把它与
   PC 上*设置 → 其他设备*里、点过**显示二维码**之后*链接与证书*（Link and certificate）下显示的指纹核对。
6. 只有两个指纹一致时，才点**信任并配对**（Trust and pair）。App 会保存 PC 的地址与指纹，并固定它的证书：
   此后只接受这个证书，拒绝任何其他证书。

截图用的是英文界面。

| 配对界面 | 询问是否信任这台电脑 | 配对之后的主页 |
| --- | --- | --- |
| ![手机端 App 的配对界面（英文界面）：配对链接输入框，以及 Paste、Photo、Picture 三个按钮](https://iili.io/oFvjPwJ.png) | ![询问是否信任这台电脑的对话框：PC 证书的指纹，以及 Cancel 与 Trust and pair 两个按钮](https://iili.io/oFvw9us.png) | ![手机端 App 的主页：vfox 的版本、在用的 SDK、健康检查与终端里要用的命令](https://iili.io/oFvwoAu.png) |

链接一次性有效，五分钟后过期。如果 App 提示链接无效、已使用或已过期，请在 PC 上重新生成。

深链 `vfoxplus://pair?link=…` 会打开配对界面并填入链接，同样去检查 PC。它不会自行配对：仍需核对指纹，
并点**信任并配对**。

## 五个标签页

底部栏有五个标签页：**主页**（Home）、**SDK**（SDKs）、**插件**（Plugins）、**任务**（Tasks）与
**设置**（Settings）。它们显示的内容与桌面应用相同，差别见[与桌面应用的区别](#与桌面应用的区别)。

| SDK 标签页 | 一个 SDK 的页面 | 插件标签页 |
| --- | --- | --- |
| ![手机端 App 的 SDK 标签页：搜索框，以及已安装的 SDK flutter、java、maven 与 nodejs](https://iili.io/oFvwztj.png) | ![SDK nodejs 的页面：Installed、Available、Options 三个页签，以及它已安装的版本](https://iili.io/oFvwTox.png) | ![手机端 App 的插件标签页：Update all 与 Add from source 两个操作、搜索框、All、Official、Community 三个筛选，以及插件仓库里的插件](https://iili.io/oFvwcHF.png) |

| 添加插件 | 任务标签页 | 设置标签页 |
| --- | --- | --- |
| ![添加插件的对话框：Source URL 与 Alias 两个输入框，以及 Upload & add 下的 Choose a file 按钮](https://iili.io/oFvwMVR.png) | ![手机端 App 的任务标签页，还没有运行过任务](https://iili.io/oFvwNHX.png) | ![手机端 App 的设置标签页：This phone（有 Pair a new link 与 Disconnect）以及 Connections 一节](https://iili.io/oFvwsRe.png) |

**主页**显示 vfox 的版本与刷新按钮、在用的 SDK，以及健康检查的前五条发现；发现提供**纳入**（Adopt）或
**重装**（Reinstall）时，会显示相应按钮。下面是终端里要用的命令 `eval "$(vfox activate bash)"`，
旁边有一个**复制**（Copy）按钮。

**SDK** 列出你已安装的插件及其当前版本，并可搜索。点开一个插件，即可看到它的详情，其中有三个页签：
**已安装**（Installed），标出版本数；**可安装**（Available）；以及**选项**（Options），只有插件有选项时才出现。

在**已安装**里，每个版本都可以**设为全局**（Use globally）、**重装**（Reinstall）、**命名别名**（Name alias）
或**卸载**（Uninstall）。在**可安装**里可以搜索版本，点**安装**（Install）或**安装并使用**（Install and use）。
**按名称安装…**（Install by name…）安装列表里没有的版本；**探测并链接**（Detect and link）与**链接已有…**
（Link existing…）把 PC 上已有的 SDK 纳入 vfox-plus。列表上方的筛选 Chip 可以收窄列表。在**选项**里有
**插件变量**（Plugin variables，每项都有**更改**（Change））、**版本文件**（Version files）、**别名**
（Aliases，每个都可以**删除别名**（Remove alias））、**须知**（Notes）与**文档**（Documentation）。

**插件**列出插件仓库。可以搜索，用**全部**（All）、**官方**（Official）或**社区**（Community）筛选，并在某个
插件上点**安装**（Install）。每张卡片上的**主页**（Homepage）按钮会在浏览器里打开插件主页。**全部更新**
（Update all）更新已安装的插件；**从来源添加**（Add from source）按来源地址（*Source URL*，来源地址）添加
插件，别名可选。同一个对话框里的**上传并添加**（Upload & add）下面，**选择文件**（Choose a file）会打开手机的文件选择器：
选一个不超过 32 MiB 的 `.zip` 压缩包或 `.lua` 脚本，App 把它发给 PC，由 PC 按它添加插件，别名用你填写的那个。
别的文件会被立即拒绝。

**任务**列出 PC 上以任务运行的操作及其状态。点开一条任务即可看到它的输出，然后用**取消**（Cancel）、
**重试**（Retry）、**完整日志**（Full log）或**复制日志**（Copy log）。**清除已完成**（Clear finished）会忘掉
已完成的任务。标签页上的角标数字是排队中与运行中的任务数。当一条排队或运行中的任务在你看别的标签页时
结束，会有消息提示，并带**查看**（View）按钮，点它打开任务页。

**设置**从上到下有以下几节：

- **这台手机**（This phone）：*已连接到*（Connected to）PC 的地址，有**配对新链接**（Pair a new link）
  与**断开连接**（Disconnect）。
- **连接**（Connections）：**其他设备**（Other devices）开关。打开时会询问*允许其他设备？*（Allow other
  devices?），点**允许**（Allow）；关闭时会询问*停止其他设备的访问？*（Stop other devices?），点**停止**
  （Stop），这台手机也会断开。然后是**端口**（Port）、**配对手机**（Pair a phone）及其**显示二维码**
  （Show code，为另一台设备显示配对链接的二维码）、**轻量模式**（Lightweight mode），以及**退出 vfox+**
  （Quit vfox+）。退出前会要求用**退出**（Quit）确认；它会结束 PC 上的 vfox-plus，断开所有设备，并取消
  正在运行的任务。
- **应用**（Application）：**vfox 可执行文件**（vfox executable）、**应用语言**（Application language，
  由 PC 保存）与**界面语言**（Interface language，只改变这台手机的界面）。两者都有*跟随系统*（system）、
  *English* 与*简体中文*。然后是**环境变量**（Environment variables），每行一个 `KEY=value`，用**保存**
  （Save）保存。
- **vfox 配置**（vfox config）：PC 上 vfox 的设置，点一项即可修改。

App 跟随手机的深色主题；文字大小与动效跟随系统设置。

| 深色主题下的配对界面 | 深色主题下的主页 |
| --- | --- |
| ![深色主题下手机端 App 的配对界面](https://iili.io/oFvjbat.png) | ![深色主题下手机端 App 的主页](https://iili.io/oFvN9UP.png) |

## 手机无法连到 PC 时

如果手机与 PC 断开，App 会继续尝试。PC 仍无响应时，标签页上方会出现一条提示：“电脑没有响应。请检查网络，
并确认电脑上的 vfox+ 正在运行。”并提供**重试**（Retry）。

会话结束后，配对界面会显示*需要重新配对*（This phone must pair again）。这发生在 PC 上的 vfox-plus 重新启动
之后、关闭*其他设备*之后，以及 24 小时的会话到期之后。在 PC 上复制新链接，然后在配对界面点**粘贴**
（Paste），再点**连接**（Connect）。

如果 PC 的证书变了，App 会拒绝连接，并显示*电脑的证书已经改变*（The computer's certificate changed）。
重新配对之前，按第 5 步在 PC 的设置里核对指纹。

## 与桌面应用的区别

- **相机**拍下二维码，不是实时扫描。
- **版本**只能设为全局（**设为全局**）。手机不能把版本固定到项目，也不能复制安装路径。
- **证书、公网地址与公网隧道**只在 PC 上设置。
- **断开连接**只在这台手机上忘记 PC。PC 上的会话会保持到 24 小时后过期，PC 不会撤销它。
