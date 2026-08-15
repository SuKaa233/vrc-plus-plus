# VRChat API 对接说明

> 本文是实现用的接口基线，不是官方稳定契约。  
> Base URL：`https://api.vrchat.cloud/api/1`  
> Pipeline：`wss://pipeline.vrchat.cloud`  
> 调研规范版本：社区 OpenAPI `1.20.8`（2026-08-13）

> 实现批注（2026-08-14）：🟡 REST Client、认证、好友/用户详情/共同好友/世界只读接口、SQLite 快照、GET 合并、Pipeline/SSE 已编码并通过模拟上游测试；真实账号与 Pipeline 长稳尚未验证。

## 1. 接入原则

1. 所有 VRChat 请求只能从 `vrc-adapter` 发出。
2. 前端只使用本项目 DTO，不能依赖上游 JSON。
3. 每个请求都带稳定、可联系的 User-Agent，例如：

   ```text
   VRCPlusPlus/0.9.0-beta.1 contact@example.com
   ```

4. Cookie、Authorization、2FA 和 Pipeline token 不进入前端状态、URL、普通日志或遥测。
5. GET 去重、缓存、限流、退避在 adapter 层统一实现。
6. 端点可能无预告变化；每类响应都做容错解析和 fixture 回归。

## 2. 登录与 2FA 时序

> 实现批注：🟡 时序代码已实现。密码仅在单次请求结构和 Basic Header 构造期间驻留，未写入浏览器存储、SQLite 或日志；JavaScript/Go 无法保证像底层安全语言一样彻底擦除所有字符串副本，因此文档中的“立刻清除内存字节”只能做到 best effort。

### 2.1 首次登录

```text
Web UI                 Local Gateway                    VRChat
  │ 账号/密码(仅本机)        │                              │
  ├────────────────────────>│                              │
  │                         │ GET /config                  │
  │                         ├─────────────────────────────>│
  │                         │ GET /auth/user               │
  │                         │ Authorization: Basic ...     │
  │                         ├─────────────────────────────>│
  │                         │<── Set-Cookie: auth          │
  │                         │<── CurrentUser 或 2FA 提示   │
  │<── 需要 2FA/登录成功 ───┤                              │
  │                         │                              │
  │ 2FA code                │ POST /auth/twofactorauth/...│
  ├────────────────────────>├─────────────────────────────>│
  │                         │<── twoFactorAuth cookie      │
  │                         │ GET /auth/user (Cookie)      │
  │                         ├─────────────────────────────>│
  │<── 登录成功 ────────────┤                              │
```

`Authorization` 的值按社区规范为：

```text
Basic base64(urlencode(username) + ":" + urlencode(password))
```

实现注意：

- 不要把 Basic 值打印出来；
- 不在浏览器中构造 Basic 值；
- 登录完成立刻清除内存中的密码字节，尽力缩短驻留时间；
- 保存 `auth` 和 `twoFactorAuth` Cookie，而不是每次启动重新提交密码；
- Cookie 域、Path、Secure、HttpOnly、Expires 都要完整保存；
- 每个账号独立 CookieJar；
- 切换账号先断 Pipeline，再切换 Jar，不能混用。

### 2.2 2FA 端点

> 实现批注：🟡 TOTP、邮件验证码和恢复码路由均已实现并通过模拟响应验证，缺真实账号烟测。

| 方式 | 方法与端点 | 请求体 |
|---|---|---|
| TOTP | `POST /auth/twofactorauth/totp/verify` | `{"code":"123456"}` |
| 邮件验证码 | `POST /auth/twofactorauth/emailotp/verify` | `{"code":"123456"}` |
| 恢复码 | `POST /auth/twofactorauth/otp/verify` | `{"code":"xxxx-xxxx"}` |

必须从登录响应中的实际 2FA 类型决定 UI，不要默认只有 TOTP。

### 2.3 会话恢复

> 实现批注：✅ Cookie 加密存取和“网络失败不删除会话”分支已实现并有自动化测试；🟡 真实 VRChat Cookie 的过期/属性兼容仍待验证。

启动流程：

1. 用系统密钥解密 CookieJar；
2. `GET /auth/user`，只带 Cookie，不带 Basic；
3. 200：恢复成功；
4. 401 Missing Credentials：清理失效 Cookie，要求重新登录；
5. 401 Unauthorized 且提示 2FA：进入相应 2FA 流程；
6. 网络/403/5xx：保留 Cookie 和离线数据，绝不能误判为密码失效并删除账号。

