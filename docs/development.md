# 本地开发

## 环境

- Windows 10/11；
- Node.js 24+ 与 npm；
- Go 1.26+。

当前工作区使用 `.tools/go` 内的免安装 Go 1.26.5。该目录被 Git 忽略，不会进入项目。

如果需要重新准备工具链：

1. 从 [Go 官方下载页](https://go.dev/dl/) 下载 Windows x64 工具链；
2. 解压到项目 `.tools`，最终路径应为 `.tools/go/bin/go.exe`。

## 安装前端依赖

```powershell
npm.cmd --prefix apps/web install
```

使用 `npm.cmd` 是为了避免某些 Windows 执行策略拦截 `npm.ps1`。

## 开发运行

终端一：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/dev.ps1
```

终端二：

```powershell
npm.cmd --prefix apps/web run dev
```

打开 `http://127.0.0.1:5173`。Vite 会把 `/local` 请求转发给 `127.0.0.1:47831`。

## 界面主题

登录页右上角、登录后侧边栏和控制台页头均可切换浅色/深色主题。首次打开跟随系统配色，手动选择写入浏览器的 `vrc-harbor-theme` 本机偏好；该值不包含账号或会话信息。浏览器禁止本地存储时，切换在当前页面仍然有效。

主题色统一维护在 `apps/web/src/tokens.css`，组件只使用语义变量，避免在页面中散落固定浅色或深色值。修改界面后至少检查浅色与深色桌面宽度，以及 390px 移动端宽度。

## 构建

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1
```

输出为 `dist/vrc-plus-plus.exe`。它只内嵌运行所需的前端资源，不再发布开发文档路由，默认只监听 `127.0.0.1:47831`。

双击或直接运行 EXE 时默认打开内嵌 WebView2 窗口；缺少 WebView2 Runtime 时自动回退到默认浏览器。程序使用 Windows 单实例守卫，重复启动会唤醒已有窗口。常用参数：

- `-desktop=false -open-browser=true`：直接使用默认浏览器；
- `-desktop=false -open-browser=false`：不打开任何窗口，用于自动化或前后端分离开发；
- `-tray=false`：不显示 Windows 通知区域图标；
- `-listen`、`-data-dir`、`-dev-origin`：覆盖监听地址、数据目录和开发源站。

## User-Agent

发行构建会按当前版本生成 `VRCPlusPlus/<版本> 2579362548@qq.com`，用户无需配置。开发者需要测试其他应用标识时可临时覆盖：

```powershell
$env:VRC_HARBOR_USER_AGENT='YourApp/0.1 your-email@example.com'
```

## 网络模式

页面右上角或登录后侧边栏可切换：

- `system`：读取 `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY`；
- `direct`：强制直连；
- `http`：本机 HTTP/HTTPS 代理，例如 `http://127.0.0.1:7890`；
- `socks5`：本机 SOCKS5 代理，例如 `socks5://127.0.0.1:7891`。

自定义代理只接受 localhost/回环 IP，不保存 URL 用户名和密码。配置持久化到本机 SQLite，并同时作用于 REST、Pipeline 和图片缓存。

## 数据目录

开发脚本使用项目 `.data`。正式构建默认使用 `%AppData%\VRC++`；若检测到旧开发版 `%AppData%\VRC Harbor` 数据目录，会继续读取以避免丢失会话。数据库中只保存 DPAPI 加密后的 Cookie 会话，不保存密码。
