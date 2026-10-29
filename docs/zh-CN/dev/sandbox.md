# 共享沙箱

> [English](../../en/dev/sandbox.md)

实现共用的 Docker 沙箱部分。沙箱在一次性容器跑 CI 检查、端到端测试与打包，
本机什么都不变：工具链与缓存不动，测试者的 vfox 安装与家目录不动。

| 文件 | 职责 |
| --- | --- |
| [`base.Dockerfile`(../../../shared/sandbox/base.Dockerfile) | 基础镜像：Ubuntu 26.04、虚拟显示器、字体、默认壁纸、Debian 打包工具、锁定的 vfox 与非特权用户 `tester` |
| [`../tooling/internal/sandbox`(../../../shared/tooling/internal/sandbox) | 宿主侧（`vfxtool sandbox`）：构建镜像、在容器跑任务、跑安装测试 |
| [`../tooling/internal/runtask`(../../../shared/tooling/internal/runtask) | 容器侧（`vfxtool runtask`）：复制仓库、以 `tester` 跑命令、发布结果、启动虚拟显示器、冒烟测试与安装测试包 |

## 沙箱如何工作

- 镜像在基础镜像上分层。标签取 Dockerfile、锁定版本（`mygo/versions.env`）、vfxtool
  源码与父镜像的哈希，一变即重建。
- 每个任务在新容器跑，仓库只读挂载于 `/src`，复制后以 `tester` 在副本工作。结果进
  `dist/sandbox/mygo/<任务>/`。
- `deps` 任务联网把依赖下载到 Docker 卷（Go 模块缓存）；检查、测试与构建断网跑
  （`--network none`）。
- 安装测试把 `.deb` 装入干净 `ubuntu:26.04` 容器，下载声明的依赖，在此启动应用：
  GUI 工具包部分库运行时加载，`ldd` 看不见，只有干净系统能证明依赖完备。

## 命令

`vfxtool sandbox` 至少接受 `check`、`e2e`、`package`、`all`、`measure`、`shell`、
`image`、`clean`；用法文本列出各自作用。默认服务器慢时设 `SANDBOX_UBUNTU_MIRROR`
（如 `http://mirrors.aliyun.com/ubuntu/`）及实现的镜像变量如 `SANDBOX_GOPROXY`。