这一步要特别区分“凭据无效”和“网络不可用”。

### 2.4 退出

- 调用上游退出端点前先停止新的业务请求；
- 关闭 Pipeline；
- 清空内存和磁盘 CookieJar；
- 清除本地敏感缓存由用户选择，不能默认删除备注和历史；
- 本地退出和“删除账号档案”是两个不同操作。

## 3. Pipeline 实时连接

> 实现批注：🟡 token 获取、WebSocket 连接、通用双层 JSON 解码、指数退避和本地 SSE 已实现；好友类事件当前触发节流后的 REST 校正，尚未完成所有事件的字段级增量归并，也未做真实账号两小时长稳。

VRCX 当前实现的基本流程：

1. 登录且好友初始快照加载完成；
2. `GET /auth` 获取临时 token；
3. 连接 `wss://pipeline.vrchat.cloud/?auth=<token>`；
4. 外层消息 JSON 解码后，`content` 通常还需要再次 JSON 解码；
5. 按 `type` 分发；
6. 断线后退避重连，并在恢复后做 REST 校正。

### 3.1 第一版关注的事件

- `notification`、`notification-v2`；
- 好友上线/离线；
- 好友位置、状态、头像变更；
- 好友关系增加/删除；
- 当前用户信息变更；
- 群组相关事件放到第二阶段。

不要把未知事件当错误。未知 `type` 应计数、脱敏落盘样本并忽略，便于以后兼容。

### 3.2 事件幂等

Pipeline 可能重连、重复或乱序。事件键建议：

```text
sha256(accountId | type | upstreamEventId | occurredAt | stablePayloadFields)
```

若上游没有事件 ID，则领域层对状态事件做 `subjectId + stateVersion/observedAt` 比较。

### 3.3 Token 安全

- Pipeline token 只在网关内存中存在；
- URL 可能被网络库错误日志记录，必须在日志 formatter 中遮盖 `auth` 查询参数；
- 不把 VRChat Pipeline 直接转发给浏览器；
- 网关向页面发送已经裁剪的本地领域事件。

## 4. MVP 端点清单

> 实现批注：认证、好友/用户/共同好友、世界与公开实例摘要、实例详情、收藏分组与世界收藏、通知、逐人邀请和好友 Boop 已开放到本地 API；写操作均由用户显式触发。

以下路径来自社区规范和 VRCX 源码，开发时仍需用测试账号验证实际返回。

### 4.1 启动与认证

| 优先级 | 方法与端点 | 用途 | 缓存 |
|---|---|---|---|
| P0 | `GET /config` | API 配置/可用性 | 启动期，短缓存 |
| P0 | `GET /health` | 健康检查 | 诊断时 |
| P0 | `GET /auth/user` | 登录或当前用户 | 不共享缓存 |
| P0 | `GET /auth` | 校验/获取 Pipeline token | 不落盘 |
| P0 | 三个 2FA POST | 完成登录 | 不缓存 |

### 4.2 好友与用户

| 优先级 | 方法与端点 | 用途 |
|---|---|---|
| P0 | `GET /auth/user/friends` | 在线/离线好友分页快照 |
| P0 | `GET /users/{userId}` | 用户详情 |
| P1 | `GET /users/{userId}/mutuals/friends` | 共同好友；详情优先使用 6 小时缓存，关系网由用户显式触发刷新 |
| P1 | `GET /users?search=...` | 用户搜索 |
| P1 | `GET /user/{userId}/friendStatus` | 好友关系状态 |
| P1 | `POST /user/{userId}/friendRequest` | 用户主动发送好友请求 |
| P1 | `DELETE /user/{userId}/friendRequest` | 取消请求 |
| P1 | `DELETE /auth/user/friends/{userId}` | 删除好友，必须二次确认 |
| P2 | `/userNotes` 系列 | VRChat 自带用户备注 |

本地备注和 VRChat 自带备注必须分开标识，不能让用户误以为本地内容已同步到 VRChat。

### 4.3 世界与实例

