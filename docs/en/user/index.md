# vfox-plus user guide

> [简体中文](../../zh-CN/user/index.md)

vfox-plus is a graphical front end for [vfox](https://vfox.dev), the
cross-platform version manager. It follows the
[Material 3 Expressive](https://m3.material.io/) design language, speaks
English and Simplified Chinese, and follows your desktop theme and accent
colour.

![vfox+ home page](https://iili.io/nD3UB0x.png)

| Installed versions of an SDK | Available versions, dark theme |
| --- | --- |
| ![Installed versions of nodejs](https://iili.io/nD3UCUQ.png) | ![Available versions of nodejs, dark theme](https://iili.io/nD3UoJV.png) |

| An installation in progress | Tasks, which can be followed and cancelled |
| --- | --- |
| ![Installing a version of nodejs](https://iili.io/nD3UzOP.png) | ![Tasks](https://iili.io/nD3UuzF.png) |

| The plugin registry | Settings in Simplified Chinese |
| --- | --- |
| ![Plugins](https://iili.io/nD3UAWg.png) | ![Settings](https://iili.io/nD3URsa.png) |

| Home in a compact window | An SDK in a compact window |
| --- | --- |
| ![Home in a compact window](https://iili.io/nD3UY0v.png) | ![An SDK in a compact window](https://iili.io/nD3UEbt.png) |

| The distribution of Java to list | Temurin, with the chip for long-term support |
| --- | --- |
| ![The distributions of Java](https://iili.io/nD3Uj1f.png) | ![Temurin releases with long-term support](https://iili.io/nD3UOdl.png) |

| Flutter, with its channels and architectures | The options of Node.js |
| --- | --- |
| ![Flutter versions of the stable channel](https://iili.io/nD3UsdQ.png) | ![The options of Node.js](https://iili.io/nD3gFpI.png) |

| The web interface, on a phone | Java in the web interface |
| --- | --- |
| ![The home page of the web interface on a phone](https://iili.io/nD3rdMu.png) | ![Temurin releases with long-term support in the web interface](https://iili.io/nD3rnwP.png) |

## What it does

- Shows your installed SDKs, the versions in use and statistics at a glance.
- Installs, uninstalls and selects SDK versions, globally or for a project,
  with live download progress.
- Browses the vfox plugin registry, adds plugins (also from a URL or a file),
  updates and removes them.
- Runs every change as a task that can be followed, cancelled and retried.
- Edits the vfox configuration (proxy, registry, storage, legacy version
  files, caches, timeouts) and extra environment variables such as download
  mirrors.
- Offers what vfox-java, vfox-nodejs and vfox-flutter provide beyond a plain
  list: the distribution of Java to list (31 of them), filter chips (LTS,
  JavaFX, channels, architectures), installation by name and download mirrors,
  see [Java, Node.js and Flutter](using.md#java-nodejs-and-flutter).
- Serves the same interface to a browser, and, if you allow it, to phones and other
  computers on your network over HTTPS; can keep running without its window, see
  [The web interface and other devices](remote.md).

vfox-plus drives the vfox command line tool, which must be installed
separately (see the [vfox installation guide](https://vfox.dev/guides/quick-start.html)).

## Guides

| Guide | Content |
| --- | --- |
| [Install](install.md) | Get vfox-plus on Linux, Windows 11 or Android. |
| [Use](using.md) | The desktop application: Home, SDKs, plugins, tasks and settings. |
| [The web interface and other devices](remote.md) | Use vfox-plus from a browser, keep it running without its window, let phones and other computers connect. |
| [The Android app](android.md) | Drive the vfox-plus on your PC from your phone. |
| [The gateway (Docker)](gateway.md) | Show the interface of a vfox-plus in the browsers that you pair with a container. |

## License

vfox-plus is licensed under the [Apache License 2.0](../../../LICENSE).
Material Symbols are licensed under the Apache License 2.0 by Google.
