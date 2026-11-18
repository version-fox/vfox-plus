# Use vfox-plus

> [简体中文](../../zh-CN/user/using.md)

The application has five destinations: Home, SDKs, Plugins, Tasks and
Settings. On wide windows they sit in the side rail; on narrow windows they
move to the bottom bar.

## Home

- The vfox version and where its executable was found, with a refresh button.
- The plugins that have a version in use; tapping one opens it on the SDKs
  page.
- The health of vfox and the installed SDKs (**Health**), with **Check health**,
  **Adopt** and **Reinstall** where `vfox doctor` finds drift or old installs.
- How to use the versions in a terminal, with the line to add to `~/.bashrc`
  and a copy button.

## SDKs

A list of your installed plugins with their current version. Pick one to see
its detail: description, homepage, and two tabs (three for the plugins that
the application knows, see [Java, Node.js, Flutter, Gradle, Go and Python](#java-nodejs-flutter-gradle-go-and-python)).
On a wide window the detail is always open, the plugin in use until you pick another.

**Installed** lists the installed versions, newest first. The current version
is marked; the others offer to become the global default. Each version can
also be pinned to a project (in its own directory), have its install path
copied, be reinstalled with a fresh copy, be named an alias, or be uninstalled.
Linked versions show their directory
(`17-system (linked → /opt/jdk)`); uninstalling one removes the link only.
Aliases (`[alias: j17]`, `@j17` after the name) work wherever a version does;
the **Options** tab lists, names and removes them. Removing the plugin
itself deletes all its versions, after a confirmation.

**Available** searches the versions you can install: filter by text, show
only versions with a note (such as LTS) or only installed ones. Installing
shows live download progress and can be cancelled. Each listed version offers
**Install**, and **Install and use** in the menu next to it. The menu at the end of the
row above the list (**More ways to add**) has the other ways to add an SDK:
**Install by name…** installs a version that is not in the list by the name vfox
understands, such as `24` for the newest Node.js 24, with **Use after install** to
switch to it at once; **Detect and link** adopts what is already on the machine
(`vfox link --detect`); **Link existing…** links a directory you name as the version
you name.

### Java, Node.js, Flutter, Gradle, Go and Python

The plugins [vfox-java](https://github.com/version-fox/vfox-java),
[vfox-nodejs](https://github.com/version-fox/vfox-nodejs),
[vfox-flutter](https://github.com/version-fox/vfox-flutter),
[vfox-gradle](https://github.com/version-fox/vfox-gradle),
[vfox-golang](https://github.com/version-fox/vfox-golang) and [vfox-python](https://github.com/version-fox/vfox-python) do more than list
versions, and vfox-plus offers all of it, in the desktop application, in the web
interface and in the Android app alike. A plugin that you added under another
name, as `vfox add --alias coffee java` does, offers the same as the plugin it was
made from:

- **Distributions of Java.** A plain `vfox search java` lists OpenJDK from
  jdk.java.net only. The button above the list (**Distribution: OpenJDK**) opens a
  list of the 31 distributions, such as Temurin, Zulu, Liberica, Corretto and
  GraalVM, which you filter by name, short name or maker; **All distributions**
  lists every one of them, about 2,700 versions, and says which each is of. The
  choice is remembered, for the next time you open the plugin and after a restart; it
  is kept in the settings of the desktop application, so the web interface and the
  Android app show the same one. It is the same as `vfox search java tem`.
- **Filter chips.** Chips above the list narrow it by what the plugin says of its
  versions: LTS and JavaFX for Java, LTS for Node.js, the channels (stable, beta,
  dev, OpenHarmony) and the architectures (x64, arm64) for Flutter, and GraalPy and GraalPy Community for Python. Chips of one
  row are alternatives; rows narrow each other.
- **Install by name.** vfox resolves some names while it installs: `24` is the
  newest Node.js 24 and `latest` the newest release of all, `stable` and `beta`
  are the releases the Flutter channels point to, `21-tem` is the newest
  Temurin 21, `21-zulu-fx` Zulu 21 with JavaFX, `28.ea-open` an early-access
  OpenJDK, `3.13` the newest Python 3.13 and `3.13t` its newest free-threaded build, and `latest` the newest Gradle, Go or standard CPython release. The dialog suggests such names for the plugin and says what each one
  stands for. A channel is replaced by the release it points to, which is the
  name that **Installed** shows afterwards.
- **Installed names.** An installed Java version names its distribution, with
  JavaFX if it has it, and an installed Flutter version its architecture.
  Linked versions keep the name you gave them (`17-system`, `brew`).

| Install by name | A download mirror |
| --- | --- |
| ![The dialog that installs a version by its name](https://iili.io/nmseXvn.png) | ![Choosing the download mirror of Node.js](https://iili.io/nmiTxNR.png) |

The **Options** tab holds what is particular to the plugin:

- **Plugin variables**: the variables that the plugin reads, with the mirrors
  that were checked to choose from: the GitHub mirror of Java and Flutter, the
  download mirror of Node.js, the storage mirror of Flutter, the Gradle mirror
  list, the Go mirror and the python.org and uv-build mirrors of Python. They are added to
  every vfox command that the application runs, and the application checks that
  an address starts with `http://` or `https://`.
- **Version files**: a switch with the files that the plugin reads under it,
  `.sdkmanrc` for Java, `.node-version` and `.nvmrc` for Node.js (which work both
  ways on vfox 1.1.0: leaving a project switches back),
  `gradle/wrapper/gradle-wrapper.properties` for Gradle, `.python-version` for Python, and, for Node.js and Python, the
  choice of the version that a file names.
- **Notes**: the few things that surprise people and that no button says, such as
  that Java without a distribution is OpenJDK from jdk.java.net, which keeps only the
  newest releases, and that a Flutter that vfox manages is not upgraded in place, and that a partial name such as `3.13` never picks a free-threaded (`3.13t`) or pre-release build.
  How a name is written, and the common names, are in the **Install by name** dialog.
- **Adopt, drift and shadowing**: when a JDK, Gradle, Flutter, Go or Python is already on
  the machine, **Detect and link** adopts it instead of downloading it again;
  when an SDK changes in place, the **Health** card on Home and `vfox use` report
  drift (**Reinstall** it, **Adopt** older installs); when another
  tool shadows the one of vfox, the health card names it and what to do
  (`path.mode`, `path.preserve`).
- **Documentation**: the pages of the plugin that explain it, in your language.

### Maven and PHP

The plugins [vfox-maven](https://github.com/version-fox/vfox-maven) and
[vfox-php](https://github.com/version-fox/vfox-php) do more than list versions too; the
three interfaces offer it alike:

- **Install by name.** `latest` installs the newest stable release. A pre-release installs
  by its name, such as `4.0.0-rc-7`. For PHP, `latest` is the newest release, which on
  Windows is the thread-safe build.
- **PHP builds.** On Linux and macOS a plain version is the non-thread-safe build; `-zts` names the
  thread-safe one, as in
  `8.5.12-zts`, and `vfox search php` does not list `-zts` builds. On Windows a plain
  version is the thread-safe build, and `-nts` names the non-thread-safe one.
- **Filter chips.** On Windows, PHP has a **Build** filter with the chips TS and NTS. Linux
  and macOS have no such filter.
- **Build options.** On Linux and macOS, the **Options** tab lists PEAR, mhash and PCRE JIT,
  each with **Change**. From PHP 7.4 PEAR is left out by default, and `PHP_WITH_PEAR=yes` builds
  with its `pear` and `pecl` commands; from PHP 8.4 the mhash functions are left out by default,
  and `PHP_WITH_MHASH=yes` builds with them. Both switches print a deprecation warning when set.
  PCRE JIT is on by default.
- **Free-text options.** `PHP_CONFIGURE_OPTIONS` and `MAKEFLAGS` take any text, so the
  **Options** tab does not offer them. Set them in *Settings → Environment variables* as
  `KEY=VALUE`: the application passes that list to every vfox command.
- **Adopt.** **Detect and link** adopts the Mavens that `M2_HOME` and `MAVEN_HOME` name, and
  those of the system package, Homebrew, SDKMAN, asdf, mise, Scoop and Chocolatey. Maven 1
  is not found that way; use **Link existing…**. It also adopts a PHP that is already
  installed, which needs vfox 1.1.0 or later.
- **Java runtime.** Maven runs on a Java runtime that the plugin does not install: put a JDK
  on `PATH` or set `JAVA_HOME`. Maven 1.1 runs on JDK 8 only.
- **Build time.** On Linux and macOS, PHP is compiled from its source, which takes a while.
  The build checks its dependencies first and names any missing packages, so nothing is
  compiled before a missing one is reported.
- **Windows runtime.** On Windows the install adds the Visual C++ 2015-2022 runtime when it
  is missing, after Windows asks for permission.

### Dart, Kotlin and .NET

The plugins [vfox-dart](https://github.com/version-fox/vfox-dart),
[vfox-kotlin](https://github.com/version-fox/vfox-kotlin) and
[vfox-dotnet](https://github.com/version-fox/vfox-dotnet) offer these extras in all
three interfaces:

- **Dart channels.** `stable`, `dev` and `beta` install the newest version on that
  channel, resolved when you install. A version published on several channels is
  listed once for each, so `vfox search dart` can show the same number twice, with
  the channel in the note; the **Channel** filter picks one. `latest` is the highest
  number in the list, which may be a dev build; use `stable` for the newest stable
  SDK. arm64 is not supported on any system.
- **Kotlin.** A version is a Kotlin release in full, such as `2.1.0`; `latest` is the
  newest release on the Kotlin release page. Kotlin/Native is bundled on hosts that
  have a prebuilt archive. `kotlinc` and `kotlinc-native` need Java 17 or newer on
  `PATH`, which the plugin does not install.
- **.NET names and variants.** A channel or release such as `8`, `8.0` or `8.0.34`
  installs the SDK; `-runtime` and `-aspnetcore` install smaller distributions, and
  the **Distribution** filter narrows the list by them. A channel and `latest` select the newest stable release, and a preview only when there
  is none; an exact version such as `11.0.0-rc.3` names a preview. The
  plugin sets `DOTNET_ROOT` and the `PATH`, and on Linux the SDK also needs ICU,
  libssl and zlib.
- **Adopt and no variables.** **Detect and link** adopts SDKs already on the machine —
  Dart SDKs including the one bundled with Flutter, Kotlin compilers from SDKMAN,
  asdf, mise, Homebrew, IDE bundles and `KOTLIN_HOME`, and .NET SDKs including those
  named by `DOTNET_ROOT` and `DOTNET_HOME` — and needs vfox 1.1.0 or later. These
  three plugins read no variables and no legacy version files, so their **Options**
  tab carries only the guide, the names and the documentation.

### CMake

The plugin [vfox-cmake](https://github.com/version-fox/vfox-cmake) offers these extras in
all three interfaces:

- **Names.** Write a release in full, such as `3.31.6`, or a release candidate such as
  `4.5.0-rc1`. A line such as `3.31` is no release, and the install fails. `latest`
  installs the newest release.
- **Release candidates.** The plugin lists a candidate directly below the release it
  belongs to, and a candidate whose release is not out yet goes last, so `latest` is a
  release as long as the list has one. The **Release candidate** filter has a chip for
  each number, `rc1` to `rc6`, and keeps the versions that end in it.
- **Adopt.** **Detect and link** adopts a CMake that is on `PATH`, in `CMAKE_HOME`, in the
  usual system places, or installed by Homebrew, Scoop or Chocolatey. Set `CMAKE_HOME` in
  *Settings → Environment variables* as `KEY=VALUE`: the application passes that list to
  every vfox command.
- **Needs.** The plugin needs vfox 1.1.0 or later, and reaches jsDelivr for the list and
  cmake.org for the archives; it has no mirror to set. It reads no variables and no legacy
  version files, so its **Options** tab carries only the guide, the names and the
  documentation.

## Plugins

- Search the registry and filter official or community plugins.
- Each card tells whether the plugin is already installed and opens its
  homepage.
- Add a plugin from the registry, from a URL, or from a local `.zip` archive
  or `.lua` script file — plus update all plugins at once.
- Installed plugins offer update and remove.

## Tasks

Every install (including install and use, reinstall, link and link detect, alias set and unset, doctor adopt), uninstall, switch, add, remove and update becomes a task,
newest first. Each task shows its state (queued, running, succeeded, failed,
cancelled), live progress for running installations, and the output log with
a count of dropped earlier lines; a failed task keeps its full log on disk
and shows where.
Failed and cancelled tasks can be retried; finished ones can be cleared. When
a task finishes while you look at another page, a message reports it.

## Settings

- **Appearance**: light, dark or follow the system; use the desktop accent
  colour or pick one of the swatches; the language (English or 简体中文).
- **Web interface**: whether vfox-plus also serves its interface to a browser on this
  computer, whether it keeps running without its window (lightweight mode), and
  whether phones and other computers on the network may connect, over HTTPS. See
  [The web interface and other devices](remote.md).
- **vfox**: which executable is used (detected automatically, or pick your
  own) and extra environment variables such as download mirrors.
- **vfox configuration**: proxy, registry, storage directory, legacy version
  files, caches and timeouts, diagnostics (`diagnostics.onUse`) and PATH order
  (`path.mode` baseline by default, `path.preserve`) — switches, menus and
  text fields; clearing a field resets it. Aliases (`vfox alias`) and linked
  versions (`vfox link`) are read from vfox itself.
- **About**: version, source code and dependency licenses.

## When vfox is missing

If no vfox executable is found, or the one found does not work or is older than vfox 1.1.0, every page
explains how to install vfox and offers to locate the executable or retry.
Settings stay reachable so you can point the application at vfox yourself.
