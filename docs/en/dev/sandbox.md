# Shared sandbox

> [简体中文](../../zh-CN/dev/sandbox.md)

The parts of the Docker sandbox that the implementation builds on. The sandbox
runs the checks of CI, the end-to-end tests and the packaging in throw-away
containers, so that nothing on the host changes: neither its toolchains and
their caches, nor the vfox installation and the home directory of whoever runs
it.

| File | Role |
| --- | --- |
| [`base.Dockerfile`](../../../shared/sandbox/base.Dockerfile) | The base image: Ubuntu 26.04, a virtual display, fonts, its default wallpaper, the Debian packaging tools, a pinned vfox and the unprivileged user `tester` |
| [`../tooling/internal/sandbox`](../../../shared/tooling/internal/sandbox) | Host side (`vfxtool sandbox`): builds the images, runs tasks in containers, runs the installation test |
| [`../tooling/internal/runtask`](../../../shared/tooling/internal/runtask) | Container side (`vfxtool runtask`): copies the repository, runs commands as `tester`, publishes results, starts a virtual display, smoke-tests and installation-tests packages |

## How the sandbox works

- The image is layered on the base image. It is tagged with a hash of its
  Dockerfile, its pinned versions (`mygo/versions.env`), the vfxtool sources
  and its parent, so it is rebuilt exactly when one of them changes.
- Every task runs in a new container that sees the repository read-only at
  `/src`, copies it and works on the copy as `tester`. Results go to
  `dist/sandbox/mygo/<task>/`.
- A task starts by deleting the results of its last run. Under rootless Docker the
  files that `tester` wrote belong to a subordinate user of the host, whom you may
  not delete from; a throw-away container of the image deletes them then, as root
  in a container may. `vfxtool sandbox clean` does the same before it deletes the
  image.
- The `deps` task downloads the dependencies into Docker volumes (the Go module
  cache, and the npm package of the icons) with network access; the checks, the
  tests and the builds then run with `--network none`.
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
implementation, such as `SANDBOX_GOPROXY` and `SANDBOX_NPM_REGISTRY`.
