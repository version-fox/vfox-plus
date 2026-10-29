# Development environment

> [简体中文](../../zh-CN/dev/environment.md)

How to set up a machine to work on vfox-plus: Ubuntu 26.04 LTS, on its own or in
WSL2 on Windows 11. What you install depends on what you do:

| You want to | You need |
| --- | --- |
| Run every check, the end-to-end tests and the packaging, and change nothing on your machine | Docker, and Go to build `vfxtool` ([The sandbox](#the-sandbox)) |
| Edit and test the Go code quickly | Go, a few packages and golangci-lint ([Go and the native tools](#go-and-the-native-tools)) |
| Build the Android client | A JDK 17 and the Android SDK ([Android](#android)) |
| Run the Android client on an emulator | KVM, an adb device, or CI ([The emulator and KVM](#the-emulator-and-kvm)) |

The pinned versions live in
[`mygo/versions.env`](../../../mygo/versions.env) and
[`android/versions.env`](../../../android/versions.env). The commands below read
them, so that this page does not go stale when one changes.

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
sudo apt install build-essential libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk \
  xvfb xauth dbus-x11
```

`build-essential` is only for the race detector of the unit tests, which needs a C
compiler; the application is built without one. GTK 3 and the fonts are what the
application loads when it starts. `xvfb`, `xauth` and `dbus-x11` give the
end-to-end tests a virtual display. Add `lintian desktop-file-utils` to check a
`.deb` the way CI does. The commands to build, test and package are in
[the mygo guide](implementation-mygo.md#everyday-commands).

`vfxtool webheap`, which measures the heap of the web interface and takes its
screenshots (of a page, or of a scene such as the distribution of Java), drives a Chrome that speaks the DevTools Protocol: name it with
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
  new session (in WSL2, `wsl --shutdown`).

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