| 优先级 | 方法与端点 | 用途 |
|---|---|---|
| P0 | `GET /worlds?search=...` | 世界搜索 |
| P0 | `GET /worlds/{worldId}` | 世界详情；规范说明可匿名但部分字段为 0 |
| P0 | `GET /instances/{worldId}:{instanceId}` | 实例详情 |
| P0 | `GET /worlds/{worldId}/{instanceId}` | 世界实例视图 |
| P1 | `GET /instances/recent` | 最近位置 |
| P1 | `POST /invite/myself/to/{worldId}:{instanceId}` | 邀请自己加入实例 |
| P2 | `POST /instances` | 创建实例；需要完整权限与参数验证 |

实例标识含访问类型、区域、群组和 nonce。日志和 UI 展示必须使用统一解析器，不能用字符串随意 split 后直接泄漏私密实例参数。

### 4.4 收藏

| 优先级 | 方法与端点 | 用途 |
|---|---|---|
| P0 | `GET /favorite/groups` | 收藏分组 |
| P0 | `GET /favorites` | 收藏项 |
| P1 | `POST /favorites` | 加入收藏 |
| P1 | `DELETE /favorites/{favoriteId}` | 移除收藏 |
| P1 | `/favorite/group/...` | 查看/改名/清空分组 |
| P1 | `GET /auth/user/favoritelimits` | 收藏上限 |

### 4.5 通知和邀请

| 优先级 | 方法与端点 | 用途 |
|---|---|---|
| P1 | `GET /notifications` | Notification V2 列表 |
| P1 | `GET /auth/user/notifications` | 旧通知列表 |
| P1 | `POST /notifications/{id}/see` | 标记已读 |
| P1 | `POST /notifications/{id}/respond` | 响应通知 |
| P1 | `PUT /auth/user/notifications/{id}/accept` | 接受好友请求 |
| P1 | `POST /invite/{userId}` | 用户主动邀请好友 |
| P1 | `POST /requestInvite/{userId}` | 请求邀请 |
| P1 | `POST /users/{userId}/boop` | 向单个好友发送 Boop；请求体使用 `emojiId` |

旧版和 V2 通知应映射到统一 DTO，同时保留 `sourceVersion` 便于执行正确的响应端点。

邀请请求只发送实例标签部分作为 `instanceId`，依赖 Cookie 会话，不应附带伪造的 `Authorization`。Boop 首版仅开放内置 `default_` 表情和合法 `file_` 表情 ID；页面不提供批量发送，并在每次发送前确认。

### 4.6 群组

第二阶段先做只读：

- `GET /groups?query=...`；
- `GET /groups/{groupId}`；
- `GET /groups/{groupId}/instances`；
- `GET /groups/{groupId}/members`；
- `GET /groups/{groupId}/posts`。

管理、封禁、角色、公告和审计日志等写操作最后再做，且必须按当前用户权限动态显示，不能只靠前端隐藏按钮。

### 4.7 头像

MVP 只做用户可见范围内的详情和收藏：

- `GET /avatars/{avatarId}`；
- `GET /avatars/favorites`；
- `POST /favorites` / `DELETE /favorites/{favoriteId}`。

不做 AssetBundle 下载、不做 avatar ripping、不展示用户无权看到的私有头像。

## 5. 请求管线

> 实现批注：🟡 参数校验、会话、User-Agent、超时、状态码归一化、限流、429 冷却、GET 合并和好友/世界 SQLite 快照已实现；全局队列、精细 schema 漂移报告尚未实现。

每个请求依次经过：

```text
DTO request
  -> 参数校验
  -> 权限/用户动作校验
  -> 缓存与 GET 合并
  -> 账号级限流
  -> 全局连接数限制
  -> CookieJar + User-Agent
  -> VRChat
  -> 状态码归一化
  -> 响应 Schema 宽松校验
  -> 上游对象映射为内部 DTO
  -> 缓存/事件落库
```

### 5.1 GET 合并

相同账号、方法、规范化 URL 的并发 GET 共享一个 in-flight promise。完成后移除，不能永久缓存 promise。

### 5.2 缓存建议

| 数据 | 新鲜 TTL | 可离线保留 |
|---|---:|---:|
| 当前好友快照 | 30–60 秒 | 是 |
| 用户详情 | 5 分钟 | 是 |
| 世界详情 | 10–30 分钟 | 是 |
| 世界搜索页 | 1–5 分钟 | 是 |
| 实例详情 | 10–30 秒 | 短期 |
| 收藏分组 | 5 分钟 | 是 |
| 通知列表 | 30–60 秒 | 是 |
| API config | 10 分钟 | 是 |
| Pipeline token | 不缓存 | 否 |

