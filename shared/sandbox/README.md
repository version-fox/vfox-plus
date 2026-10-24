# Shared sandbox

The parts of the Docker sandbox that the implementation builds on. The sandbox
runs the checks of CI, the end-to-end tests and the packaging in throw-away
containers, so that nothing on the host changes: neither its toolchains and
their caches, nor the vfox installation and the home directory of whoever runs
it.

| File | Role |
| --- | --- |
| [`base.Dockerfile`](base.Dockerfile) | The base image: Ubuntu 26.04, a virtual display, fonts, its default wallpaper, the Debian packaging tools, the Material Symbols font that icon fonts are cut from, a pinned vfox and the unprivileged user `tester` |
| [`../tooling/internal/sandbox`](../tooling/internal/sandbox) | Host side (`vfxtool sandbox`): builds the images, runs tasks in containers, runs the installation test |
| [`../tooling/internal/runtask`](../tooling/internal/runtask) | Container side (`vfxtool runtask`): copies the repository, runs commands as `tester`, publishes results, starts a virtual display, smoke-tests and installation-tests packages |

## How the sandbox works

- The image is layered on the base image. It is tagged with a hash of its
  Dockerfile, its pinned versions (`mygo/versions.env`), the vfxtool sources
  and its parent, so it is rebuilt exactly when one of them changes.
- Every task runs in a new container that sees the repository read-only at
  `/src`, copies it and works on the copy as `tester`. Results go to
  `dist/sandbox/mygo/<task>/`.
- The `deps` task downloads the dependencies into Docker volumes (the Go module
  cache) with network access; the checks, the tests and the builds then run
  with `--network none`.
- The installation test installs the `.deb` into a clean `ubuntu:26.04`
  container, which downloads the declared dependencies, and starts the
  application there: GUI toolkits load some libraries at run time, which `ldd`
  does not show, so only a clean system proves that the dependencies are
  complete.

## Commands

`vfxtool sandbox` accepts at least `check`, `e2e`, `package`, `all`,
`measure`, `shell`, `image` and `clean`; its usage text lists what it does.
Behind a slow connection, set `SANDBOX_UBUNTU_MIRROR` (for example
`http://mirrors.aliyun.com/ubuntu/`) and the mirror variables of the
implementation, such as `SANDBOX_GOPROXY`.
