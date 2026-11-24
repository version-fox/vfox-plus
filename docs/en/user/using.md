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
versions, and vfox-plus offers all of it in the desktop application and in the web
interface of the gateway. The phone app has the same SDK pages, except for the
differences that [What differs from the desktop application](android.md#what-differs-from-the-desktop-application)
lists. A plugin that you added under another
name, as `vfox add --alias coffee java` does, offers the same as the plugin it was
made from:

- **Distributions of Java.** A plain `vfox search java` lists OpenJDK from
  jdk.java.net only. The button above the list (**Distribution: OpenJDK**) opens a
  list of the 31 distributions, such as Temurin, Zulu, Liberica, Corretto and
  GraalVM, which you filter by name, short name or maker; **All distributions**
  lists every one of them, about 2,700 versions, and says which each is of. The
  choice is remembered, for the next time you open the plugin and after a restart; it
  is kept in the settings of the desktop application, so the gateway and the phone app show
  the same one. It is the same as `vfox search java tem`.
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
desktop application and the web interface offer it alike, and the phone app offers it
too:

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
  SDK. The plugin does not install arm64 builds, although the official Dart archive
  has arm64 SDKs for Linux, macOS and Windows.
- **Kotlin.** A version is a Kotlin release in full, such as `2.1.0`; `latest` is the
  newest release on the Kotlin release page. If that page cannot be read, search and
  install say why instead of guessing a version. Kotlin/Native is bundled on hosts
  that have a prebuilt archive. `kotlinc` and `kotlinc-native` need Java 17 or newer
  on `PATH`, which the plugin does not install.
- **.NET names and variants.** A channel or release such as `8`, `8.0` or `8.0.34`
  installs the SDK; `-runtime` and `-aspnetcore` install smaller distributions, and
  the **Distribution** filter narrows the list by them. A channel and `latest` select
  the newest stable release, and a preview only when there is none; an exact version
  such as `11.0.0-rc.3` names a preview. A version that matches no release for your
  platform fails the install with an error that names it, and installs nothing. The
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

### Erlang and Clang

The plugins [vfox-erlang](https://github.com/version-fox/vfox-erlang) and
[vfox-clang](https://github.com/version-fox/vfox-clang) offer these extras in all three
interfaces. vfox-erlang was checked on Linux only. vfox-clang was checked on Linux, and its
install and run were also checked on macOS, on Apple silicon and on Intel.

- **Prebuilt Erlang/OTP.** On Linux, the **Prebuilt build** option picks an Ubuntu release,
  20.04, 22.04 or 24.04, and installs Erlang/OTP from its prebuilt build instead of compiling
  it. The version list then holds the prebuilt builds under the release numbers, such as
  `29.1.2`, and its newest build depends on the release: on Ubuntu 20.04 it is 28.4.2. The name
  `OTP-29.1.2` also installs the prebuilt build of that release. vfox records the install under
  the release number, so run it as `erlang@29.1.2`: `vfox exec erlang@OTP-29.1.2` does not find
  it and starts an install of it, which is a source build without `USE_PREBUILT_OTP`. Leave the
  option empty to compile from the source, which takes a long time and needs the tools that the
  plugin's source-build page lists. With a prebuilt build selected, `latest` is the newest release
  in the list. The search cache is kept per plugin and does not depend on the option, so turn it
  off before you use prebuilt releases, as the plugin's prebuilt page says.
- **Erlang settings.** `OTP_CONFIGURE_ARGS` passes arguments to configure, such as
  `--without-javac`, and `MAKEFLAGS` sets the options of make, such as `-j8`. Set them in
  *Settings → Environment variables* as `KEY=VALUE`.
- **Documentation.** The **Options** tab links to the documentation website of vfox-erlang,
  in English and in Simplified Chinese, and to its prebuilt and source-build pages and its README.
- **Windows.** On Windows, `vfox exec erlang@29.1.2 -- erl` does not find `erl` by its bare name
  ([vfox#787](https://github.com/version-fox/vfox/issues/787)); `-- erl.exe` runs it.
- **Clang.** `latest` installs the version that the list marks as latest. The **Conda
  channel** option picks the conda channel that pixi installs clang from; its presets are
  prefix.dev and a Tsinghua mirror, and an empty option means conda-forge. Each install
  downloads pixi first, which took 1 to 2 minutes, and then clang from the channel, which took
  6 to 14 minutes, on a network of 250 to 850 kB/s. On GitHub's macOS runners, which have a fast
  network, an install of `latest` took 6 seconds on Apple silicon and 15 seconds on Intel, and the
  clang it installed ran.
- **Clang's GitHub address.** The plugin reads `GITHUB_URL` for its version list and for
  the pixi download. It has no preset, so set it in *Settings → Environment variables* to
  a mirror that keeps GitHub's paths. A mirror should also serve the `.sha256` file next to each
  archive: without it, the install warns that no checksum is provided, and an error from the mirror
  stops the install.
- **pixi checksum.** The pixi archive that an install downloads is checked against the SHA-256
  that pixi publishes beside it, before vfox unpacks it. When pixi publishes none, the install
  prints `WARNING: Checksum is not provided, skip verify...` and skips the check; when the SHA-256
  cannot be read, the install stops.
- **Detect and link.** `vfox link --detect clang` finds the clang installations that are already on the machine, such as those of conda, Homebrew and the system llvm directories, and links them instead of installing another copy. It needs vfox 1.1.0 or later. It was checked on Linux. On macOS, on GitHub's runners, only the detection was run: it found Homebrew's llvm, and nothing was linked.
- **Home directory.** A source build of Erlang/OTP stops before configure, once it has downloaded the source, when the path of `VFOX_HOME` contains an apostrophe, a double quote, a dollar sign, a backtick, a space, a tab or a newline; the refusal names the character. With `VFOX_HOME` set to a path that has none of those characters, the check passes. A prebuilt build works with a space in the path.

### Elixir

The plugin [vfox-elixir](https://github.com/version-fox/vfox-elixir) offers these extras in all
three interfaces. It was checked on Linux (x86-64 and arm64), on macOS (Apple silicon and Intel) and on
Windows (x86-64 and arm64). On Windows the list is the one of the installers: `latest` installs
`1.19.5-elixir-otp-28` when checked.

- **Erlang/OTP first.** Elixir is built with Erlang/OTP and needs `erl` on `PATH` to run. vfox
  does not install Erlang/OTP for Elixir. A source build uses the `erlc` of an Erlang/OTP that
  vfox installed when `erlc` is not on `PATH`. Install Erlang/OTP first with the erlang plugin,
  and run Elixir together with it, for example
  `vfox exec erlang@29.1.2 elixir@1.20.5 -- elixir --version`. Run alone,
  `vfox exec elixir@1.20.5 -- elixir --version` fails with `exec: erl: not found`.
- **Pre-built builds.** On Linux, the **Pre-built builds** option installs Elixir from the
  pre-built builds of hex.pm instead of compiling it. Their names carry the Erlang/OTP they need,
  such as `1.20.5-otp-29`, and `vfox search elixir hex` lists only those names. A name without an
  `-otp-N` suffix, such as `1.20.5`, is not listed; if you type one, the install prints a warning,
  and the build may not run with your Erlang/OTP. The option does not change `vfox search elixir`,
  which lists the releases. Leave the option empty to compile from the source. A source build takes
  about a minute, and the path of `VFOX_HOME` must not hold a space or another shell character, such
  as a quote, a dollar sign or an ampersand; a pre-built build works with such a path.
- **Checksums.** A pre-built build is checked against the SHA-256 that hex.pm's `builds.txt`
  publishes for it. A build that the index lists without a SHA-256 installs with a warning, and
  is not checked.
- **Versions.** `latest` is the first line of the list: `1.20-latest` on Linux, which ran as
  1.20.5 when checked. With pre-built builds, `latest` stops before the download, so type a build
  name such as `1.20.5-otp-29`. Write a version in full, such as `1.20.5`. For a source build, a
  partial name such as `1.16` is not a release, because its tag does not exist.
- **Searching.** `vfox search elixir` lists the releases whatever the option is, and
  `vfox search elixir hex` lists the pre-built builds. The plugin's documentation says that vfox
  keeps each list for up to 12 hours.
- **Detect and link.** `vfox link --detect elixir` finds the Elixir installations already on the machine, from asdf, mise, Homebrew, Scoop, the system directories and PATH, and links them instead of installing another copy. It needs vfox 1.1.0 or later. It was checked on Linux; on macOS and on Windows the detection ran
  and found nothing, since no other Elixir is installed on those runners.
- **Documentation.** The **Options** tab links to the documentation website of vfox-elixir,
  in English and in Simplified Chinese, and to its README.

### etcd, Deno, Groovy, Scala and Zig

The plugins [vfox-etcd](https://github.com/version-fox/vfox-etcd),
[vfox-deno](https://github.com/version-fox/vfox-deno),
[vfox-groovy](https://github.com/version-fox/vfox-groovy),
[vfox-scala](https://github.com/version-fox/vfox-scala) and
[vfox-zig](https://github.com/version-fox/vfox-zig) offer these extras in all three
interfaces, with the limits that their lists have today:

- **Names.** `latest` installs the first version of the list for Deno, Groovy, Scala and Zig,
  and `nightly` installs the development build of the master branch of Zig. Groovy takes full
  names such as `4.0.33`, but not partial names such as `4`, and its list sorts each
  pre-release below its release. Scala takes full names such as `3.10.1`, but not partial
  names such as `3` or `2.13`. Deno also takes a line such as `2` or `2.1`, which installs
  the newest release of that line. Zig takes full names such as `0.17.1`, and the names of Mach
  builds such as `mach-latest`, but not partial names such as `0.17`. etcd takes full names such
  as `3.7.3`, with or without a `v`, but not partial names such as `3.5`. `latest` installs the
  first line of its list, the newest release.
- **Archive mirror.** The **Archive mirror** option of Groovy offers dlcdn.apache.org, another
  copy of the Apache archive, for when archive.apache.org is slow. It keeps only the current
  releases, so older versions do not appear in the list. A Tsinghua mirror is not offered: its
  list works, but its downloads were refused when this was checked.
- **Stages.** The **Stage** filter of Groovy picks the pre-releases or the incubator releases.
- **Downloads.** Scala 2.10.4 and older download from scala-lang.org; Scala 2.10.5 and later,
  and all of Scala 3, download from GitHub. A GitHub download is checked against the `.sha256`
  file next to it when Scala publishes one, and the install says so when it does not. Scala
  3.9.0, which the list shows as LTS, installs under its number.
- **Java.** Groovy and Scala need a JDK to run, and the plugins install none. Scala 2.10.4 did
  not run on the JDK 25 that was checked.
- **Checksums.** Deno and Groovy check each archive against its checksum file when the release
  has one: `.sha256sum` for Deno and `.sha256` for Groovy. They say so when a release has none.
  etcd checks each archive against the `SHA256SUMS` file of its release. A release before 3.4.0
  that publishes no such file, such as 0.3.0, is checked against checksums that the plugin pins,
  and the install says so; a release from 3.4.0 on without the file is refused.
- **Deno.** The list starts at 0.36.0, which is the first release with a download for Linux on
  x86-64. The plugin sets the execute bit on Deno after it unpacks an archive, so 0.36.0 runs.
  An install of 0.36.0 made by a plugin before 0.2.3 keeps the bit off: reinstall it.
- **Zig.** The list needs ziglang.org: when it cannot be reached, the whole list fails.
  machengine.org is optional: without it the list shows the official releases, with a warning.
  A name that the list does not show is refused, and nothing is installed. vfox 1.1.0 `exec`
  does not take `latest`, `nightly` or a Mach name, so give it the version name that `vfox list`
  shows; `vfox use` takes these names. A Mach build that ziglang.org does not publish is
  downloaded from pkg.hexops.org, and its checksum is still checked.
- **Detect and link.** `vfox link --detect zig` finds the Zig installations already on the machine, from asdf, mise, zvm, Scoop, the system directories and PATH, and links them instead of installing another copy. A zig that cannot run is not listed, and neither are the installs that vfox manages or anything under VFOX_HOME. It needs vfox 1.1.0 or later, and it was checked on Linux only.
- **etcd.** The failpoint build, switched on by `BUILD_WITH_FAILPOINT=yes`, builds the version
  you ask for from source with failpoints. It needs Git, Go and GNU Make on the `PATH`, and it
  takes longer than an install from a release. It was checked on Linux and on macOS: 3.4.32, 3.5.12, 3.6.0 and 3.7.4 build with failpoints on
  x86-64 Linux, and 3.5.12 on arm64 Linux and on both kinds of Mac. A release without a gofail target, such as 3.5.5, is refused before anything is
  cloned, and the install names it; the failpoint page of its documents lists the releases that
  have one. Check the result with `GOFAIL_HTTP`. Only `yes` turns the build
  on: `1` and `YES` are refused, and `no` or an empty value installs the official build. The
  **Options** tab offers no switch for it.
- **Windows.** On Windows, `vfox exec` does not find `deno`, `etcd` or `zig` by its bare name
  ([vfox#787](https://github.com/version-fox/vfox/issues/787)). The name with `.exe` runs them, for
  example `vfox exec deno@2.9.8 -- deno.exe --version`.
- **Documentation.** The **Options** tab of vfox-etcd links to its documentation website, in
  English and in Simplified Chinese, then to its documents and its README. The **Options** tabs
  of vfox-deno and vfox-groovy link their documents, in English and in Simplified Chinese, and
  their README.

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
- **Other devices**: whether phones, the gateway and other computers on the network may
  connect, over HTTPS, and whether vfox-plus keeps running without its window for them
  (lightweight mode). See [Other devices and the lightweight mode](remote.md).
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
