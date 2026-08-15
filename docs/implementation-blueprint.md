# VRC Web Companion 实施蓝图

> 状态：方案基线  
> 日期：2026-08-13  
> 参考项目：[VRCX](https://github.com/vrcx-team/VRCX)  
> 目标：在中国网络环境下尽量稳定、使用方便，并把账号封禁风险控制在可接受范围内的 VRCX 类 Web 应用。

> 实现批注（2026-08-13）：🟡 已进入编码阶段。阶段 1 工程骨架约完成 80%，阶段 2 仅完成登录/2FA/会话的代码主体；真实账号、Pipeline 和业务页面尚未验证。逐项证据见 [实现状态批注](implementation-status.md)。

## 1. 结论先行

建议做成 **浏览器 Web UI + Windows 本机网关**，第一版由本机网关直接托管前端页面，用户打开 `http://127.0.0.1:<port>` 使用。

这不是为了形式上的“桌面化”，而是由 VRChat 的接口形态决定：

1. VRChat 没有面向第三方应用的 OAuth；登录依赖 Basic Auth 换取 Cookie，并可能进入 2FA。
2. 登录后实时状态来自 `wss://pipeline.vrchat.cloud`，需要长期连接。
3. VRCX 的部分高价值信息来自本机 VRChat 日志，并不来自 Web API。
4. 浏览器无法可靠设置规范要求的自定义 `User-Agent`，跨域、Cookie 和 Cloudflare 也会阻碍纯前端直连。
5. 如果全部请求集中到云端，同一出口 IP、共享限流和账号会话集中保存会显著放大风控、泄漏和连坐风险。

因此，第一版应是“Web 技术做界面，本机程序做安全边界和网络适配”。以后可以增加云端入口，但 VRChat 请求仍由用户自己的在线网关执行。

### 1.1 对“国内不受限制”的准确解释

能保证的是我们自己这一段：中文 UI、本地启动、本地缓存、国内更新源、断网可读、失败重试和诊断清晰。

不能承诺 VRChat 上游永远可达。其 API、Pipeline、图片/CDN 和 Cloudflare 都不由本项目控制。正确策略是：

- 不让 UI 的可用性依赖 GitHub、Google Fonts 或境外前端 CDN；
- API 失败时展示缓存数据和明确状态，而不是整页不可用；
- 支持用户显式配置系统代理或单独的 HTTP/SOCKS 代理；
- 代理是可选网络能力，不内置公共代理池，不自动轮换 IP；
- 提供连通性检查，分别测试 API、Pipeline、图片和本地日志。

## 2. 从 VRCX 借鉴什么

调研的 VRCX 快照使用 Vue 3、Pinia、Vite、Electron/.NET 和 SQLite。真正值得复用的不是窗口外观，而是下面的结构：

- API 按资源拆分：用户、好友、世界、实例、收藏、群组、通知、邀请、头像等；
- REST 请求统一经过请求层，集中处理 Cookie、错误、GET 合并、失败冷却和调试日志；
- 先通过 `GET /auth` 获取 Pipeline token，再连接 `wss://pipeline.vrchat.cloud/?auth=...`；
- Pipeline 事件与定时 REST 同步互相校正；
- 本地 VRChat 日志提供进房、离房、视频 URL、资源加载等附加事件；
- SQLite 保存历史、备注、缓存和活动统计；
- UI 状态和上游原始对象分离。

### 2.1 不建议直接 Fork 后硬改

VRCX 已有大量 Electron、C# 桥接、Windows Overlay 和历史兼容逻辑。直接 Fork 会把与 Web 目标无关的复杂度一起带进来。

推荐做法：

- 参考它的 API 端点覆盖、事件语义、重连策略和数据表思路；
- 新项目从干净的接口适配层开始；
- 若复制具体 MIT 源码，保留 VRCX 的版权和许可证声明；
- UI、数据模型和网络层重新设计，以便将来替换 VRChat 上游字段。

## 3. 产品范围

### 3.1 MVP 必须有

1. 单账号登录、TOTP/邮件 2FA/恢复码。
2. 会话复用，重启后通常不需要再次输入密码。
3. 好友列表、在线状态、所在世界/隐私状态。
4. 用户详情、备注、标签和本地收藏。
5. 世界搜索、世界详情、实例详情、复制/启动加入链接。
6. VRChat 收藏夹：好友、世界、头像的读取和基础管理。
7. 邀请、好友请求、通知的查看与用户主动响应。
8. Pipeline 实时事件、断线重连和 REST 补偿同步。
9. 本机日志监听：当前世界、玩家加入/离开、视频 URL、会话历史。
10. SQLite 本地数据、图片缓存、数据导入导出。
11. 中文界面、浅色/深色主题、诊断页和一键脱敏日志导出。

### 3.2 第二阶段

- 多账号档案，但同一时刻默认只激活一个账号；
- 好友活动时间热力图、共同在线时长、重逢提醒；
- 群组浏览、成员与实例查看；
- 仪表盘组件和自定义过滤器；
- 截图与世界/实例元数据关联；
- PWA 安装、托盘常驻、浏览器通知；
- 端到端加密的备注/设置跨设备同步；
- 手机 Web 通过在线的家庭网关远程查看。

### 3.3 暂时不做

- Photon 或任何游戏内协议交互；
- Avatar/World AssetBundle 下载、解析或导出；
- 自动批量加好友、群发邀请、批量私信；
- 代用户上传世界、头像或类似资产；
- 集中式账号托管、公共代理池、IP 轮换；
- 以抓取量为目标的全站用户/世界镜像；
- Economy、支付、KYC 等高敏感接口；
- 无人值守的群组批量管理。

这些功能不是“永远不能做”，而是它们对首版价值不高，却会明显扩大风控和维护面。

## 4. 推荐技术栈

> 实现批注：🟡 Vue 3 + TypeScript + Vite 与 Go 网关已落地；SQLite、Windows DPAPI 已接入。Pinia、TanStack Query、ECharts、Zod、Playwright 尚未引入，因为对应业务模块尚未开始。

### 4.1 前端

- Vue 3 + TypeScript + Vite；
- Pinia：登录态、实时状态和 UI 状态；
- TanStack Query：服务端状态、缓存和请求失效；
- Vue Router；
- Tailwind CSS 或 UnoCSS；
- ECharts：活动热力图和时序统计；
- Zod：本地网关响应和 Pipeline 事件运行时校验；
- Vitest + Playwright：单测与端到端测试。

选择 Vue 是因为与 VRCX 技术栈接近，理解它的页面/状态逻辑更直接，也方便逐项对照行为。

### 4.2 本机网关

推荐 Go：

- 单个 EXE，无需用户安装 JRE 或 Node；
- `net/http`、CookieJar、WebSocket、文件监听和 SQLite 生态成熟；
- 可用 `go:embed` 把前端产物放进 EXE；
- 资源占用和启动速度适合托盘常驻；
- Windows、macOS、Linux 后续可共用大部分代码。

建议库（落地时再锁版本）：

- HTTP 路由：标准库或 chi；
- WebSocket：coder/websocket 或 gorilla/websocket；
- SQLite：modernc.org/sqlite（纯 Go）或 mattn/go-sqlite3；
- 日志：slog + 文件滚动；
- 系统密钥：Windows DPAPI，其他平台接系统 Keychain/Secret Service；
- 文件监听：fsnotify；
- 托盘：systray 类库或一个极薄的 Tauri/Wails 外壳。

### 4.3 为什么首版不选纯 Next.js 云服务

纯云服务开发看起来最快，但会遇到：

- 所有 VRChat API 请求来自少数服务器 IP；
- 每个账号的 Cookie、2FA 会话都必须存在服务器；
- Cloudflare/上游线路异常会影响所有人；
- WebSocket 按在线用户常驻，运维和成本快速上升；
- 一个服务漏洞可能泄漏全部账号会话；
- 上游封一个出口可能让全部用户同时失效。

小规模也不代表可以忽略这些单点问题。本机网关反而更接近 VRCX 已验证多年的运行方式。

## 5. 总体架构

> 实现批注：🟡 浏览器 → 回环地址网关 → VRChat REST 的主链路已实现；Pipeline、本地日志、图片缓存和可选云端均未实现。

```text
┌──────────────────────── 浏览器 ────────────────────────┐
│ Vue Web UI                                              │
│ 页面 / 查询缓存 / 实时状态 / IndexedDB 非敏感草稿       │
└───────────────────────┬─────────────────────────────────┘
                        │ http://127.0.0.1 + SSE/WebSocket
┌───────────────────────▼─────────────────────────────────┐
│ Local Gateway (Go)                                     │
│                                                        │
│ Local API ─ VRC Adapter ─ Rate Limit/Cache ─ CookieJar │
│     │             │                     │              │
│     │             ├─ Pipeline Client    ├─ DPAPI       │
│     │             └─ REST Client        └─ SQLite      │
│     └─ Game Log Watcher / Image Cache / Diagnostics    │
└──────────────┬──────────────────────┬───────────────────┘
               │ HTTPS/WSS            │ 本地文件
        VRChat API/Pipeline      VRChat logs/screenshots

可选云端（第二阶段）：静态更新源、版本清单、E2E 加密同步、
远程 UI 的加密消息中继；不保存 VRChat 密码和明文 Cookie。
```

### 5.1 进程边界

前端永远只访问本项目的本地 API，例如 `/local/v1/friends`，不直接拼 VRChat URL。网关负责把上游 JSON 转成稳定的内部 DTO。

这层隔离是项目能否长期维护的关键。VRChat API 未正式保证兼容，字段和端点变化时只改 adapter，不应该让几十个页面一起坏掉。

### 5.2 本地 API 的保护

> 实现批注：✅ 已实现仅回环地址、Origin、CSRF、CSP、`X-Frame-Options` 和 `nosniff`；🟡 当前端口固定为 `47831`，随机端口和启动器配对 token 尚未实现。

即使只绑定 `127.0.0.1`，也要防止恶意网页调用：

- 启动时生成随机端口和一次性配对 token；
- 严格校验 `Origin`；
- 写操作要求 CSRF token；
- CORS 只允许内置 UI origin；
- 默认拒绝 LAN 地址；
- 远程模式另走设备配对和端到端加密，不能简单监听 `0.0.0.0`；
- 日志中自动遮盖 Authorization、Cookie、Pipeline token、邮箱和位置参数。

## 6. 核心模块

### 6.1 `vrc-adapter`

> 实现批注：🟡 已实现统一 REST Client、User-Agent、CookieJar、登录/2FA、状态码归一化、探针、1 req/s token bucket 和 429 `Retry-After` 冷却；请求合并和领域 DTO 全覆盖尚未实现。

职责：

- 唯一知道 VRChat 基础 URL 和原始字段的模块；
- 登录、2FA、CookieJar、User-Agent；
- REST 请求、重试、限流、缓存和错误归一化；
- 将上游对象映射为内部 DTO；
- 保存上游规范版本和兼容开关。

不能让 UI 使用诸如 `travelingToLocation`、`location`、`platform` 等上游字段自行推断语义；这些判断统一放在 adapter/domain 层。

### 6.2 `presence-engine`

> 实现批注：⬜ 未实现，依赖好友快照和 Pipeline。

把三种来源合并成一个最终好友状态：

1. 首次 REST 好友快照；
2. Pipeline 增量事件；
3. 断线后的 REST 补偿快照。

每条状态携带 `source`、`observedAt` 和版本号，过期事件不能覆盖新状态。

### 6.3 `event-store`

> 实现批注：⬜ 未实现。目前数据库只有 migration、账号档案和安全会话表。

Pipeline 和游戏日志都先归一化为领域事件，再写入数据库：

```text
friend.online
friend.offline
friend.location.changed
friend.status.changed
friend.avatar.changed
relationship.added
relationship.removed
notification.received
instance.joined
instance.left
player.joined
player.left
video.played
```

事件要有稳定的幂等键，防止 WebSocket 重连和 REST 补偿造成重复历史。

### 6.4 `game-log-watcher`

> 实现批注：⬜ 未实现。

监听 `%USERPROFILE%\AppData\LocalLow\VRChat\VRChat` 下的新日志并从末尾增量读取。

要求：

- 记录文件 inode/路径、offset 和最后时间戳；
- 支持日志轮转和 VRChat 重启；
- 解析失败保留原始行的哈希和脱敏片段；
- 不修改游戏文件，不注入进程，不读取内存；
- 解析规则版本化并配 fixture 测试。

### 6.5 `sync-engine`

> 实现批注：⬜ 未实现，仍按第二阶段以后处理。

首版只做本地。第二阶段同步这些自有数据：

- 用户备注、世界备注、标签；
- 仪表盘布局、过滤器和偏好；
- 可选的统计聚合结果。

默认不同步：

- VRChat Cookie、密码、2FA token；
- 原始好友位置流水；
- 原始游戏日志和截图；
- 私密实例完整标识。

同步载荷用设备密钥端到端加密，服务端只保存密文、版本向量和冲突元数据。

## 7. 数据模型建议

> 实现批注：🟡 SQLite WAL、`schema_migration`、`account_profile`、`secure_session` 已实现并通过临时数据库测试；其余业务表尚未建立。

SQLite 开启 WAL，所有时间统一存 UTC ISO-8601，UI 再按用户时区展示。

### 7.1 核心表

```sql
account_profile(
  account_id TEXT PRIMARY KEY,
  display_name TEXT,
  avatar_url TEXT,
  last_login_at TEXT,
  active INTEGER NOT NULL DEFAULT 0
);

secure_session(
  account_id TEXT PRIMARY KEY,
  encrypted_cookie_jar BLOB NOT NULL,
  encryption_version INTEGER NOT NULL,
  updated_at TEXT NOT NULL
);

entity_cache(
  account_id TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  etag TEXT,
  fetched_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  PRIMARY KEY(account_id, entity_type, entity_id)
);

domain_event(
  event_id TEXT PRIMARY KEY,
  account_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  subject_id TEXT,
  occurred_at TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  source TEXT NOT NULL,
  payload_json TEXT NOT NULL
);

user_note(
  account_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  note TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(account_id, user_id)
);

entity_tag(
  account_id TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  color TEXT,
  PRIMARY KEY(account_id, entity_type, entity_id, tag)
);
```

还需要：`friend_snapshot`、`notification`、`instance_session`、`player_session`、`image_cache_index`、`request_metric`、`schema_migration`。

### 7.2 隐私保留策略

好友位置历史属于敏感程度较高的数据，不能无限期默认保存：

- 默认只保存 30 天原始事件；
- 30 天后仅保留按天聚合的在线时长；
- 用户可选择 7/30/90 天或不记录；
- 私密实例默认只保存世界 ID，不保存完整实例 nonce；
- 支持按好友、按日期和全部清除；
- 导出时再次提示数据包含好友活动信息。

## 8. 国内网络与离线体验

> 实现批注：🟡 Web 静态资源已全部本地打包，未引用境外运行时 CDN；VRChat `/config` 与 `/health` 诊断已实现。离线业务缓存、代理设置、图片 LRU 和国内更新源尚未实现。

### 8.1 静态资源

- 字体、图标、JS、CSS 全部随本机网关分发；
- 不依赖 Google Fonts、unpkg、jsDelivr；
- 更新清单可以放国内对象存储/CDN，GitHub Releases 只作备用；
- 发布包带 SHA-256 和签名，网关只接受可信签名更新；
- 软件可永久关闭自动更新。

### 8.2 VRChat API

- 连接超时、TLS 错误、403、401、429、5xx 分开显示；
- 首页采用 stale-while-revalidate：先显示本地最后快照，再后台刷新；
- 图片采用磁盘 LRU 缓存，避免每次重新下载；
- GET 同 URL 并发合并；
- 失败不清空已有数据；
- Proxy 只作为用户设置，支持“跟随系统”和“自定义地址”；
- 切换代理后必须主动重建 HTTP 连接池、Cookie 域仍保持 VRChat 原域；
- 诊断页展示实际出口变化提醒，避免短时间频繁变更登录 IP。

调研当天从当前环境请求 `/api/1/config` 和 `/api/1/health` 得到 Cloudflare 403。这只说明必须把 403/线路/Cloudflare 当成正常故障类型设计，不能据此断言所有中国线路都不可用。

### 8.3 Pipeline

- 指数退避：1s、2s、4s、8s，最大 60s，并加入 20% jitter；
- 网络恢复时不要所有客户端整点同时重连；
- 连续失败后切到“快照模式”，降低频率做 REST 补偿；
- 重连成功立即做一次好友快照校正；
- UI 明确显示“实时”“延迟”“离线缓存”三种状态。

## 9. 降低账号风险的工程规则

> 实现批注：🟡 已实现本机出站、稳定 User-Agent 配置入口、密码不落库、DPAPI Cookie 会话、日志不记录请求机密、基础 token bucket 与 429 冷却；写操作幂等尚未实现，相关业务写端点暂未开放。

目标不是零风险，而是避免最容易导致账号封禁或整批用户受影响的行为。

### 9.1 强制规则

- 每个请求使用清晰、稳定的 `User-Agent: AppName/version contact`；
- 账号请求从用户本机发出，不汇聚到共享云出口；
- 一个账号只有一个有效 CookieJar，不在每次启动时重新用密码创建会话；
- 默认不保存明文密码；若以后允许“记住密码”，必须是明确选项并用系统密钥加密；
- 遇到 429 立即停发该类请求，服从 `Retry-After`，没有时指数退避；
- 所有循环加 jitter，禁止整点同步；
- 写操作只由用户点击触发，带二次确认和幂等保护；
- 不接触 Photon，不改客户端，不注入 VRChat 进程；
- 不批量抓取非好友用户，不做隐藏状态推断；
- 日志和崩溃报告禁止包含 Cookie、Authorization 和 Pipeline token。

### 9.2 默认限流建议

上游没有承诺固定额度，所以这些只是客户端自我保护值，不代表 VRChat 官方限额：

- 普通读取：每账号平均不超过 1 req/s，短突发 5；
- 搜索：用户停止输入 400–600ms 后请求，最少 2 秒一次；
- 写操作：串行，每次操作完成后再允许下一次；
- 好友全量快照：启动、Pipeline 恢复或人工刷新时触发，不做高频固定轮询；
- 429 后：至少按 `Retry-After`，否则 2s 起指数退避，上限 15 分钟；
- 同一失败 GET：短期负缓存，403/404 默认 15 分钟内不重复撞击。

这些值应通过配置下发和遥测观察微调，但不能提供“极速/无限制”开关。

## 10. 开发路线

### 阶段 0：技术探针（2–4 天）

> 当前：🟡 进行中。REST 诊断、登录/2FA/Cookie 代码已完成；真实账号烟测、Pipeline 2 小时运行和日志轮转未完成。

只用开发者自己的测试账号：

- 验证目标网络上的 `/config`、`/auth/user`、2FA、Cookie 复用；
- 验证 `GET /auth` 和 Pipeline 事件类型；
- 记录 401/403/429/5xx 的实际响应；
- 验证 VRChat 日志路径和关键行格式；
- 得到一份脱敏 fixture，禁止把真实 Cookie 提交到 Git。

退出条件：登录重启可复用、Pipeline 能稳定运行 2 小时、日志轮转可恢复。

### 阶段 1：工程骨架（3–5 天）

> 当前：🟡 约 80%。代码结构、内嵌前端、本地安全中间件、SQLite、日志、诊断和本地构建已完成；随机端口启动器与远程 CI 未完成。

- `apps/web`、`apps/gateway`、`packages/contracts`；
- Go 内嵌 Vue 构建产物；
- 本地随机端口、Origin 校验、CSRF；
- SQLite migration、结构化日志和诊断页；
- CI 执行 Go test、前端 lint/typecheck/test。

### 阶段 2：可用 MVP（2–3 周）

> 当前：🟡 主体推进中。登录、2FA、会话恢复、好友/用户详情、世界、Pipeline、图片缓存和共同好友关系网已有首版；实例、收藏、本地备注与导入导出未完成。

- 登录/2FA/退出/会话恢复；
- 当前用户、好友、世界、实例、收藏；
- Pipeline 与好友状态合并；
- 图片缓存、搜索、错误状态；
- 本地备注和导入导出。

退出条件：每天自用，不需要手工清 Cookie；网络短断后自动恢复；429 不会形成重试风暴。

### 阶段 3：VRCX 核心体验（2–4 周）

> 当前：⬜ 未开始。

- 通知/邀请；
- 游戏日志历史；
- 活动热力图；
- 群组和实例；
- 仪表盘、筛选器、系统通知；
- 安装包、签名更新和国内镜像。

### 阶段 4：小范围 Beta（2 周以上）

> 当前：⬜ 未开始。

- 5–20 名用户灰度；
- 只收集脱敏请求指标和版本信息，并允许完全关闭；
- 观察登录失败、403、429、Pipeline 重连、数据库损坏和升级回滚；
- 建立上游 API 变更的兼容测试与紧急开关。

一个开发者从零到稳定 Beta，合理预期约 8–12 周；先做技术探针可避免把数周时间浪费在错误的纯 Web 架构上。

## 11. 建议仓库结构

```text
vrc-web-companion/
├─ apps/
│  ├─ web/                    # Vue 3 Web UI
│  └─ gateway/                # Go 本机网关
│     ├─ cmd/gateway/
│     └─ internal/
│        ├─ localapi/
│        ├─ vrchat/
│        ├─ pipeline/
│        ├─ ratelimit/
│        ├─ cache/
│        ├─ gamelog/
│        ├─ storage/
│        ├─ security/
│        └─ diagnostics/
├─ packages/
│  ├─ contracts/              # OpenAPI/JSON Schema/TS 类型
│  └─ fixtures/               # 脱敏上游响应和日志样本
├─ docs/
├─ scripts/
├─ LICENSES/
└─ README.md
```

## 12. 第一批验收标准

- 用户输入账号和密码时只发送给本机网关；
- 密码不会写入 SQLite、日志、崩溃报告和浏览器存储；
- CookieJar 使用系统密钥加密后持久化；
- 重启应用可用旧 Cookie 恢复，大多数情况下不新建登录会话；
- Pipeline 断开 10 次不会产生紧密重连循环；
- 429 会冻结对应请求队列并给用户明确提示；
- 断网后仍能打开最近好友、世界、历史和备注；
- 上游字段缺失不会导致整个页面白屏；
- 本机端口不能被任意网页跨域写操作；
- 日志导出经过自动脱敏测试；
- 没有 Photon、进程注入、资产下载或自动批量行为。

## 13. 需要在编码前确定的产品决策

1. 首版仅 Windows，还是同步支持 macOS/Linux；建议先 Windows。
2. 是否允许用户选择“记住密码”；建议首版不保存密码，只保存 Cookie。
3. 本地历史默认保留 30 天是否合适。
4. 是否首版就需要手机远程访问；建议推迟到本机 MVP 稳定后。
5. 项目是否开源；若开源，建议 MIT/Apache-2.0，并明确与 VRChat/VRCX 无官方隶属关系。

## 14. 资料来源

- [VRCX 仓库及功能列表](https://github.com/vrcx-team/VRCX)
- [VRChat Creator Guidelines：API Usage / Bots](https://hello.vrchat.com/creator-guidelines)
- [VRChat Terms of Service](https://hello.vrchat.com/legal)
- [社区维护的 VRChat API 文档](https://vrchat.community/)
- [社区 OpenAPI 规范仓库](https://github.com/vrchatapi/specification)
- [VRChat 官方 Launch Options](https://docs.vrchat.com/docs/launch-options)
