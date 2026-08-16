# VRC++

面向中文用户的非官方 VRChat 本机伴侣。项目采用 **WebView2/Vue 桌面界面 + Go 本机网关**：页面负责交互，本机网关负责 VRChat 会话、API、Pipeline、游戏日志和本地数据，避免把用户凭据与会话集中上传到第三方服务器。缺少 WebView2 Runtime 时会自动使用默认浏览器。

> 当前版本：`0.9.0-beta.4`。这是测试版本，不是 VRChat 官方产品，也不能替代 VRChat 客户端。VRChat API 与平台规则可能变化，请在使用和发布前重新核对上游要求。

## 主要能力

- 登录、2FA、Cookie 会话恢复与安全退出；
- 好友、动向、共同好友关系网、世界、公开实例、群组与头像浏览；
- “特别关心”好友全景档案：公开资料、本机日志、世界轨迹、私人位置类型、同场人物与关系变化证据；
- 关系网默认扫描最多 100 位好友，好友不足 100 位时扫描全部，并提供一键完整扫描；
- 通知、逐人确认的邀请与 Boop；
- Pipeline 实时事件与本机 VRChat 游戏日志历史；
- `system`、`direct`、本机 HTTP 和本机 SOCKS5 四种网络模式；
- VRChat/VRC CDN 图片白名单代理与本地缓存；
- WebView2 桌面窗口、单实例唤醒、Windows 托盘、自动检查与安装器静默更新；
- 浅色/深色主题和移动端布局，运行时不依赖公共前端 CDN。

## 架构与隐私边界

```text
Browser UI (127.0.0.1)
        |
        | Local REST / SSE
        v
Go Local Gateway ---- VRChat API / Pipeline
        |
        +---- SQLite / DPAPI / media cache / game logs
              (all stored on the current Windows user profile)
```

- 网关默认仅监听 `127.0.0.1:47831`，并校验 Origin、CSRF 和安全响应头；
- 密码不会落盘，Cookie 会话使用 Windows CurrentUser DPAPI 加密后保存；
- 自定义代理仅接受 localhost/回环地址，不提供公共代理池或自动换 IP；
- 不实现 Photon 接入、资产下载、批量机器人或绕过平台访问控制的能力；
- 网络适配只能改善本地资源、诊断、缓存和用户自有网络出口体验，不承诺 VRChat 上游永久可达。

更多设计细节见 [实施蓝图](docs/implementation-blueprint.md) 和 [VRChat API 对接说明](docs/vrchat-api-integration.md)。

## 仓库结构

```text
apps/web/       Vue 3 + TypeScript 前端
apps/gateway/   Go 本机网关、VRChat 适配、SQLite 与 DPAPI
docs/           架构、接口、路线图、实现状态和发布说明
scripts/        开发、测试、构建、打包、签名与 UI 检查脚本
```

`.data`、`.tools`、参考仓库、前端依赖、构建产物和发布包不会提交到 Git。

## 环境要求

- Windows 10/11；
- Node.js 24+ 与 npm；
- Go 1.26+，或按 [本地开发说明](docs/development.md) 准备 `.tools/go` 便携工具链。

## 快速开始

```powershell
git clone https://github.com/SuKaa233/vrc-plus-plus.git
cd vrc-plus-plus
npm.cmd --prefix apps/web install

powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1
.\dist\vrc-plus-plus.exe
```

发行版已经内置带联系邮箱的 VRChat User-Agent；开发者仍可通过环境变量覆盖。程序会启动本机网关，并默认打开内嵌的 WebView2 窗口；缺少 WebView2 Runtime 时会回退到默认浏览器。开发模式、启动参数、数据目录和代理配置见 [本地开发说明](docs/development.md)。

## 测试与构建

完整构建脚本依次执行前端类型检查与生产构建、Vitest、Go 全包测试和 Windows 单文件构建：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1
```

也可以分别执行：

```powershell
npm.cmd --prefix apps/web run typecheck
npm.cmd --prefix apps/web run test
& .\.tools\go\bin\go.exe -C apps/gateway test ./...
```

生成标准 Windows 安装程序：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/package.ps1
```

产物为 `dist/VRC++-Setup-0.9.0-beta.4.exe`，不再生成 ZIP 发布包。

发布新版本、生成更新清单以及上传 GitHub Releases 的步骤见 [Windows 发布说明](docs/windows-release.md)。

## 当前验证边界

已覆盖静态检查、自动化测试、本机网关与 UI 验收，以及真实账号只读主流程的多轮验收。以下内容仍需在正式发布环境人工确认：

- 邀请和 Boop 等写操作；
- 真实账号长时间 Pipeline 稳定性与大好友量性能；
- 安装、覆盖升级、卸载和 Authenticode 正式签名；
- 安装器模式自动更新和不同地区/运营商网络表现。

详细进度以 [实现状态](docs/implementation-status.md) 为准；版本变化见 [0.9.0-beta.4 发布说明](docs/release-notes-0.9.0-beta.4.md)。

## 文档

- [本地开发](docs/development.md)
- [实施蓝图](docs/implementation-blueprint.md)
- [VRChat API 对接](docs/vrchat-api-integration.md)
- [功能路线图](docs/feature-roadmap.md)
- [产品点子与待办](docs/ideas-backlog.md)
- [实现状态](docs/implementation-status.md)
- [Windows 发布、签名与更新](docs/windows-release.md)
- [Windows 代码签名指南](docs/code-signing.md)
- [安全策略](SECURITY.md)
- [贡献指南](CONTRIBUTING.md)

## 研究基线与致谢

项目参考了 [VRCX](https://github.com/vrcx-team/VRCX) 的产品思路，但没有复制其界面或将其作为运行时依赖。当前研究基线：

- VRCX 源码快照 `914ea4d3c4d253a3733d364dbaeff99449c6c202`（2026-08-09）；
- VRChat 社区 OpenAPI 规范 `1.20.8`，仓库 HEAD `b7fff1afbf8912def1964bd900f900893cecffd8`（2026-08-13）。

VRChat、VRCX 及相关名称和商标归其各自权利人所有。本仓库未声明开源许可证；未经权利人另行许可，默认保留全部权利。
