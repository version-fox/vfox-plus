# The phone app

> [简体中文](../../zh-CN/user/android.md)

The phone app drives the vfox-plus on your PC from your phone. It is a native
Android app, named vfox+ on the phone: it is not a web page in a window. It draws
its own screens and calls the vfox-plus on your PC directly, so *Other devices*
must be turned on there. Links to other sites, such as the homepage of a plugin,
open in the browser of the phone.

## Install

Install the `.apk` of the latest GitHub release, as [Install](install.md#android)
describes. The phone app has the package name `dev.vfox.vfoxplus`, the same as the
earlier Android client, which showed the web interface in a window. If Android
refuses to install over that client, uninstall it first; that forgets the pairing,
so pair the phone again afterwards.

## Pair the phone

The phone must reach the PC, on the same Wi-Fi or on another network route.

1. On the PC, open vfox-plus, go to *Settings → Other devices* and turn on
   *Other devices*. Confirm with **Allow** (允许).
2. Allow the port of other devices (8369 by default) in the firewall of the PC.
3. On the PC, tap **Show code** (显示二维码) next to *Pair a phone*. The PC shows a
   QR code and its link; **Copy link** (复制链接) copies the link.
4. On the phone, open vfox+. The pairing page has three ways to get the link, side by
   side.

   **Photo** (拍照) takes a photo of the code with the camera. Android asks for the
   camera once. If you refuse, the app says *Camera access is off* and offers
   **Open settings** (打开设置), or a picture instead. Point the camera at the code and
   tap **Take the photo** (拍摄).

   **Picture** (相册) chooses a picture of the code from the library of the phone.
   Only PNG and JPEG pictures can be read.

   **Paste** (粘贴) pastes the link that you copied. Then tap **Connect** (连接). You can
   also type or paste the link into the field *Pairing link* (配对链接) and tap
   **Connect**.

   A photo or a picture needs no **Connect**: the app checks the PC as soon as it reads
   the code.

5. The app checks the PC and shows *Trust this computer?* (信任这台电脑吗？) with the
   SHA-256 fingerprint of the certificate of the PC. Compare it with the fingerprint
   that the PC shows in *Settings → Other devices*, under *Link and certificate*
   (链接与证书), after **Show code**.
6. Only when the two fingerprints match, tap **Trust and pair** (信任并配对). The app
   saves the address and the fingerprint of the PC and pins its certificate: from then
   on it accepts that certificate only, and refuses any other one.

| The pairing page | The question that asks to trust the computer | The home page, once the phone is paired |
| --- | --- | --- |
| ![The pairing page of the phone app, with the field for the pairing link and the buttons Paste, Photo and Picture](https://iili.io/oFvjPwJ.png) | ![The question Trust this computer?, with the fingerprint of the certificate of the PC and the buttons Cancel and Trust and pair](https://iili.io/oFvw9us.png) | ![The home page of the phone app: the version of vfox, the SDKs in use, the health check and the command for the shell](https://iili.io/oFvwoAu.png) |

The link works once and expires after five minutes. If the app says that the link is
invalid, already used or expired, make a new one on the PC.

The deep link `vfoxplus://pair?link=…` opens the pairing page with the link filled in,
and checks the PC in the same way. It never pairs by itself: you still compare the
fingerprint and tap **Trust and pair**.

## The five tabs

The bottom bar has five tabs: **Home** (主页), **SDKs** (SDK), **Plugins** (插件),
**Tasks** (任务) and **Settings** (设置). They show what the desktop application
shows, with the differences that [What differs from the desktop application](#what-differs-from-the-desktop-application)
lists.

| SDKs | The page of an SDK | Plugins |
| --- | --- | --- |
| ![The SDKs tab of the phone app, with a search field and the installed SDKs flutter, java, maven and nodejs](https://iili.io/oFvwztj.png) | ![The page of the SDK nodejs, with the tabs Installed, Available and Options, and its installed versions](https://iili.io/oFvwTox.png) | ![The Plugins tab of the phone app, with the actions Update all and Add from source, the search field, the filters All, Official and Community, and the plugins of the registry](https://iili.io/oFvwcHF.png) |

| Adding a plugin | Tasks | Settings |
| --- | --- | --- |
| ![The dialog that adds a plugin, with the fields Source URL and Alias, and the button Choose a file under Upload and add](https://iili.io/oFvwMVR.png) | ![The Tasks tab of the phone app, before any task has run](https://iili.io/oFvwNHX.png) | ![The Settings tab of the phone app: This phone, with Pair a new link and Disconnect, and the section Connections](https://iili.io/oFvwsRe.png) |

**Home** shows the version of vfox with a refresh button, the SDKs in use, and the
health check: its first five findings, with **Adopt** (纳入) or **Reinstall** (重装)
where a finding offers them. Below them is the command for the terminal,
`eval "$(vfox activate bash)"`, with a **Copy** (复制) button.

**SDKs** lists the plugins that you have installed, with their current versions, and
searches them. Open a plugin to see its detail, which has three tabs: **Installed**
(已安装), which counts its versions; **Available** (可安装); and **Options** (选项),
which appears when the plugin has options.

Under **Installed**, each version offers **Use globally** (设为全局), **Reinstall**
(重装), **Name alias** (命名别名) and **Uninstall** (卸载). Under **Available**, you
search the versions and tap **Install** (安装) or **Install and use** (安装并使用).
**Install by name…** (按名称安装…) installs a version that the list does not show, and
**Detect and link** (探测并链接) and **Link existing…** (链接已有…) take the SDKs that are
already on the PC under vfox-plus. Filter chips above the list narrow it. Under
**Options**, you find **Plugin variables** (插件变量), each with **Change** (更改),
**Version files** (版本文件), **Aliases** (别名), each with **Remove alias** (删除别名),
**Notes** (须知) and **Documentation** (文档).

**Plugins** lists the plugin registry. Search it, filter it with **All** (全部),
**Official** (官方) or **Community** (社区), and tap **Install** (安装) on a plugin. Each
card has a **Homepage** (主页) button, which opens the homepage in the browser.
**Update all** (全部更新) updates the installed plugins, and **Add from source**
(从来源添加) adds a plugin from a source URL (*Source URL*, 来源地址), with an optional
alias. Under **Upload & add** (上传并添加) in the same dialog, **Choose a file** (选择文件)
opens the file picker of the phone: choose a `.zip` archive or a `.lua` script of up to 32 MiB,
and the app sends it to the PC, which adds the plugin from it, with the alias that you typed.
Any other file is refused at once.

**Tasks** lists the changes that the PC runs as tasks, with their states. Tap a task to
open its output, then use **Cancel** (取消), **Retry** (重试), **Full log** (完整日志) or
**Copy log** (复制日志). **Clear finished** (清除已完成) forgets the finished tasks. The
badge on the tab counts the tasks that are queued or running. When a task that was queued
or running finishes while you look at another tab, a message says so, with **View**
(查看), which opens Tasks.

**Settings** has these sections, from the top:

- **This phone** (这台手机): *Connected to* (已连接到) the address of the PC, with
  **Pair a new link** (配对新链接) and **Disconnect** (断开连接).
- **Connections** (连接): the **Other devices** (其他设备) switch. Turning it on asks
  *Allow other devices?* (允许其他设备？) with **Allow** (允许). Turning it off asks
  *Stop other devices?* (停止其他设备的访问？) with **Stop** (停止), and this phone loses its
  connection too. Then come **Port** (端口), **Pair a phone** (配对手机) with **Show code**
  (显示二维码), which shows the QR code of a pairing link for another device,
  **Lightweight mode** (轻量模式), and **Quit vfox+** (退出 vfox+). Quitting asks for a
  confirmation with **Quit** (退出); it ends vfox-plus on the PC, disconnects every device,
  and cancels the tasks that run.
- **Application** (应用): **vfox executable** (vfox 可执行文件), **Application language**
  (应用语言), which the PC keeps, and **Interface language** (界面语言), which changes only
  the screens of this phone. Both offer *system* (跟随系统), *English* and *简体中文*. Then
  come **Environment variables** (环境变量), one `KEY=value` per line, saved with **Save**
  (保存).
- **vfox config** (vfox 配置): the settings of vfox on the PC; tap one to change it.

The app follows the dark theme of the phone, and its text size and motion follow the
system's settings.

| The pairing page in the dark theme | The home page in the dark theme |
| --- | --- |
| ![The pairing page of the phone app in the dark theme](https://iili.io/oFvjbat.png) | ![The home page of the phone app in the dark theme](https://iili.io/oFvN9UP.png) |

## When the phone cannot reach the PC

If the phone loses the PC, the app keeps trying. While the PC does not answer, a strip
above the tabs says *The computer did not answer. Check the network, and that vfox+ runs
on the computer.* and offers **Retry** (重试).

If the session has ended, the pairing page shows *This phone must pair again* (需要重新配对).
That happens after vfox-plus on the PC restarts, after *Other devices* is turned off, and
when the session of 24 hours expires. Copy the new link on the PC, then on the pairing page
tap **Paste** (粘贴) and **Connect** (连接).

If the certificate of the PC has changed, the app refuses to connect and shows *The
computer's certificate changed* (电脑的证书已经改变). Before you pair again, compare the
fingerprint in the settings of the PC, as in step 5.

## What differs from the desktop application

- **The camera** takes a photo of the code. It does not scan the code live.
- **Versions** are set globally only, with **Use globally**. The phone cannot pin a
  version to a project or copy an install path.
- **The certificate, the public address and the public tunnel** are set on the PC only.
- **Disconnect** forgets the PC on this phone only. The PC keeps the session until it
  expires, after 24 hours; the PC does not revoke it.
