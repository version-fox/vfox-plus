# Development environment

> [简体中文](../../zh-CN/dev/environment.md)

How to set up a machine to work on vfox-plus: Ubuntu 26.04 LTS, on its own or in
WSL2 on Windows 11, or a dev container that has all of it. What you install
depends on what you do:

| You want to | You need |
| --- | --- |
| Have every tool in your editor at once, and change nothing on your machine | Docker, and an editor that opens dev containers ([The dev container](#the-dev-container)) |
| Run every check, the end-to-end tests and the packaging, and change nothing on your machine | Docker, and Go to build `vfxtool` ([The sandbox](#the-sandbox)) |
| Edit and test the Go code quickly | Go, a few packages and golangci-lint ([Go and the native tools](#go-and-the-native-tools)) |
| Build the Android client | A JDK 17 and the Android SDK ([Android](#android)) |
| Run the Android client on an emulator | KVM, an adb device, or CI ([The emulator and KVM](#the-emulator-and-kvm)) |

The pinned versions live in
[`mygo/versions.env`](../../../mygo/versions.env) and
[`android/versions.env`](../../../android/versions.env). The commands below read
them, so that this page does not go stale when one changes, and so does the image of
the dev container when it is built.

## The sandbox

`vfxtool sandbox` runs the checks of CI, the end-to-end tests and the packaging in
throw-away Docker containers (see [Shared sandbox](sandbox.md)), so that nothing
but Docker has to be installed, and nothing on the machine changes: not its
toolchains, not its home directory, not a vfox that is installed. Docker Engine
works, rootful or rootless (`vfxtool sandbox` tells them apart). In WSL2 install the
Engine inside the distribution; it needs systemd (see [WSL2](#wsl2)).

`vfxtool` is a Go program: build it once with the Go of the next section, then run
everything through it. The first run builds the images, which takes a while.

```sh
go -C shared/tooling build -o ~/.local/bin/vfxtool ./cmd/vfxtool
vfxtool sandbox all
```

Behind a slow connection, set `SANDBOX_UBUNTU_MIRROR`, `SANDBOX_GOPROXY` and `SANDBOX_NPM_REGISTRY` (see
[the mygo guide](implementation-mygo.md#sandbox)).

## The dev container

[`.devcontainer`](../../../.devcontainer) describes a container that has what the
sections below install, so that you install none of it: Ubuntu 26.04 with Go,
golangci-lint and vfox in the pinned versions, the packages that CI installs, a
virtual display, a JDK and the Android SDK, and Docker. VS Code (with the Dev
Containers extension), GitHub Codespaces and the [dev container
CLI](https://github.com/devcontainers/cli) open it. The image is for x86-64, like
the archives it installs; Docker runs it under emulation on another processor.

In VS Code, open the repository and run *Dev Containers: Reopen in Container*. From a
terminal, with Docker running:

```sh
npm install --global @devcontainers/cli
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . bash
```

The first start builds the image, which takes minutes, downloads about 1 GB and fills
3.6 GB of disk, and then runs the `postCreateCommand` of
[`devcontainer.json`](../../../.devcontainer/devcontainer.json): it builds `vfxtool`
into `/home/vscode/go/bin` (`~/go/bin` of the user, on the `PATH`), downloads the Go
modules and prepares the assets that the application embeds (without the licenses of
the Go modules, which only a package needs; `vfxtool prepare` without
`MYGO_SKIP_LICENSES` collects them). After that the commands of [the mygo
guide](implementation-mygo.md#everyday-commands) and of
[Contributing](contributing.md#workflow) run as they are written:

```sh
cd mygo
golangci-lint run && CGO_ENABLED=1 go test -race ./...
vfxtool build --e2e build/vfox-plus-e2e
xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- vfxtool e2e-run real build/vfox-plus-e2e
```

`VFOX_PLUS_E2E_VFOX` names the vfox that the image installs, in the version that
`mygo/versions.env` pins, so the scenario against the real vfox needs no prefix and
has no `~/.vfox` of yours to touch. `ANDROID_HOME` is set, and `cd android &&
./gradlew :app:assembleRelease` builds the Android client ([Android](#android)).

The container runs with an init process (`init`), which reaps the processes that the
tests leave behind, and with 1 GB of shared memory (`--shm-size`), because software
rendering and the embedded test server need more than the 64 MB that Docker gives a
container by default. The sandbox runs its containers the same way.

### Rebuilding

The Dockerfile reads `mygo/versions.env` and `android/versions.env` while the image is
built, as CI and the sandbox do, so a version that changes there (in a `git pull`,
say) reaches the container when you rebuild it: *Dev Containers: Rebuild Container*,
or `devcontainer up --workspace-folder . --remove-existing-container`. The
tools are in `/usr/local` and `/opt`, for the user to use and not to change. What
the user downloads lives under `/home/vscode`, and two Docker volumes keep it when the
container is rebuilt: `vfox-plus-devcontainer-go` (the modules and the build cache of
Go) and `vfox-plus-devcontainer-gradle` (Gradle, and the libraries of the Android
app). `docker volume rm` deletes them.

### Behind a slow connection

The variables of the sandbox work here too. Set `SANDBOX_UBUNTU_MIRROR` (for the
build of the image), `SANDBOX_GOPROXY` and `SANDBOX_NPM_REGISTRY` (they reach the
container as `GOPROXY` and `VFOX_PLUS_NPM_REGISTRY`) in the environment that starts the
editor or the CLI. Go, golangci-lint, vfox and the Android command-line tools come
from go.dev, GitHub and dl.google.com, and the Docker feature from ghcr.io, with no
mirror to name for any of them ([Docker inside](#docker-inside) says what to do where
ghcr.io cannot be reached). Your `~/.gradle` is not
the container's: the download mirror that the Gradle wrapper reads from
`gradle-wrapper-neo.json` ([Building](packaging-android.md#building)) belongs in the
Gradle user home of the container, `/home/vscode/.gradle` (`$GRADLE_USER_HOME`).

### Rootless Docker

Rootless Docker maps the root of a container to you and every other user to a
subordinate user, who may not write to your files: in a container that runs as
`vscode`, the repository is read-only. Set `VFOX_PLUS_DEVCONTAINER_USER=root` in the
environment that starts the editor or the CLI (in `~/.profile`, before `code .` or
`devcontainer up`), and the container runs as root, which is you outside it: the files
it writes are yours. With Docker Desktop, Docker Engine as root, or Codespaces, leave
it alone. The user is `vscode`, with `sudo` and no password, and on Linux it gets
your uid (`updateRemoteUserUID`).

### Docker inside

Docker Engine runs in the container, with images of its own (the
[`docker-in-docker`](https://github.com/devcontainers/features/tree/main/src/docker-in-docker)
feature, which makes the container privileged). The three things of this
repository that use Docker work in it as they are written: `vfxtool sandbox`, and the
license header check and the Markdown check of [Contributing](contributing.md#markdown).
The container already is a throw-away environment with the tools of the sandbox, so
you need the sandbox inside it only for what it fixes: the installation into a clean
Ubuntu (`package`), the memory (`measure`) and the screenshots of the desktop
application (`screenshots`). Where privileged containers are not allowed, or ghcr.io
(where the feature comes from) cannot be reached, delete `features` from
`devcontainer.json`; nothing else needs it.

### Limits

- There is no emulator and no KVM ([The emulator and KVM](#the-emulator-and-kvm)): use
  the `measure` job of CI, or an adb device, as in [Without KVM](#without-kvm).
- There is no display. The window of the application needs one: the tests run in
  `xvfb`, and to use the application, run `vfox-plus-serve` and pair a browser outside
  the container.
- There is no Chrome, which `vfxtool webheap` drives ([Go and the native
  tools](#go-and-the-native-tools)). Install one and name it with `--chrome`.
- A git worktree is a folder whose `.git` is a file that points into the clone it
  belongs to, outside the folder that the container mounts, so git does not work in
  it there (Go still builds). Open the clone.

## Go and the native tools

Go, from its release archive and checked against the checksum that
`mygo/versions.env` pins (`go.mod` asks for 1.27.3 or later). Run this from the
root of the repository:

```sh
. ./mygo/versions.env
curl -fsSL -o /tmp/go.tar.gz "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz"
echo "$GO_SHA256  /tmp/go.tar.gz" | sha256sum --check --strict -
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tar.gz
echo 'export PATH="$HOME/.local/bin:/usr/local/go/bin:$PATH"' >> ~/.profile
```

golangci-lint, the pinned version, the way the sandbox installs it:

```sh
. ./mygo/versions.env
archive="golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64"
curl -fsSL -o "/tmp/$archive.tar.gz" \
  "https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_LINT_VERSION/$archive.tar.gz"
echo "$GOLANGCI_LINT_SHA256  /tmp/$archive.tar.gz" | sha256sum --check --strict -
tar -xzf "/tmp/$archive.tar.gz" -C /tmp
install -D -m 755 "/tmp/$archive/golangci-lint" ~/.local/bin/golangci-lint
```

The Ubuntu packages are the ones that CI installs (`LINUX_PACKAGES` in
[`mygo.yml`](../../../.github/workflows/mygo.yml)):

```sh
sudo apt install build-essential libgtk-3-0t64 libayatana-appindicator3-1 fontconfig fonts-ubuntu fonts-noto-cjk \
  xvfb xauth dbus-x11
```

`build-essential` is only for the race detector of the unit tests, which needs a C
compiler; the application is built without one. GTK 3 and the fonts are what the
application loads when it starts. `xvfb`, `xauth` and `dbus-x11` give the
end-to-end tests a virtual display. Add `lintian desktop-file-utils` to check a
`.deb` the way CI does. `libayatana-appindicator3-1` draws the icon that stands in the tray for the application
while its window is closed, which the end-to-end scenario `tray` needs; it runs with the session bus of
`dbus-run-session`, for it uses a stand-in for the host of the tray icons (see
[The tray](architecture.md#the-tray)). To see the icon on your own desktop, the desktop has to be GNOME
(with its AppIndicator extension) or Windows. The scenario `tray-gnome` runs in the container of
`shared/sandbox/gnome.Dockerfile`, so it needs Docker. The commands to build, test and package are in
[the mygo guide](implementation-mygo.md#everyday-commands).

`vfxtool webheap`, which screenshots the web interface
(of a page, or of a scene such as the distribution of Java) with a look at its
JavaScript heap for local diagnosis, and `vfxtool gateway-shots`, which takes the
pictures of the pages of the gateway, drive a Chrome that speaks the DevTools Protocol: name it with
`--chrome`, or let the tool look for `google-chrome`, `chromium` and the like (see
[Screenshots](contributing.md#screenshots), which also says which Chrome gives the
size you ask for). The end-to-end scenarios against the real vfox run in the
sandbox, which brings its own pinned vfox, so that nothing touches your `~/.vfox`.

## Android

The build needs a JDK 17 (CI uses Temurin) and the Android SDK with platform 36 and
build-tools 35.0.0, and runs through the committed wrapper; see
[Packaging Android](packaging-android.md#building). That needs no KVM. The
emulator does.

## The emulator and KVM

The `measure` job of [`android.yml`](../../../.github/workflows/android.yml)
installs the release APK on an emulator (API 36, `google_apis`, x86_64) and runs
`vfxtool measure-apk` there. The emulator will not start an x86_64 image without
hardware acceleration, which on Linux is KVM, so to do the same on your machine
`/dev/kvm` must work. CI enables it with a udev rule on the runner. Nothing else
in this repository needs KVM.

### Check it

Inside Ubuntu:

```sh
grep -cE '\b(vmx|svm)\b' /proc/cpuinfo   # more than 0: the processor offers virtualization
ls -l /dev/kvm                           # exists once kvm_intel or kvm_amd is loaded
id -nG | grep -w kvm                     # you may use it: the group of /dev/kvm
journalctl -k -b | grep -E 'Hyper-V: (Host Build|Nested features)'
```

The last line is for WSL2. `Nested features: 0x0` means that Windows does not
offer virtualization to the WSL2 virtual machine, and nothing inside Ubuntu can
change that; a value other than 0 means that it does (`0x4a0000` on a machine
where it worked). `journalctl --list-boots` and `journalctl -k -b -<n>` show the
same lines for the earlier boots, which tells when a change happened. `sudo apt
install cpu-checker` adds `kvm-ok`, which says it in words.

### Make it work

- On WSL2 it needs Windows 11, virtualization (Intel VT-x or AMD-V, SVM) enabled in
  the firmware, and `nestedVirtualization` in `%UserProfile%\.wslconfig`, which is
  `true` unless the file says otherwise ([Microsoft's
  documentation](https://learn.microsoft.com/en-us/windows/wsl/wsl-config)). After
  a change, `wsl --shutdown` in a Windows terminal restarts WSL2.
- WSL2 needs systemd for the next step: `[boot]` with `systemd=true` in
  `/etc/wsl.conf`. The kernel of WSL2 builds KVM as modules, and with systemd
  running, udev loads `kvm_intel` or `kvm_amd` when the processor offers the
  extension. Where `/dev/kvm` is still missing although `/proc/cpuinfo` shows
  `vmx` or `svm`, `sudo modprobe kvm_intel` (or `kvm_amd`) says why.
- `/dev/kvm` belongs to the group `kvm`: `sudo usermod -aG kvm "$USER"`, then start a
  new session (in WSL2, `wsl --shutdown`, which also ends everything that runs in it).
  To skip the new session, give the device to everybody with the rule that CI uses. It
  takes effect at once, programs that already run included, and it stays after a restart;
  on a computer that one person uses there is no one to keep the device from:

  ```sh
  echo 'KERNEL=="kvm", GROUP="kvm", MODE="0666", OPTIONS+="static_node=kvm"' | sudo tee /etc/udev/rules.d/99-kvm4all.rules
  sudo udevadm control --reload-rules
  sudo udevadm trigger --name-match=kvm
  ```

  `ls -l /dev/kvm` shows `crw-rw-rw-` then. This asks the kernel whether KVM works, and prints
  12 when it does (a `PermissionError` says that you may not use the device yet):

  ```sh
  python3 -I -c 'import fcntl,os; print(fcntl.ioctl(os.open("/dev/kvm", os.O_RDWR), 0xAE00))'
  ```

### Known problem: AMD processors, since the Windows update of October 2026

The cumulative update KB5127648 of Windows 11 (October 2026; build 26100.9711, and
26200.9711 on 25H2) took the virtualization away from WSL2 on AMD processors: no
`svm` in `/proc/cpuinfo`, no `/dev/kvm`, `Nested features: 0x0`, and `kvm_amd: SVM
not supported by CPU` when the module is loaded, although `.wslconfig` is right and
the firmware has SVM on. WSL asks Windows what the processor can do, the answer
fails with `ERROR_INVALID_DATA`, and WSL leaves nested virtualization out without a
word. The reports come from AMD processors of Zen 2 to Zen 4; a report of an Intel
laptop that was not affected is among them.
[microsoft/WSL#41917](https://github.com/microsoft/WSL/issues/41917) tracks it. On
one development machine the journal of the kernel shows the day: the boot of
2026-10-13 has `Host Build 10.0.26100.9444`, `Nested features: 0x4a0000` and
`kvm_amd: Nested Virtualization enabled`, and the boot of 2026-10-14 has
`Host Build 10.0.26100.9711` and `Nested features: 0x0`.

Microsoft wrote on 2026-10-26 that it believed a fix would come through Windows
Update the next day. In Windows, install the updates that are offered, restart, run
`wsl --shutdown`, and check again as above. Until then the update can be removed
with `wusa /uninstall /kb:5127648` in an administrator terminal (it is a security
update, and for some it cannot be removed this way). WSL 3.0.4 only adds a warning,
`wsl: Failed to detect nested virtualization support, error: ERROR_INVALID_DATA`,
and does not help.

### Without KVM

- Let CI do it: the `measure` job of `android.yml` runs on every pull request that
  touches `android/` or `mygo/pair/`, and uploads `memory-android` and
  `screenshot-android`.
- `vfxtool measure-apk` needs only `adb` on `PATH` (`sudo apt install adb`) and a
  device that `adb devices` lists: a phone with wireless debugging (`adb pair`, then
  `adb connect`), or an emulator that runs on the Windows side and is reached
  through an adb server there (set `ANDROID_ADB_SERVER_ADDRESS` and
  `ANDROID_ADB_SERVER_PORT` in WSL, let that server listen with `adb -a nodaemon
  server`, and let the Windows firewall through). Its numbers belong to that
  device: they do not compare with the emulator of CI.

### An emulator of your own

With KVM working, the emulator needs the command-line tools of the SDK, the build that
`android/versions.env` pins (the zip `commandlinetools-linux-<build>_latest.zip` of
dl.google.com, checked against its checksum, unpacked to `$ANDROID_HOME/cmdline-tools/latest`), and
three packages. The download is about 3 GB, and `sdkmanager` asks you to accept the licenses of the
Android SDK:

```sh
export ANDROID_HOME=$HOME/Android/Sdk
export PATH=$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/emulator:$PATH
sdkmanager --licenses
sdkmanager platform-tools emulator "system-images;android-36;google_apis;x86_64"
avdmanager create avd -n pixel6 -k "system-images;android-36;google_apis;x86_64" -d pixel_6
emulator -avd pixel6 -no-window -no-audio -no-boot-anim -no-snapshot -gpu swiftshader_indirect &
adb wait-for-device && adb shell getprop sys.boot_completed   # 1 once it has booted, about a minute
```

The profile `pixel_6` is 1080 by 2400 at 420 dpi, as the emulator of CI is. The back camera of the
scanner shows a living room when `hw.camera.back=virtualscene` is in
`~/.android/avd/pixel6.avd/config.ini` (`avdmanager` writes `emulated`, which shows a
plain animation); `hw.ramSize=3072M` is a better size than its 1536M; the paired session holds the whole inventory in memory. Leave out `-no-window` to see the
emulator on a display (WSLg). To install the release APK that CI has built, without building it,
download it from the run of the pull request: `gh run download RUN -n vfox-plus-android`.

### Run the app

With an emulator that has booted (`adb devices` lists it), build the app, install
it and pair it with the application on your machine:

```sh
cd android && ./gradlew :app:assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell pm grant dev.vfox.vfoxplus android.permission.CAMERA   # for the scanner
```

The emulator reaches your machine as `10.0.2.2`. The session cookie of the servers is
`Secure`, and the app keeps it for `https://` and for loopback only, so pairing over
`http://10.0.2.2:PORT` does not work; the server for other devices speaks HTTPS anyway. Make a
self-signed certificate that names `10.0.2.2`, and give it to the application:

```sh
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -days 30 \
  -subj /CN=vfox-plus-emulator -addext subjectAltName=IP:10.0.2.2,IP:127.0.0.1 \
  -keyout key.pem -out cert.pem
```

Start the application (`vfxtool build build/vfox-plus && build/vfox-plus`) and, in its settings,
turn on the web interface and *Other devices*, then choose *Certificate → Browse…* and the
files `cert.pem` and `key.pem`. (The same settings can be written to
`~/.config/vfox-plus/settings.json` before the start: `"webInterface": true`, `"webRemote": true`,
`"webCertificate": "/path/cert.pem"` and `"webKey": "/path/key.pem"`.)

The end-to-end build (`vfxtool build --e2e`) needs neither the certificate nor the replacing of the address:
with `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2` in its environment, the certificate that it makes for itself names
`10.0.2.2`, and so do the settings and the link that *Show code* gives. That is how the pictures of the guide
are taken ([Screenshots](contributing.md#screenshots)).

Choose *Show code* next to *Pair a phone*, copy the link, and type it into the app, with the
address of your machine replaced by `10.0.2.2`. The app shows the fingerprint of the
certificate; it has to be the one that the settings show and that
`openssl x509 -in cert.pem -noout -fingerprint -sha256` prints.

`vfxtool measure-apk` runs on the same emulator. It installs without `-r`, so
uninstall the app first (`adb uninstall dev.vfox.vfoxplus`).

## WSL2

- **systemd** is needed by the steps above and by rootless Docker. `[boot]`,
  `systemd=true` in `/etc/wsl.conf`, then `wsl --shutdown`.
- **Memory** is half of the machine's unless `%UserProfile%\.wslconfig` says
  `[wsl2]` with `memory=<size>`. The race tests of the Go code, the Docker builds
  and Chrome are the heavy ones.
- **A display** for graphical programs comes from WSLg (`guiApplications` in
  `.wslconfig` is `true` unless it says otherwise). Tests that need a display
  without a window use `xvfb` or the sandbox.
- **Files** in the Linux file system (`~/...`) are much faster than the ones under
  `/mnt/c`: clone the repository into your Linux home.
- **The dev container** works with the WSL integration of Docker Desktop and with a
  Docker Engine in the distribution. Open the clone in the WSL window of VS Code
  (`code .` in the distribution) and reopen it in the container. With a rootless
  Engine, set `VFOX_PLUS_DEVCONTAINER_USER=root` first ([Rootless
  Docker](#rootless-docker)).
