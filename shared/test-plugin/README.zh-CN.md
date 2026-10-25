# 离线测试插件

> [English](README.md)

[`testsdk.lua`](testsdk.lua) 是让端到端测试对**真实** vfox 离线运行、不碰任何人 vfox
安装的 vfox 插件：`vfxtool e2e-run` 启动进程内测试服务器
（`shared/tooling/internal/testserver`）实现下述协议，驱动应用跑 `real` 场景。

## 测试如何使用

1. 建空家目录，`HOME` 指向它跑 vfox，vfox 全部状态（`~/.vfox`）落此。
2. 在回环接口启动实现下述协议的 HTTP 服务器，基 URL 经 `VFOX_PLUS_TEST_SERVER` 告诉
   vfox，如 `http://127.0.0.1:43210`。
3. vfox 指向测试仓库：`vfox config registry.address <基URL>/registry`。
4. 驱动应用：列仓库，从仓库添加 `testsdk`（或 `vfox add --source` 加 `testsdk.lua` 拷贝），
   搜索，安装两版本，使用其一，卸载另一，移除插件。

不出本机，结果确定。

## 服务器协议

| 路径 | 内容 |
| --- | --- |
| `/registry/index.json` | `[{"name": "testsdk", "desc": "…", "homepage": "https://github.com/version-fox/vfox-plus"}]`——主页让 vfox 把该插件列为官方。 |
| `/registry/testsdk.json` | 插件清单：`{"name": "testsdk", "version": "0.1.0", "license": "Apache-2.0", "author": "vfox-plus", "downloadUrl": "<基URL>/plugin/testsdk.lua", "minRuntimeVersion": "1.0.0"}`。 |
| `/plugin/testsdk.lua` | 本插件。 |
| `/sdk/testsdk-<版本>.tar.gz` | SDK：gzip 压缩 tar，内含单目录 `testsdk-<版本>/`，下有 `bin/testsdk`。带 `Content-Length` 头分多块发送，vfox 打印下载进度。 |

## 插件提供什么

| 版本 | 备注 | 附带工具 |
| --- | --- | --- |
| `2.0.0` | `LTS` | `testtool 2.0` |
| `1.0.0` | — | — |

故 `vfox search testsdk` 打印：

```text
Available versions:
 - 2.0.0 (LTS) [testtool 2.0]
 - 1.0.0
```
