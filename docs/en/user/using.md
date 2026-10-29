# Use vfox-plus

> [简体中文](../../zh-CN/user/using.md)

The application has five destinations: Home, SDKs, Plugins, Tasks and
Settings. On wide windows they sit in the side rail; on narrow windows they
move to the bottom bar.

## Home

- The vfox version and where its executable was found, with a refresh button.
- Three statistic tiles: plugins, installed versions and versions in use.
  Tapping one opens SDKs.
- The plugins that have a version in use; tapping one opens it on the SDKs
  page.
- How to use the versions in a terminal, with the line to add to `~/.bashrc`
  and a copy button.
- Shortcuts to the plugin registry and to the vfox documentation.

## SDKs

A list of your installed plugins with their current version. Pick one to see
its detail: description, homepage, and two tabs (three for the plugins that
the application knows, see [Java, Node.js and Flutter](#java-nodejs-and-flutter)).

**Installed** lists the installed versions, newest first. The current version
is marked; the others offer to become the global default. Each version can
also be pinned to a project (in its own directory), have its install path
copied, or be uninstalled. Removing the plugin itself deletes all its
versions, after a confirmation.

**Available** searches the versions you can install: filter by text, show
only versions with a note (such as LTS) or only installed ones. Installing
shows live download progress and can be cancelled. **Install by name…** installs
a version that is not in the list by the name vfox understands, such as `24` for
the newest Node.js 24.

### Java, Node.js and Flutter

The plugins [vfox-java](https://github.com/version-fox/vfox-java),
[vfox-nodejs](https://github.com/version-fox/vfox-nodejs) and
[vfox-flutter](https://github.com/version-fox/vfox-flutter) do more than list
versions, and vfox-plus offers all of it, in the desktop application, in the web
interface and in the Android app alike. A plugin that you added under another
name, as `vfox add --alias coffee java` does, offers the same as the plugin it was
made from:

- **Distributions of Java.** A plain `vfox search java` lists OpenJDK from
  jdk.java.net only. The button above the list (**Distribution: OpenJDK**) opens a
  list of the 31 distributions, such as Temurin, Zulu, Liberica, Corretto and
  GraalVM, which you filter by name, short name or maker; **All distributions**
  lists every one of them, about 2,500 versions, and says which each is of. The
  choice is remembered, for the next time you open the plugin and after a restart; it
  is kept in the settings of the desktop application, so the web interface and the
  Android app show the same one. It is the same as `vfox search java tem`.
- **Filter chips.** Chips above the list narrow it by what the plugin says of its
  versions: LTS and JavaFX for Java, LTS for Node.js, the channels (stable, beta,
  dev, OpenHarmony) and the architectures (x64, arm64) for Flutter. Chips of one
  row are alternatives; rows narrow each other.
- **Install by name.** vfox resolves some names while it installs: `24` is the
  newest Node.js 24 and `latest` the newest release of all, `stable` and `beta`
  are the releases the Flutter channels point to, `21-tem` is the newest
  Temurin 21, `21-zulu-fx` Zulu 21 with JavaFX, `28.ea-open` an early-access
  OpenJDK. The dialog suggests such names for the plugin and says what each one
  stands for. A channel is replaced by the release it points to, which is the
  name that **Installed** shows afterwards.
- **Installed names.** An installed Java version names its distribution, with
  JavaFX if it has it, and an installed Flutter version its architecture.

| Install by name | A download mirror |
| --- | --- |
| ![The dialog that installs a version by its name](https://iili.io/n8ex6UN.png) | ![Choosing the download mirror of Node.js](https://iili.io/n8eIKvI.png) |

The **Options** tab holds what is particular to the plugin:

- **Downloads**: the environment variables that the plugin reads, with the mirrors
  that were checked to choose from: the GitHub mirror of Java and Flutter, the
  download mirror of Node.js, the storage mirror of Flutter. They are added to
  every vfox command that the application runs, and the application checks that
  an address starts with `http://` or `https://`.
- **Version files**: `.sdkmanrc` for Java, `.node-version` and `.nvmrc` for
  Node.js, with the vfox settings that decide whether and how vfox reads them.
- **Names you can type**, each with a button that opens the dialog with that name.
- **Good to know**: the things that surprise people, such as that Java without a
  distribution is OpenJDK from jdk.java.net, which stops at 21.0.2, and that
  `flutter@stable` cannot be used afterwards because the install is named after
  its release.
- **Documentation**: the pages of the plugin that explain it, in your language.

## Plugins

- Search the registry and filter official or community plugins.
- Each card tells whether the plugin is already installed and opens its
  homepage.
- Add a plugin from the registry, from a URL, or from a local `.zip` archive
  or `.lua` script file — plus update all plugins at once.
- Installed plugins offer update and remove.

## Tasks

Every install, uninstall, switch, add, remove and update becomes a task,
newest first. Each task shows its state (queued, running, succeeded, failed,
cancelled), live progress for running installations, and the output log with
a count of dropped earlier lines; a failed task keeps its full log on disk
and shows where.
Failed and cancelled tasks can be retried; finished ones can be cleared. When
a task finishes while you look at another page, a message reports it.

## Settings

- **Appearance**: light, dark or follow the system; use the desktop accent
  colour or pick one of the swatches; the language (English or 简体中文).
- **vfox**: which executable is used (detected automatically, or pick your
  own) and extra environment variables such as download mirrors.
- **vfox configuration**: proxy, registry, storage directory, legacy version
  files, caches and timeouts — switches, menus and text fields; clearing a
  field resets it.
- **About**: version, source code and dependency licenses.

## When vfox is missing

If no vfox executable is found, or the one found does not work, every page
explains how to install vfox and offers to locate the executable or retry.
Settings stay reachable so you can point the application at vfox yourself.
