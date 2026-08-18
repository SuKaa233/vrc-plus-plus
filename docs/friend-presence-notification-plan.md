# 好友上下线提醒与强关注通知实施方案

> 规划日期：2026-08-17  
> 状态：待实施

## 1. 产品结论

这项能力应复用现有 VRChat Pipeline 长连接，不新增逐好友高频 REST 轮询。

- **普通关注**：程序在后台运行且 Pipeline 在线时，通过 Windows 托盘提示已选择好友的上线或离线变化。
- **强关注**：增加邮件投递、持久化队列、失败重试、重连后的状态补偿和监测健康提示。
- **能力边界**：电脑关机、程序完全退出、登录失效、网络断开或 Pipeline 不可用期间不能保证实时监测。恢复连接后只能校正当前状态，不能虚构断线期间的准确变化时间。
- **本机优先**：关注名单、状态基线、邮件配置和投递历史留在当前电脑，不建设集中收集好友动态的云端服务。

## 2. 现有底座与缺口

现有 Pipeline 已把 `friend-online`、`friend-offline`、`friend-location` 发布到事件总线，并脱敏保存 30 天；应用关闭窗口后可留在托盘后台运行。当前缺少：

- 多好友关注规则和账号隔离；
- 好友事件字段级归一化、状态机、去重和冷却；
- Windows 托盘通知发送接口；
- SMTP 配置、DPAPI 密钥保护、邮件 Outbox 和重试；
- 批量选择、静默时段、测试通知和投递历史界面；
- Pipeline 断线时的明确“监测暂停”提示。

## 3. 推荐事件链路

```text
VRChat Pipeline
      │
      ▼
事件归一化器 ─► 好友状态机 ─► 关注规则匹配
      │               │             ├─► 托盘通知
      │               │             └─► 邮件 Outbox ─► SMTP Worker
      ▼               ▼
30 天活动历史    状态基线 / 去重记录

Pipeline 重连 ─► 单次好友快照校正 ─► 恢复监测，不批量误报
```

新增的通知消费者必须异步订阅事件总线，不能阻塞 Pipeline 读取循环。规则匹配和 Outbox 写入在 SQLite 中完成，外部邮件请求由独立 Worker 执行。

## 4. 数据模型

### `presence_watch_rule`

以 `(account_id, user_id)` 为主键，保存缓存昵称、`normal/strong` 级别、上线/离线开关、托盘/邮件渠道、邮件确认延迟、静默时段和更新时间。账号 ID 必须进入所有查询条件，防止多账号串数据。

### `presence_state`

记录被关注好友最后确认的 `online/offline/unknown`、最后事件时间、来源和 Pipeline 会话编号。首次启用只建立基线，不发送“上线”提醒。

### `notification_outbox`

记录事件、规则、渠道、脱敏内容、状态、尝试次数和下次重试时间。唯一键 `(event_id, channel)` 防止重复邮件。

### `notification_delivery`

保留最近 30 天的“已通知、已合并、静默、失败、重试中”记录，不保存 SMTP 密码和完整私人位置。

### `secure_secret`

SMTP 密码或应用专用密码使用 Windows CurrentUser DPAPI 加密。API 只返回 `configured`，不得返回密钥；日志和诊断包必须遮盖邮箱、授权信息和邮件正文。

## 5. 防误报规则

1. **首次基线**：启用关注时读取当前状态，只记录不通知。
2. **真实转换**：仅 `offline/unknown -> online` 提醒上线，`online -> offline` 提醒离线。
3. **事件去重**：同一好友和目标状态 5 分钟内只处理一次。
4. **抖动确认**：托盘延迟 10～15 秒，邮件默认延迟 120 秒；期间反转则合并或取消。
5. **重连保护**：重连快照只校正基线。断线期间变化标记“时间不确定”，不批量报告“刚刚上线”。
6. **洪峰合并**：30 秒内超过 3 位好友变化时，合并为“5 位关注好友上线”，详情在应用内查看。
7. **语义边界**：统一写“本机观察到上线/离线”，不解释为对方主动下线、隐身或删除好友。

## 6. 托盘通知

第一版扩展 `internal/tray`：

```go
type Notifier interface {
    Notify(title, body string, level NotificationLevel) error
}
```