写成功后按实体精确失效，禁止粗暴清空全部缓存造成请求风暴。

### 5.3 错误模型

本地 API 统一返回：

```json
{
  "error": {
    "code": "VRC_RATE_LIMITED",
    "httpStatus": 429,
    "retryable": true,
    "retryAfterMs": 120000,
    "message": "请求过于频繁，已自动暂停刷新",
    "requestId": "local-request-id"
  }
}
```

### 5.4 国内网络与图片链路

> 实现批注：🟡 已实现 `system/direct/http/socks5` 四种网络模式；自定义地址仅允许本机回环接口。REST、Pipeline 握手与图片下载复用同一个 Transport，避免三条链路从不同出口访问 VRChat。

前端不直接加载 VRChat 图片 URL。`GET /local/v1/media?url=...` 只接受 VRChat/VRC CDN 的 HTTPS 域名，拒绝 URL 凭据和 SVG，限制单图 16 MiB，成功后落盘复用。这样可以避免浏览器绕过网关代理，也能在线路波动时继续显示已缓存图片。

这里的“国内适配”不代表能够绕过 VRChat 的地区或 Cloudflare 策略；它保证应用没有境外前端 CDN 依赖，并允许用户选择自己的本机网络出口。

建议错误码：

- `VRC_AUTH_REQUIRED`
- `VRC_2FA_REQUIRED`
- `VRC_SESSION_EXPIRED`
- `VRC_RATE_LIMITED`
- `VRC_FORBIDDEN`
- `VRC_NOT_FOUND`
- `VRC_UPSTREAM_UNAVAILABLE`
- `VRC_RESPONSE_INCOMPATIBLE`
- `LOCAL_OFFLINE`
- `LOCAL_DATABASE_ERROR`
- `LOCAL_SECURITY_ERROR`

403 不等于账号被封；它也可能来自 Cloudflare、网络出口或资源权限。UI 不应吓用户，诊断页再展示技术细节。

## 6. 本地网关对前端的 API

> 实现批注：🟡 已实现认证、好友/关系网、世界/实例、本机与上游收藏、通知、邀请、活动、游戏日志状态、更新、网络、媒体和 SSE；仍未引入通用 command 幂等层。

建议本地契约：

```text
POST   /local/v1/auth/login
POST   /local/v1/auth/2fa
GET    /local/v1/auth/session
DELETE /local/v1/auth/session

GET    /local/v1/me
GET    /local/v1/friends
GET    /local/v1/friend-network
GET    /local/v1/users/{id}
PUT    /local/v1/profile             # 仅修改当前登录用户，提交前由 UI 二次确认
GET    /local/v1/users/{id}/mutual-friends
GET    /local/v1/worlds
GET    /local/v1/worlds/{id}
GET    /local/v1/instances/{location}
GET    /local/v1/favorites
GET    /local/v1/notifications
GET    /local/v1/groups?userId={currentUserId}
GET    /local/v1/groups/{groupId}/posts
GET    /local/v1/groups/{groupId}/instances
GET    /local/v1/groups/{groupId}/calendar?month=YYYY-MM
GET    /local/v1/avatars/favorites

GET    /local/v1/events/stream       # SSE，页面实时更新
GET    /local/v1/network
PUT    /local/v1/network             # 修改后重建 REST/Pipeline/图片 Transport
GET    /local/v1/media?url=...       # VRChat 图片白名单代理与磁盘缓存
GET    /local/v1/history
GET    /local/v1/diagnostics
POST   /local/v1/diagnostics/export
```

> 2026-08-15 批注：群组中心使用 VRChat `GET /users/{userId}/groups`、`GET /groups/{groupId}/posts`、`GET /groups/{groupId}/instances`；头像柜使用 `GET /avatars/favorites`。所有接口只读取当前会话有权看到的内容，失败时仅回退本机缓存，不尝试扩展权限。

> 2026-08-15 资料编辑批注：`PUT /local/v1/profile` 不接受用户 ID，网关只使用登录会话内记录的当前用户 ID。基础状态字段写入 VRChat `PUT /users/{currentUserId}`，简介与链接写入 `PUT /profile/{currentUserId}`，随后重新读取资料；输入按 VRChat 字段长度、状态枚举、链接数量和 URL 协议校验。若第二步失败，接口明确返回“基础资料已保存但简介失败”，前端不宣称事务性成功。

