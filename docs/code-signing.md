# VRC++ Windows 代码签名指南

> 实现批注（2026-08-16）：签名脚本已经支持 PFX、Windows 当前用户/本机证书库和自定义 SignTool 路径。当前开发电脑尚未安装 SignTool，也没有可信代码签名证书，因此现有候选安装程序仍为未签名版本。

## 1. 签名解决什么问题

VRC++ 会同时签名内部的 `vrc-plus-plus.exe` 和最终的 `VRC++-Setup-<版本>.exe`。签名可让 Windows 识别发布者、发现文件被篡改，并让自动更新器拒绝来源不可信的安装程序。

时间戳必须保留。带有效时间戳的签名在证书以后到期时仍可验证。

## 2. 选择证书

### 公开开源项目：优先申请 SignPath Foundation

适合预算有限的开源项目。通常要求仓库公开、采用 OSI 认可许可证、项目已经有公开版本、构建可复现，并配置 GitHub CI 和人工签名审批。签名发布者显示为 SignPath Foundation，而不是个人名字。

申请地址：<https://signpath.org/apply.html>

### 个人直接下载分发：购买 OV 代码签名证书

向受 Windows 信任的 CA 购买 OV Code Signing Certificate。申请时需要完成个人或组织身份验证。现代证书通常要求私钥存储在硬件令牌或云 HSM 中，不一定提供可导出的 PFX；购买前要确认供应商提供的 SignTool 接入方式。

不要为了消除 SmartScreen 专门购买高价 EV。新证书仍需要逐步积累 SmartScreen 信誉。

### 自签名证书：仅限自己的测试电脑

自签名证书不会被其他 Windows 电脑自动信任，不能用于公开发布。它只适合验证“签名、自动更新、静默升级”整条流程。

## 3. 安装 SignTool

从微软 Windows SDK 安装程序中只选择 `Windows SDK Signing Tools for Desktop Apps`。安装后常见位置为：

```text
C:\Program Files (x86)\Windows Kits\10\bin\<SDK版本>\x64\signtool.exe
```

脚本会自动查找，也可以显式指定：

```powershell
$env:VRC_HARBOR_SIGNTOOL_PATH = 'C:\Program Files (x86)\Windows Kits\10\bin\<SDK版本>\x64\signtool.exe'
```

## 4. 使用 PFX 签名

不要把 PFX、密码或证书导出文件放进 Git 仓库。

```powershell
$env:VRC_HARBOR_SIGN_CERT_PATH = 'D:\Certificates\vrc-plus-plus-code-signing.pfx'
$securePassword = Read-Host 'PFX password' -AsSecureString
$temporaryCredential = [pscredential]::new('unused', $securePassword)
$env:VRC_HARBOR_SIGN_CERT_PASSWORD = $temporaryCredential.GetNetworkCredential().Password
powershell -ExecutionPolicy Bypass -File scripts\package.ps1 -RequireSigned
Remove-Item Env:\VRC_HARBOR_SIGN_CERT_PASSWORD
```

构建顺序是：签名主程序 → 生成安装程序 → 签名安装程序 → 验证最终签名。任何一步失败都会停止发布。

## 5. 使用 Windows 证书库签名

列出当前用户代码签名证书：

```powershell
Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert |
  Select-Object Subject, Thumbprint, NotAfter
```

选择证书后：

```powershell
$env:VRC_HARBOR_SIGN_CERT_THUMBPRINT = '<证书 Thumbprint>'
$env:VRC_HARBOR_SIGN_CERT_STORE = 'CurrentUser'
powershell -ExecutionPolicy Bypass -File scripts\package.ps1 -RequireSigned
```

如果证书装在 `LocalMachine\My`，把存储位置改为：

```powershell
$env:VRC_HARBOR_SIGN_CERT_STORE = 'LocalMachine'
```

## 6. 验证结果

```powershell
Get-AuthenticodeSignature 'dist\vrc-plus-plus.exe' |
  Select-Object Status, StatusMessage, SignerCertificate

Get-AuthenticodeSignature 'dist\VRC++-Setup-<版本>.exe' |
  Select-Object Status, StatusMessage, SignerCertificate
```

正式发布必须显示 `Valid`。脚本中的签名摘要算法属于 Authenticode 标准流程，不会另外生成供用户核对的文件校验值。

## 7. 发布密钥安全

- 证书私钥和密码不能写入源码、Markdown、日志或 Release；
- GitHub Actions 使用加密 Secrets，且只允许受保护的发布分支触发签名；
- PFX 应保存在加密磁盘或密码管理器附件中，并保留离线备份；
- 证书泄露时立即联系 CA 撤销，并停止自动更新发布；
- 每次都签名主程序和安装程序，不能只签最外层安装器。