Windows 可先使用现有托盘图标的 `Shell_NotifyIcon` 信息通知，内容例如“特别关心好友上线 / Alice 已上线 · 12:35”。点击托盘图标打开应用并定位通知历史。

需要通知按钮、头像和直接打开好友档案时，再升级为 Windows Toast，并在安装器快捷方式配置稳定 AppUserModelID；第一版不同时维护两套通知技术。

## 7. 邮件通道

第一版使用用户自己的 SMTP 服务，不经过 VRC++ 中央服务器：

- 配置 SMTP 主机、端口、TLS、发件地址、收件地址、用户名和应用专用密码；
- 强制 TLS，拒绝明文认证；
- 测试邮件成功后才允许启用强关注；
- 状态稳定 2 分钟后发送，可选 0、2、5、10 分钟；
- 临时错误按 1、5、15、60 分钟退避，最多 6 次；认证失败暂停通道并要求用户修复；
- 默认只发送好友昵称、观察状态和时间，不发送私人实例 ID；
- 支持即时邮件和 15 分钟合并摘要，多人变化时推荐摘要。

## 8. 前端交互

### 好友列表

- 多选好友；
- 批量“开启上线提醒 / 开启上下线提醒 / 设为强关注 / 取消提醒”；
- 好友卡显示铃铛或强关注标识。

### 特别关心页面

- 设置托盘、邮件、上线、离线、延迟和静默时段；
- 显示 `监测正常 / Pipeline 重连中 / 登录失效 / 程序退出后不可监测`；
- 显示最后信号、最后通知和最近投递结果。

### 设置页

- SMTP 配置与 DPAPI 说明；
- “测试托盘通知”“发送测试邮件”；
- 全局静默时段、洪峰合并、邮件摘要；
- 邮件失败原因和手动重试。

## 9. 本机 API

```text
GET    /local/v1/presence-watches
PUT    /local/v1/presence-watches/{userId}
DELETE /local/v1/presence-watches/{userId}
POST   /local/v1/presence-watches/batch
GET    /local/v1/notification-settings
PUT    /local/v1/notification-settings
POST   /local/v1/notification-test/desktop
POST   /local/v1/notification-test/email
GET    /local/v1/notification-deliveries
POST   /local/v1/notification-deliveries/{id}/retry
```

所有写接口继续要求 CSRF。

## 10. 分阶段实施

### Phase 1：事件与托盘闭环

- 关注规则、状态基线和投递历史表；
- 上下线事件归一化；
- 托盘通知、去重、抖动确认和洪峰合并；
- 好友列表批量选择和通知历史；
- Pipeline 不可用时显示“监测暂停”。

退出条件：模拟重复、乱序、首次基线、重连快照和批量洪峰测试通过；真实账号完成一次上线和离线托盘验收。

### Phase 2：强关注邮件

- DPAPI 邮件密钥；
- SMTP TLS、测试邮件、Outbox Worker、重试与摘要；
- 静默时段和邮件延迟确认；
- 投递状态与错误修复入口。

退出条件：本地 SMTP 模拟覆盖成功、超时、认证失败、重试、重启续投和重复事件；真实邮箱只向用户本人验收。

### Phase 3：可靠性补偿

- Pipeline 会话编号和断线区间；
- 重连后单次好友快照校正；
- Windows 登录后自动启动引导；
- 监测健康日报或连续离线告警；
- 可选 Windows Toast 深链。

退出条件：Pipeline 连续运行两小时，模拟断网、睡眠和恢复不误报，不因重连向全部在线好友发送通知。

## 11. 关键测试

- 同一上线事件重复 3 次只通知一次；
- `online -> offline -> online` 在抖动窗口内合并；
- 第一次启用时好友已在线，不发送上线提醒；
- Pipeline 重连返回 50 位在线好友，不产生 50 条通知；
- 切换 VRChat 账号后关注规则不串用；
- SMTP 密码不出现在数据库明文、API、日志和诊断包；
- 应用重启后 Outbox 续投且不重复；
- 邮件认证失败不会无限重试；
- 慢通知消费者不阻塞 Pipeline。

## 12. 推荐优先级

先实现 Phase 1。它直接复用现有 Pipeline、事件总线、SQLite 和托盘底座，风险可控，也能先验证真实事件语义。邮件涉及外部投递和敏感凭据，应在托盘提醒稳定后独立上线。