> 第三批补充：群组日历使用 VRChat `GET /calendar/{groupId}`，仅在用户选择具体群组时读取指定月份，最多 100 条并缓存 30 分钟。今晚罗盘、本机社交查询和重逢雷达不增加 VRChat 上游端点，全部基于现有好友快照、本机事件、世界缓存和关系图计算。

### 7.1 共同好友关系网边界

- `GET /local/v1/friend-network` 只读取好友快照和本机 SQLite，不主动访问上游；
- `GET /local/v1/users/{id}/mutual-friends?refresh=1` 才强制请求上游并更新该好友的边快照；
- Web 端一次最多串行处理 20 位好友，沿用全局平均 1 req/s 限流，可在当前请求结束后停止；
- 同一无向边只展示一份；已不在当前好友快照中的节点和边不展示；
- 上游 403/404 记录为“未共享/不可用”，不把空结果解释为不存在关系；
- 关系图是局部、随时间变化的本机观测结果，不是 VRChat 官方完整社交图。

用户动作单独建 command：

```text
POST /local/v1/commands/send-friend-request
POST /local/v1/commands/respond-notification
POST /local/v1/commands/invite-user
POST /local/v1/commands/add-favorite
POST /local/v1/commands/remove-favorite
```

command 必须包含客户端生成的 `idempotencyKey`，网关在短期内拒绝重复提交。

## 7. 凭据存储

> 实现批注：✅ Windows CurrentUser DPAPI + SQLite 已实现并通过本机往返测试。当前仅保存一个 `default` 会话，多账号隔离尚未实现。

### 7.1 首版默认

- 用户名可以保存；
- 密码不保存；
- CookieJar 用随机数据密钥加密；
- 数据密钥再由 Windows DPAPI 当前用户范围保护；
- 数据库只存密文、nonce、算法版本和更新时间；
- 备份/导出默认排除 `secure_session`。

### 7.2 浏览器存储禁止项

禁止写入 localStorage、sessionStorage、IndexedDB、URL 或 Service Worker cache：

- VRChat 密码；
- Basic Authorization；
- `auth`、`twoFactorAuth` Cookie；
- Pipeline token；
- 完整的私密实例 location（除非用户明确保存）。

## 8. 测试策略

> 实现批注：🟡 已有模拟上游登录/2FA、Cookie 重启复用、好友分页与缓存回退、世界搜索、Pipeline 消息解码、SQLite/DPAPI、本地安全中间件和前端 CSRF 测试；真实账号 smoke suite 未执行。

### 8.1 不接触真实账号的测试

- 所有 adapter 使用脱敏 JSON fixture；
- HTTP server 模拟 200、401、403、404、429、502、超时和畸形 JSON；
- CookieJar 序列化/加密/解密往返；
- Pipeline 重复、乱序、未知事件和断线；
- 日志轮转、半行写入和非法 UTF-8；
- DTO 缺字段、加字段和类型漂移。

### 8.2 测试账号烟测

只在手工、低频的 smoke suite 使用：

- 登录 + 所有 2FA 类型中实际可用的类型；
- Cookie 重启恢复；
- 一页好友；
- 一个已知世界详情；
- Pipeline 连接和一次状态变化；
- 退出。

禁止在 CI 使用真实账号，禁止把凭据放 GitHub Actions Secret 后自动高频跑。

## 9. 上游变更维护

- 每周或发布前检查 `vrchatapi/specification` diff；
- 记录当前适配的规范 commit；
- 生成的类型只能作为参考，领域 DTO 手工维护；
- 对端点增加 feature flag，可远程关闭单个坏接口；
- 未知字段保留但不透传给 UI；
- 兼容失败时继续提供离线缓存并显示“上游接口已变化”。

## 10. 上线前红线检查

- [ ] 没有 Photon 调用。
- [ ] 没有共享云端 CookieJar。
- [ ] 没有默认保存明文密码。
- [ ] User-Agent 格式和联系邮箱已配置。
- [ ] 429、403、401、网络失败走不同分支。
- [ ] Cookie 和 token 已加入所有日志脱敏器测试。
- [ ] 所有写操作由用户主动触发并防重复。
- [ ] 本地端口有 Origin/CSRF/随机 token 防护。
- [ ] 断网、重连、休眠唤醒和切换代理已测试。
- [ ] 更新包有哈希和签名验证。
