# Windows 发布、签名与双源更新

> 实现批注（2026-08-15）：🟡 流程和代码已完成，开发包可构建；可信证书、正式对象存储和 GitHub Release 尚未配置，因此不能声称当前开发包已签名或可线上自动更新。

> 首个候选版本：`0.9.0-beta.1`。候选包允许无证书供小范围验收；对外公开发布仍必须完成本文第 4 节检查。

## 1. 构建与签名

普通开发包允许无证书构建，并明确输出警告：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\package.ps1
```

正式发布必须带 `-RequireSigned`，否则证书或 `signtool.exe` 缺失会直接失败：

```powershell
$env:VRC_HARBOR_SIGN_CERT_THUMBPRINT = '<Windows 证书库中的 SHA1 指纹>'
powershell -ExecutionPolicy Bypass -File scripts\package.ps1 -RequireSigned
```

也可以设置 `VRC_HARBOR_SIGN_CERT_PATH` 指向 PFX；密码只通过本机环境变量 `VRC_HARBOR_SIGN_CERT_PASSWORD` 提供，不写进仓库。脚本按 DigiCert、Sectigo 顺序尝试 RFC3161 时间戳，并在打包前执行 Authenticode 验证。

## 2. 双源配置

用分号配置两个同内容的 HTTPS manifest，建议国内对象存储在前、GitHub Releases 在后：

```powershell
$env:VRC_HARBOR_UPDATE_URLS = 'https://download.example.cn/vrc-plus-plus/update-manifest.json;https://github.com/owner/repo/releases/latest/download/update-manifest.json'
```

网关顺序尝试来源；首源网络失败、非 200 或清单无效时自动回退。更新包下载后必须匹配 manifest 中的 SHA-256，失败时保留当前版本不动。

## 3. 安装更新

设置页执行“检查 -> 下载 -> 安装并重启”。下载包先进入用户数据目录的 `updates`，校验后仅提取主程序。安装时从单独复制出的 update-helper 启动，等待旧进程退出，再保留 `.previous` 备份、替换主程序并重启；这样不会由正在运行的主 exe 替换自身。

## 4. 发布前仍需验证

- 在干净 Windows 10/11 当前用户环境安装、托盘常驻、退出和卸载；
- 使用真实 EV/OV 代码签名证书签名并通过 SmartScreen/Authenticode 检查；
- 两个正式 HTTPS 来源发布完全一致的 ZIP 与 manifest；
- 手工破坏 ZIP 验证 SHA-256 拒绝；
- 首源断网时验证第二源可检查、下载和替换；
- 更新失败后确认旧版本与本机数据库仍可启动。
