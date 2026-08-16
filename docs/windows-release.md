# Windows 安装程序、签名与更新

> 实现批注（2026-08-16）：🟡 已切换为 Inno Setup 单文件安装程序，不再生成或对外分发 ZIP；WebView2 桌面窗口、单实例唤醒、浏览器回退、当前用户安装、完整卸载和安装器自动更新已接入。可信代码签名和真实线上更新仍待发布验收。

> 当前候选版本：`0.9.0-beta.2`。开发安装程序可供小范围验收；公开发布前仍需完成本文检查。

## 1. 标准构建

安装 Inno Setup 7 后执行：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\package.ps1
```

输出为：

```text
dist\VRC++-Setup-0.9.0-beta.2.exe
```

安装向导默认安装到 `%LOCALAPPDATA%\Programs\VRC++`，不要求管理员权限。开始菜单入口始终创建；桌面快捷方式由用户选择；开机启动默认关闭。卸载只删除程序文件和快捷方式，不删除 `%APPDATA%\VRC++` 中的账号会话、历史和本机设置。

## 2. 正式签名

证书选择、PFX、Windows 证书库和测试签名步骤见 [Windows 代码签名指南](code-signing.md)。

正式发布必须带 `-RequireSigned`，否则证书或 `signtool.exe` 缺失会直接失败：

```powershell
$env:VRC_HARBOR_SIGN_CERT_THUMBPRINT = '<Windows 证书库中的 SHA1 指纹>'
powershell -ExecutionPolicy Bypass -File scripts\package.ps1 -RequireSigned
```

也可以设置 `VRC_HARBOR_SIGN_CERT_PATH` 指向 PFX；密码只通过本机环境变量 `VRC_HARBOR_SIGN_CERT_PASSWORD` 提供，不写进项目。主程序先签名，再进入安装程序；最终安装程序随后单独签名。

## 3. WebView2

安装程序不捆绑数百 MB 的固定浏览器运行时。Windows 11 和大多数 Windows 10 已提供 Evergreen WebView2 Runtime；应用启动时会检测运行时，缺失时使用默认浏览器打开同一本机界面。以后可在安装器中加入微软 Evergreen Bootstrapper，作为联网补装选项。

## 4. 自动更新

程序启动 12 秒后自动检查一次，之后每 6 小时检查一次；页面每 30 秒读取本机更新状态。发现新版本时显示更新提示，用户确认后下载安装程序，再以静默模式覆盖安装并重启。更新请求会跟随应用中的直连、系统代理或本机代理设置。升级只替换 `%LOCALAPPDATA%\Programs\VRC++` 中的程序文件，不删除 `%APPDATA%\VRC++` 数据。

Beta 默认清单：

```text
https://github.com/SuKaa233/vrc-plus-plus/releases/download/update-beta/update-manifest.json
```

稳定版默认清单：

```text
https://github.com/SuKaa233/vrc-plus-plus/releases/latest/download/update-manifest.json
```

可在第一次公开分发前把国内源编译进安装包，GitHub 会自动作为备用：

```powershell
$env:VRC_PLUS_PLUS_UPDATE_URLS = 'https://你的下载域名/vrc-plus-plus/update-manifest.json'
powershell -ExecutionPolicy Bypass -File scripts\package.ps1 -RequireSigned
```

下载完成后会使用 Windows Authenticode 验证安装程序。正式更新不接受未签名文件；`VRC_PLUS_PLUS_ALLOW_UNSIGNED_UPDATES=1` 只用于开发机端到端模拟，不能给普通用户配置。

## 5. 发布新版本

版本号只需修改根目录 `package.json`。构建脚本会把它写入主程序和安装程序：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\package.ps1 -RequireSigned
powershell -ExecutionPolicy Bypass -File scripts\prepare-release.ps1 `
  -ReleaseNotes '修复好友头像加载','优化关系网性能'
```

生成两个发布文件：

```text
dist\VRC++-Setup-<版本>.exe
dist\update-manifest.json
```

### GitHub Releases

仓库和 Release 必须公开，否则普通客户端无法匿名下载。安装并登录 GitHub CLI 后执行：

```powershell
gh auth login
powershell -ExecutionPolicy Bypass -File scripts\publish-github-release.ps1
```

脚本会创建 `v<版本>` Release 并上传安装程序。Beta 版本还会维护固定的 `update-beta` Release，只替换其中的更新清单；稳定版把清单上传到最新正式 Release。

### 国内对象存储

在腾讯云 COS、阿里云 OSS 或其他支持 HTTPS 的公开读存储中创建固定目录，把安装程序和更新清单上传到同一目录。生成清单时指定下载目录：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\prepare-release.ps1 `
  -PrimaryDownloadBaseUrl 'https://你的下载域名/vrc-plus-plus' `
  -ReleaseNotes '本次更新内容'
```

发布顺序必须是“先上传安装程序，最后覆盖更新清单”。这样客户端不会先发现一个尚未上传完成的新版本。

## 6. 发布前仍需验证

- 在干净 Windows 10/11 当前用户环境完成安装、覆盖升级和卸载；
- 验证开始菜单、桌面快捷方式、可选开机启动和“已安装的应用”条目；
- 验证关闭窗口后托盘驻留、重复双击唤醒和托盘退出；
- 在未预装 WebView2 Runtime 的环境验证浏览器回退；
- 使用可信代码签名证书签名主程序和安装程序并检查 SmartScreen；
- 在测试版本上完整执行“发现更新、下载、退出、覆盖安装、自动重启”；
- 验证卸载和升级不会删除用户会话、历史、标签或关系网布局。
