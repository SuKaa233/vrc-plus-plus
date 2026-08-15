# 贡献指南

感谢参与 VRC++。当前项目仍在测试阶段，修改应优先保护本机数据、账号会话和上游平台合规边界。

## 开发流程

1. 阅读 [本地开发说明](docs/development.md)、[实施蓝图](docs/implementation-blueprint.md) 和 [实现状态](docs/implementation-status.md)。
2. 从独立分支完成一个范围明确的修改。
3. 不提交 `.data`、`.tools`、参考仓库、依赖、构建产物、账号数据、Cookie、日志或截图中的个人信息。
4. 运行完整构建：

   ```powershell
   powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/build.ps1
   ```

5. 在变更说明中区分：静态检查、自动化测试、本机模拟、真实账号验收以及发布环境验证，不把未执行的验证写成已通过。

## 设计原则

- VRChat 会话和敏感数据只保存在用户本机；
- 网关仅监听回环地址，写操作必须经过 Origin、CSRF、权限和用户确认；
- 保持上游对象与内部 DTO 隔离，兼容 API 字段变化；
- 遵守限流和 `429` 冷却，不引入公共代理池、自动换 IP、批量操作或访问控制绕过；
- UI 保持克制、紧凑、以中文功能信息为主，并同时检查浅色、深色和移动端布局；
- 新能力同步更新相关 Markdown、实现状态和验证边界。

## 提交建议

提交信息建议使用 Conventional Commits，例如：

```text
feat: 增加实例详情只读视图
fix: 修复会话过期后的状态恢复
docs: 补充 Windows 发布验证清单
```

涉及安全问题时请遵循 [安全策略](SECURITY.md)，不要创建包含利用细节的公开 Issue。
