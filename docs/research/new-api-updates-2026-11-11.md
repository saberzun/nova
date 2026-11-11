# new-api 近期更新评估（2026-11-11）

范围：QuantumNous/new-api main，2026-10-10 至 2026-11-11，共 189 条提交。API 本次读取头部：`579573ac74513736e83842b6ed243c261de16dd4`。
本地基线：e1c400eae3527e60163632c58485dfa5e94932a1（2026-09-20），含大量定制及未提交修改。检查的是本地现有代码；未逐文件重新比对正式容器。
本次仅研究与生成清单，无代码修改、数据库修改或部署。
上游已使用 relaykit，本地仍有 dto 等旧布局；建议按行为移植，不整体覆盖。

## 优先评估项

### 48a6c87f9e fix(oauth): persist inviter when creating OAuth users (#8247)
https://github.com/QuantumNous/new-api/commit/48a6c87f9e7757fdc58e5be9bcb4f8281ad0dfd0

第二批：OAuth 注册持久化邀请人。本地 OAuth 计算 inviterId 但未赋 user.InviterId；需结合自定义邀请返现回归。

涉及：controller/oauth.go

### ada15c050e fix(channel): keep a manual channel disable when a request fails
https://github.com/QuantumNous/new-api/commit/ada15c050eb7e0b25513e5273b797cc3e4ee49f4

第二批：请求失败不覆盖管理员手动停用渠道。当前本地无对应状态保护。

涉及：model/channel.go

### 433830b4e2 fix(channel): restore a multi-key channel when a key is enabled after all keys failed
https://github.com/QuantumNous/new-api/commit/433830b4e201c5eb091f06179fd8de3839db25dc

条件更新：多 Key 全部失效后启用其中一个应恢复渠道。需先核实本地多 Key 管理前置实现，不能孤立拣这个提交。

涉及：controller/channel.go

### a5a4659266 fix(vertex): cache service-account access tokens until they expire
https://github.com/QuantumNous/new-api/commit/a5a46592665e840181e304f29b015a132ed043ac

条件更新：Vertex 服务账号令牌缓存。仅使用该认证方式时有直接收益。

涉及：relay/channel/vertex/service_account.go

### 2de51a19e3 fix(cache): 修复用户和令牌缓存的 Redis 跨槽错误 (#8218)
https://github.com/QuantumNous/new-api/commit/2de51a19e35a602a0041d380b29be7c9bcd1e0e5

条件更新：Redis Cluster 跨槽。目前已知拓扑为单实例 Redis，不是当前优先问题；若迁移 Cluster，按整套缓存版本协调升级。

涉及：model/token_cache.go, model/user_auth_cache.go

### 706144769e fix(auth): 校验 Redis 鉴权缓存完整性 (#8200)
https://github.com/QuantumNous/new-api/commit/706144769e5bed5a47b78721e3bb11da3cf4c0b9

第二批安全专项：Redis 用户/令牌/会话鉴权缓存增加 MAC 与有效期校验。本地无 auth_cache_integrity.go；涉及缓存格式、并发修复和额度保护，需独立回归，不表示已遭缓存篡改。

涉及：model/auth_cache_integrity.go, model/token.go, model/token_cache.go, model/user.go, model/user_auth_cache.go, model/user_cache.go, model/user_session.go

### 9b473a3fe4 fix(twofa): restart the failure count after a lockout expires (#8172)
https://github.com/QuantumNous/new-api/commit/9b473a3fe4fed8d18980fd8d3c6a8908847a5f25

第二批：2FA 锁定到期后失败计数重新开始，本地未重置。

涉及：model/twofa.go

### 77436271f4 fix(deps): upgrade DOMPurify to 3.4.17 (#8116)
https://github.com/QuantumNous/new-api/commit/77436271f4792f99a293b87ed7ab8d2606a411b3

第一批：DOMPurify 升级。本地固定 3.4.11，上游提交升至 3.4.17；文档 Markdown 渲染直接依赖此库。需要检查依赖公告和回归用例，不能仅凭升级标题断言存在可利用漏洞。

涉及：THIRD-PARTY-LICENSES.md, web/bun.lock, web/package.json

### 3150dfea9c chore(deps): bump github.com/go-jose/go-jose/v4 from 4.1.4 to 4.1.6 (#8112)
https://github.com/QuantumNous/new-api/commit/3150dfea9cbe7244914291f8724a45c71bc02b85

条件更新：go-jose 上游依赖升级，本地 go.mod 未找到该模块，先确认依赖图，不能直接加入。

涉及：go.mod, go.sum

### a3851c29e1 fix: save Turnstile keys before enabling (#7960)
https://github.com/QuantumNous/new-api/commit/a3851c29e1640d275b47af9ec141d64ae7325d65

第二批：先保存 Turnstile 密钥再启用，避免启用校验时尚未有密钥。当前本地未排序。

涉及：web/src/features/system-settings/auth/__tests__/bot-protection-settings.test.tsx, web/src/features/system-settings/auth/bot-protection-section.tsx

### 4193734a2a fix(logger): keep gin access and recovery logs on the current log file after rotation (#8098)
https://github.com/QuantumNous/new-api/commit/4193734a2abf2764375f29c26be8f540e9b9d4db

第二批：日志轮转后 logger/recovery 跟随新 writer。本地仍启动时捕获 writer。

涉及：common/sys_log.go, main.go, middleware/logger.go, middleware/recover.go

### 54788117d6 fix: count completed non-stream relays after context cancellation (#7844)
https://github.com/QuantumNous/new-api/commit/54788117d6574edcabbff2aa61d838f4d85616eb

可与流式统计修复一起评估：已完成的非流式请求不因随后取消而被误判失败。性能统计职责与计费分别验证。

涉及：pkg/perf_metrics/outcome.go

### 9823ebc7ea fix(billing): bill cache_creation_input_tokens reported in prompt_tokens_details (#7909)
https://github.com/QuantumNous/new-api/commit/9823ebc7ea4ac7102d5d8478d395c7d7eaf6c285

第一批计费专项：识别 prompt_tokens_details.cache_creation_input_tokens；本地缺少这个字段。与多个缓存字段取最大值而非相加一起移植。

涉及：relaykit/dto/openai_response.go, relaykit/dto/usage_merge.go, relaykit/relayconvert/internal/oai_chat/to_oai_responses_resp.go

### 30748c6558 fix(logging): 隐藏访问日志中的查询凭据 (#7972)
https://github.com/QuantumNous/new-api/commit/30748c6558bc97de49ff74871cfec717e50f3322

第一批：访问日志查询凭据脱敏。本地 logger 直接输出 param.Path，缺少对应脱敏分支。

涉及：middleware/logger.go

### 94cc3ff090 fix(billing): bill Claude cache writes that arrive without a 5m/1h split
https://github.com/QuantumNous/new-api/commit/94cc3ff0906db37703d5ce2bc6a479209b7b9b23

第一批计费专项：Claude 只返回缓存写入总量时补齐 5m/1h 分配。本地 Anthropic 分支覆盖为拆分字段，缺少余量归一化。

涉及：service/tiered_settle.go

### 7e6596728f fix(relay): stop cutting off slow clients during large stream events
https://github.com/QuantumNous/new-api/commit/7e6596728ff640392bd77169ebb6137a8fe0c309

第二批：大流事件按字节预算写入超时，避免慢客户端被固定超时截断。需连同写错误和取消行为测试。

涉及：controller/plugin_protocol.go, controller/relay.go, relay/helper/common.go, relay/helper/stream_scanner.go, relay/responses_websocket.go

### 0c2af9780e fix(relay): finish delivered Responses streams without false disconnect errors (#7956)
https://github.com/QuantumNous/new-api/commit/0c2af9780e45f6f9073528fb0fcbdcd64b5d9495

第二批：Responses 完成流不再误报断连，传播真实写入错误。

涉及：common/custom-event.go, relay/channel/openai/helper.go, relay/channel/openai/relay_responses.go, relay/helper/common.go, relay/helper/stream_result.go, relay/helper/stream_scanner.go

### 7d8b91268d fix: upgrade Go to 1.26.10 for CVE-2026-111943 (#7942)
https://github.com/QuantumNous/new-api/commit/7d8b91268dab69d7150f345c448b5e250163146a

构建工具链专项：上游以安全修复为由升 Go 1.26.10；本地 Dockerfile 仍 1.26.1。需独立核验公告与固定镜像、编译回归后升级；不把提交标题当漏洞验证。

涉及：.github/workflows/electron-build.yml, .github/workflows/release.yml, Dockerfile, Dockerfile.dev, go.mod

### e2a0c4b16a fix(limiter): recover redis script after flush (#6486)
https://github.com/QuantumNous/new-api/commit/e2a0c4b16aa6c2e365f0845cf073c52b933385a2

第一批：限流 EvalSha 遇 NOSCRIPT 时恢复。已确认本地缺少恢复逻辑。

涉及：common/limiter/limiter.go

### 09109ec315 fix(middleware): 过载哨兵改用容器口径并加入连续性与新鲜度判定 (#7317)
https://github.com/QuantumNous/new-api/commit/09109ec315a5086366f94bcce00f177ec831dbe1

条件更新：容器资源口径、连续超阈值和过期采样判断；适合同 VPS 多容器，仅过载哨兵启用时影响拒绝请求。不能替代边缘 DDoS 防护。

涉及：common/container_usage.go, common/system_monitor.go, middleware/performance.go

### 45fbb149f0 fix(rate-limit): count each group with its own limit separately
https://github.com/QuantumNous/new-api/commit/45fbb149f0c89ff48d805335f22419be392d0350

第一批：配置独立限额的分组使用用户+分组计数。本地仍按用户共用计数；对多订阅/分组业务有意义。

涉及：middleware/model-rate-limit.go

### 1b63c28dcc fix(claude): preserve explicit zero usage and cache-only billing (#7816)
https://github.com/QuantumNous/new-api/commit/1b63c28dcc999c4445c70f21ec90103ab49e8be5

第一批计费专项：保留显式零 usage 并处理纯缓存请求。上游修复需对照本地定制结算路径验证，不照抄整个 service/text_quota。

涉及：relay/channel/claude/relay-claude.go, service/text_quota.go

### 767a82b876 fix(claude): forward thinking.block_binding and every anthropic-beta value (#7732)
https://github.com/QuantumNous/new-api/commit/767a82b8764778b847462ebecdc231eacd1e282c

第一批：Claude thinking.block_binding、未知字段及多行 beta 请求头。已确认本地缺失；与本次签名错误相关，但尚未证明为同一根因。

涉及：relay/channel/aws/dto.go, relay/channel/claude/adaptor.go, relaykit/dto/claude.go

## 暂缓
插件架构、视频插件、全套定价 UI、moego/moejs 性能重构及 Electron 更新不作为当前第一批。DeepSeek、Vertex、Gemini、图像新参数按实际渠道启用情况筛选。RSA 易支付属于新增支付集成能力，需支付服务商支持与验签回归后单独接入。SQLite WAL 修复对目前 PostgreSQL 部署不直接适用。

## 完整提交索引

- 2026-11-11 [579573ac74](https://github.com/QuantumNous/new-api/commit/579573ac74513736e83842b6ed243c261de16dd4) feat(payment): 易支付支持 RSA 签名方式 (#8291)
- 2026-11-10 [0409deccff](https://github.com/QuantumNous/new-api/commit/0409deccfff2b0e8babc28ca552af44d526375bf) fix(deepseek): map developer role to system for chat completions (#8287)
- 2026-11-09 [bd6bb53019](https://github.com/QuantumNous/new-api/commit/bd6bb53019fb7e21804b31ab7365c9b2097f4b6c) feat: support HOST env to set the listen address (#8269)
- 2026-11-08 [8faf1d45e8](https://github.com/QuantumNous/new-api/commit/8faf1d45e8c101974dc7483324d3d91c8a462ffe) fix(web): translate legal consent text on auth pages (#8254)
- 2026-11-08 [48a6c87f9e](https://github.com/QuantumNous/new-api/commit/48a6c87f9e7757fdc58e5be9bcb4f8281ad0dfd0) fix(oauth): persist inviter when creating OAuth users (#8247)
- 2026-11-08 [e8298df63f](https://github.com/QuantumNous/new-api/commit/e8298df63f3a0c366029fbadce8236c89ad18909) feat(channels): 高级自定义渠道增加 DeepSeek 搜索与抓取路径选项 (#8250)
- 2026-11-07 [dd50ee0a56](https://github.com/QuantumNous/new-api/commit/dd50ee0a567b810ea60e74dcd9ea6a5786b37b91) feat(plugins): add gemini omni 1.2 flash and send every input image to vertex (#8228)
- 2026-11-07 [a3fd673bb7](https://github.com/QuantumNous/new-api/commit/a3fd673bb7897691b1041aaf8d78bab6cc15e1d5) feat(deepseek): 补全 Responses 工具历史缺失的 reasoning (#8162)
- 2026-11-07 [9c2c370a61](https://github.com/QuantumNous/new-api/commit/9c2c370a6156b3ac41874231952a938891ac5483) fix(sqlite): keep the default SQLite settings when SQLITE_PATH is a plain path
- 2026-11-07 [ada15c050e](https://github.com/QuantumNous/new-api/commit/ada15c050eb7e0b25513e5273b797cc3e4ee49f4) fix(channel): keep a manual channel disable when a request fails
- 2026-11-07 [433830b4e2](https://github.com/QuantumNous/new-api/commit/433830b4e201c5eb091f06179fd8de3839db25dc) fix(channel): restore a multi-key channel when a key is enabled after all keys failed
- 2026-11-06 [8c8940c999](https://github.com/QuantumNous/new-api/commit/8c8940c999c6c38031dacdd0139cf7b0e1c38308) fix(plugins): serve uploaded plugins at top-level /v1 paths (#8241)
- 2026-11-06 [e034390332](https://github.com/QuantumNous/new-api/commit/e03439033238722c0b8d4437a052445f3aeeed83) refactor(aws): remove the unused api_key HTTP request path
- 2026-11-06 [a5a4659266](https://github.com/QuantumNous/new-api/commit/a5a46592665e840181e304f29b015a132ed043ac) fix(vertex): cache service-account access tokens until they expire
- 2026-11-06 [6fa80abc70](https://github.com/QuantumNous/new-api/commit/6fa80abc70890c6c03a16f45db3d16a68696f1e9) fix: clarify session cache actions and expose per-rule clearing
- 2026-11-05 [2de51a19e3](https://github.com/QuantumNous/new-api/commit/2de51a19e35a602a0041d380b29be7c9bcd1e0e5) fix(cache): 修复用户和令牌缓存的 Redis 跨槽错误 (#8218)
- 2026-11-05 [72f3dc7daa](https://github.com/QuantumNous/new-api/commit/72f3dc7daaeba86280b01e2d4d5d745df5c7c67b) fix(pricing): review model price changes before saving
- 2026-11-05 [e3e74205fe](https://github.com/QuantumNous/new-api/commit/e3e74205fe57f1226716de6f84cb4f0cf80cee89) fix(pricing): guide users through model pricing issues
- 2026-11-05 [3374f251e3](https://github.com/QuantumNous/new-api/commit/3374f251e3a84eebfd7f37d4746d7bb017e4f4b1) feat: forward image complexity and markup for gpt-image-2.6 (#8208)
- 2026-11-04 [706144769e](https://github.com/QuantumNous/new-api/commit/706144769e5bed5a47b78721e3bb11da3cf4c0b9) fix(auth): 校验 Redis 鉴权缓存完整性 (#8200)
- 2026-11-03 [ba5f736779](https://github.com/QuantumNous/new-api/commit/ba5f7367797566175a6cd97a748d430af3602a01) fix(gemini): bill embeddings from upstream usageMetadata (#8182)
- 2026-11-03 [94b5216645](https://github.com/QuantumNous/new-api/commit/94b521664599d416fead8e57d8318717c0e29058) feat(vertex): add per-channel v1beta1 endpoint switch (#8179)
- 2026-11-03 [83b18fe6cc](https://github.com/QuantumNous/new-api/commit/83b18fe6cc3fcf489fb64772445c8587d71251f1) feat(channel): return created IDs with input mappings (#8164)
- 2026-11-03 [52c173038e](https://github.com/QuantumNous/new-api/commit/52c173038e3551618ef7c62f2e4db62ca43a8f69) feat(gemini): add Argon fields and reasoning mappings (#8177)
- 2026-11-03 [9b473a3fe4](https://github.com/QuantumNous/new-api/commit/9b473a3fe4fed8d18980fd8d3c6a8908847a5f25) fix(twofa): restart the failure count after a lockout expires (#8172)
- 2026-11-02 [45746eee4d](https://github.com/QuantumNous/new-api/commit/45746eee4de25ab1eb75d4b7fbb8de2199483ea3) feat(relay): add opt-in HTTP/2 health checks (#8059)
- 2026-11-02 [c046559bbd](https://github.com/QuantumNous/new-api/commit/c046559bbd8b154e88e69905f60e71a5f4f1e5aa) feat(plugins): bill gemini omni by the tokens vertex reports (#8153)
- 2026-11-02 [651d5a213f](https://github.com/QuantumNous/new-api/commit/651d5a213fc81e7447e88cd35364695ee664bf09) fix(xai): preserve token details in stream usage conversion (#6420)
- 2026-11-01 [70e6af6429](https://github.com/QuantumNous/new-api/commit/70e6af6429824f99de98e96e65254c8c2ef7ee0b) perf: read and edit request JSON and base64 with moego (#8148)
- 2026-11-01 [d0abbf6e0d](https://github.com/QuantumNous/new-api/commit/d0abbf6e0d33552a41c402d9b1fadbaff27cf40d) feat(plugins): generate video with Gemini Omni on Vertex AI channels (#8142)
- 2026-10-31 [ea343c4a81](https://github.com/QuantumNous/new-api/commit/ea343c4a81151bfd537d33b66527c25fa21a1112) fix(plugins): charge relayed streams even when the reserve or task insert fails (#8121)
- 2026-10-31 [fe57cba425](https://github.com/QuantumNous/new-api/commit/fe57cba425b51e113d26333185a2266e448c7651) fix(plugins): prefer Kling's returned price (list_price) for Kling 4.0 cash billing (#8125)
- 2026-10-30 [1424ffdd4a](https://github.com/QuantumNous/new-api/commit/1424ffdd4aef28812a0e9309a465453cbdd034e4) fix(plugins): load plugins that declare their own /{key} address (#8119)
- 2026-10-30 [8b253de42d](https://github.com/QuantumNous/new-api/commit/8b253de42d2820d2772be8cab3fbd31c31cfecf3) fix(plugins): keep Kling 4.0 durations in range and map every video size (#8118)
- 2026-10-30 [c59f6fd553](https://github.com/QuantumNous/new-api/commit/c59f6fd553921702cb47dacadc9731fa01548e96) feat(plugins): serve task plugins at the vendors' own paths and answer as the vendors do (#8117)
- 2026-10-30 [37d23dcc86](https://github.com/QuantumNous/new-api/commit/37d23dcc86b8eef4972a867582859a85d770c56c) fix(profile): honor global sidebar visibility (#7429)
- 2026-10-30 [77436271f4](https://github.com/QuantumNous/new-api/commit/77436271f4792f99a293b87ed7ab8d2606a411b3) fix(deps): upgrade DOMPurify to 3.4.17 (#8116)
- 2026-10-30 [883eeb87d8](https://github.com/QuantumNous/new-api/commit/883eeb87d864a0d657b39ff3703e057ebfaa523b) feat(plugins): warn about task models a plugin cannot charge
- 2026-10-30 [1e8c707aca](https://github.com/QuantumNous/new-api/commit/1e8c707acae93377336a5cc24f167a3e116fb13b) chore(deps): bump @codemirror/language from 6.12.4 to 6.13.1 in /web (#8102)
- 2026-10-30 [6e78cb3288](https://github.com/QuantumNous/new-api/commit/6e78cb3288bb07d2c3972d83776c8aeeaf6358eb) chore(deps): bump marked from 18.0.6 to 18.1.2 in /web (#8103)
- 2026-10-30 [9fc70bb834](https://github.com/QuantumNous/new-api/commit/9fc70bb83451a8c3ffcfcb17c71bb6706c4b52a6) chore(deps-dev): bump @tanstack/react-query-devtools in /web (#8104)
- 2026-10-30 [cef78205e8](https://github.com/QuantumNous/new-api/commit/cef78205e86b1d17cfe79b532ee04cc3aeea93cb) chore(deps-dev): bump knip from 6.27.0 to 6.43.0 in /web (#8105)
- 2026-10-30 [1695171fa2](https://github.com/QuantumNous/new-api/commit/1695171fa2b14bdafe91bec3b9b6bed1ed367c11) chore(deps): bump tailwind-merge from 3.6.0 to 3.7.0 in /web (#8106)
- 2026-10-30 [7fa815edb2](https://github.com/QuantumNous/new-api/commit/7fa815edb29257d14fe9d4cf7d8672d5e8173d27) chore(deps-dev): bump fast-uri from 3.1.7 to 3.1.9 in /electron (#8007)
- 2026-10-30 [ee0f0ac78b](https://github.com/QuantumNous/new-api/commit/ee0f0ac78b5395099a8ebb9fa95e2d4bdb8038bb) chore(deps-dev): bump electron from 41.10.6 to 42.11.16 in /electron (#8008)
- 2026-10-30 [837b8145f8](https://github.com/QuantumNous/new-api/commit/837b8145f875ec2060ede30afa97fa0814367837) chore(deps): bump undici in /electron (#8006)
- 2026-10-30 [3150dfea9c](https://github.com/QuantumNous/new-api/commit/3150dfea9cbe7244914291f8724a45c71bc02b85) chore(deps): bump github.com/go-jose/go-jose/v4 from 4.1.4 to 4.1.6 (#8112)
- 2026-10-30 [a3851c29e1](https://github.com/QuantumNous/new-api/commit/a3851c29e1640d275b47af9ec141d64ae7325d65) fix: save Turnstile keys before enabling (#7960)
- 2026-10-30 [4193734a2a](https://github.com/QuantumNous/new-api/commit/4193734a2abf2764375f29c26be8f540e9b9d4db) fix(logger): keep gin access and recovery logs on the current log file after rotation (#8098)
- 2026-10-30 [f3660175e0](https://github.com/QuantumNous/new-api/commit/f3660175e06eb05e609d56f1627c160a5a52172e) Merge commit from fork
- 2026-10-30 [829f0828a5](https://github.com/QuantumNous/new-api/commit/829f0828a5a9a099f24068d21b99411750ff8037) fix: unmount authenticated layout on sign-out to stop credential-less refetches (#8110)
- 2026-10-30 [9eda50a32f](https://github.com/QuantumNous/new-api/commit/9eda50a32fedb9122531d0c7ca0112dd8efdb9a0) Merge commit from fork
- 2026-10-29 [a2defd2a82](https://github.com/QuantumNous/new-api/commit/a2defd2a8256abe039e5295bd583aeb2d02f111f) Merge commit from fork
- 2026-10-28 [e449cc700e](https://github.com/QuantumNous/new-api/commit/e449cc700eb569cdfc2199cb73e97de46944f43c) feat: sample immediate task successes in model performance metrics (#8067)
- 2026-10-28 [fcc241cdf8](https://github.com/QuantumNous/new-api/commit/fcc241cdf8ae0a7ba54107a256b5b0167d5a4fdc) fix: start affinity fallback at the highest channel priority (#8078)
- 2026-10-28 [b08ee86fe2](https://github.com/QuantumNous/new-api/commit/b08ee86fe2a899dcf9060d8cde7858c93c2b770c) fix(cloudflare): preserve embedding response data (#8041)
- 2026-10-28 [26c488c447](https://github.com/QuantumNous/new-api/commit/26c488c4476aa7d8f7e8afc33421b70ee8e9f617) fix(relay): allow Azure channels to opt in to Responses WebSocket (#8047) (#8075)
- 2026-10-28 [55515dbae2](https://github.com/QuantumNous/new-api/commit/55515dbae24179e7dd9ba332adfa43c2961521f5) fix(relay): terminate streams on exact bare done markers (#8074)
- 2026-10-28 [49271d2d8b](https://github.com/QuantumNous/new-api/commit/49271d2d8b5fd85c6d69e3947594ac79804ba31f) fix(aws): preserve safeguards in Bedrock Invoke requests (#8071)
- 2026-10-28 [54788117d6](https://github.com/QuantumNous/new-api/commit/54788117d6574edcabbff2aa61d838f4d85616eb) fix: count completed non-stream relays after context cancellation (#7844)
- 2026-10-28 [ad56112820](https://github.com/QuantumNous/new-api/commit/ad561128202c0a1f0a547df6c283d1efeb803692) fix(relay): apply GPT-6 chat parameter rules to gpt-6.1-sol and gpt-6.1-terra (#8064)
- 2026-10-27 [01da3a55f8](https://github.com/QuantumNous/new-api/commit/01da3a55f838cfa21010b05c7451c0a42cf5a173) perf(relay): stop recompiling param override regexes on every request (#8061)
- 2026-10-25 [34003e9c2e](https://github.com/QuantumNous/new-api/commit/34003e9c2eed2c20e617f583fdfe54bf1bf42237) test: stabilize channel affinity cache keys (#8009)
- 2026-10-25 [0165fef98d](https://github.com/QuantumNous/new-api/commit/0165fef98d41b02a7368e89dc5fa2df651f8587b) fix(relaykit): preserve Gemini text alongside tool calls (#7998)
- 2026-10-25 [4d962e464d](https://github.com/QuantumNous/new-api/commit/4d962e464de7a16ff8bf16c660307899d49108ba) fix(relaykit): preserve tool argument numbers in Claude responses (#7996)
- 2026-10-25 [f1569d1ea7](https://github.com/QuantumNous/new-api/commit/f1569d1ea7e87cd1a73c5f89d989b6b3bdaf68b3) fix(pricing): classify dynamic per-request billing correctly (#8010)
- 2026-10-25 [bc795cc118](https://github.com/QuantumNous/new-api/commit/bc795cc1185a0ecef0ca51f73d337667026ec62f) fix:修复stop为非字符串和[]string的协议转换问题 (#7935)
- 2026-10-25 [9823ebc7ea](https://github.com/QuantumNous/new-api/commit/9823ebc7ea4ac7102d5d8478d395c7d7eaf6c285) fix(billing): bill cache_creation_input_tokens reported in prompt_tokens_details (#7909)
- 2026-10-24 [e835b1bfc4](https://github.com/QuantumNous/new-api/commit/e835b1bfc43b565d625537830dbfb19328925134) chore(deps): bump axios from 1.18.1 to 1.20.0 in /web (#7835)
- 2026-10-24 [5c8032d700](https://github.com/QuantumNous/new-api/commit/5c8032d7002f82fb04fabf9a9c9b143de78547fe) chore(deps): bump react-i18next from 17.0.10 to 17.1.0 in /web (#7981)
- 2026-10-24 [0b667ca478](https://github.com/QuantumNous/new-api/commit/0b667ca478d6dbce6f60a2e3c4a217a7aab64031) chore(deps): bump i18next from 26.3.6 to 26.5.3 in /web (#7979)
- 2026-10-24 [4105bd089c](https://github.com/QuantumNous/new-api/commit/4105bd089cce1a7c1067e77babf45ebd8d4455ec) chore(deps): bump input-otp from 1.4.2 to 1.5.0 in /web (#7980)
- 2026-10-24 [b6b85cae59](https://github.com/QuantumNous/new-api/commit/b6b85cae5931cd27e9adf21cfccfede800980a83) chore(deps): bump react-resizable-panels from 4.12.2 to 4.14.6 in /web (#7982)
- 2026-10-24 [1dafb19a42](https://github.com/QuantumNous/new-api/commit/1dafb19a429aa7c17b3f4efa457078984b6a4fd3) fix(gemini): preserve top-level imageConfig in chat conversion
- 2026-10-24 [0034b24914](https://github.com/QuantumNous/new-api/commit/0034b24914981078e6ef1a3794e43afe28df43af) fix(task): wait up to 72 hours before timing out tasks
- 2026-10-24 [a404460413](https://github.com/QuantumNous/new-api/commit/a404460413fb7c0c2182e935bb373ba51904df70) chore(web): upgrade Bun to 1.4.4 and use bun check
- 2026-10-23 [33c61a2423](https://github.com/QuantumNous/new-api/commit/33c61a24234aa20cb10e8a644304322c476361de) fix: prevent stale values when switching edit drawer records (#7989)
- 2026-10-23 [30748c6558](https://github.com/QuantumNous/new-api/commit/30748c6558bc97de49ff74871cfec717e50f3322) fix(logging): 隐藏访问日志中的查询凭据 (#7972)
- 2026-10-22 [7fb6e57a5f](https://github.com/QuantumNous/new-api/commit/7fb6e57a5ff671582131c74e00bdfaa867d4e985) fix(channels): preserve advanced custom body passthrough settings (#7971)
- 2026-10-22 [e9d4d6353c](https://github.com/QuantumNous/new-api/commit/e9d4d6353ce2c11df753d751ba1ad778e2433a88) feat(billing): bill OpenAI 12-hour prompt cache writes at their own price (#7965)
- 2026-10-22 [d0b840e75f](https://github.com/QuantumNous/new-api/commit/d0b840e75ff36d731b04ff981ee361819acd593f) fix(relay): keep the client's JSON Schema property order
- 2026-10-22 [d8e65613c0](https://github.com/QuantumNous/new-api/commit/d8e65613c09adf3663f960ea4316414a99694751) feat(web): copy model names from usage log details
- 2026-10-22 [8802113263](https://github.com/QuantumNous/new-api/commit/8802113263b8a94bf8957a1231b4ec04705d7dd9) feat(task): let ordinary channels compete on shared plugin endpoints
- 2026-10-22 [eb169e74d7](https://github.com/QuantumNous/new-api/commit/eb169e74d772afdb281970d0cbe0b62fbf9fa994) feat(pricing): adjust synced upstream prices by the site exchange rate
- 2026-10-22 [e2321a5a9f](https://github.com/QuantumNous/new-api/commit/e2321a5a9f590e8398868cb7ef56af3d833faa07) chore(web): sort locale keys
- 2026-10-22 [db80ac8e0c](https://github.com/QuantumNous/new-api/commit/db80ac8e0c4063e67938cf651c240c69f4f4cf54) fix(task): let task plugins validate their own requests
- 2026-10-22 [dbe97beb9b](https://github.com/QuantumNous/new-api/commit/dbe97beb9b06e8a4e23b5b8f009d882cc49f70bb) feat(task): pass client request headers to task plugin drivers
- 2026-10-22 [556517fa57](https://github.com/QuantumNous/new-api/commit/556517fa57d255c73428325f6a094fd8b4cab76c) feat(task): accept multipart bodies on native plugin routes
- 2026-10-22 [94cc3ff090](https://github.com/QuantumNous/new-api/commit/94cc3ff0906db37703d5ce2bc6a479209b7b9b23) fix(billing): bill Claude cache writes that arrive without a 5m/1h split
- 2026-10-22 [7e6596728f](https://github.com/QuantumNous/new-api/commit/7e6596728ff640392bd77169ebb6137a8fe0c309) fix(relay): stop cutting off slow clients during large stream events
- 2026-10-22 [0c2af9780e](https://github.com/QuantumNous/new-api/commit/0c2af9780e45f6f9073528fb0fcbdcd64b5d9495) fix(relay): finish delivered Responses streams without false disconnect errors (#7956)
- 2026-10-21 [7d8b91268d](https://github.com/QuantumNous/new-api/commit/7d8b91268dab69d7150f345c448b5e250163146a) fix: upgrade Go to 1.26.10 for CVE-2026-111943 (#7942)
- 2026-10-21 [4d36f2c524](https://github.com/QuantumNous/new-api/commit/4d36f2c5245dd6676075be1c4f668dfe56d3340a) chore(deps-dev): bump vitest from 4.1.10 to 5.0.4 in /web (#7838)
- 2026-10-21 [eea05e010c](https://github.com/QuantumNous/new-api/commit/eea05e010c7f212e1a617638fe27164f24d9478d) chore(deps): bump github.com/klauspost/compress from 1.18.3 to 1.18.7 (#7931)
- 2026-10-20 [206e0fa5c6](https://github.com/QuantumNous/new-api/commit/206e0fa5c600127a314a97f88592c067f784f4a6) fix(kling): bump plugin to 1.1.1
- 2026-10-20 [5900426821](https://github.com/QuantumNous/new-api/commit/5900426821433e9bdb0ea1acc8db07125a1c0783) fix(task): restrict query redirect credentials to allowed hosts
- 2026-10-20 [120dbeafb1](https://github.com/QuantumNous/new-api/commit/120dbeafb14528c40873c44253b82b589649fc69) test(security): match the email code only as a whole number
- 2026-10-20 [f9a0634f84](https://github.com/QuantumNous/new-api/commit/f9a0634f843a5367ade7ec4078ade8627dd0098e) test(audit): read PostgreSQL index columns in key order
- 2026-10-20 [55e2ae54d1](https://github.com/QuantumNous/new-api/commit/55e2ae54d1445d92482ac9bb27b2409bb8f46b70) fix(task): reprice each retry and reserve it before submission
- 2026-10-20 [fc8f7d6fef](https://github.com/QuantumNous/new-api/commit/fc8f7d6fef519938d31e17f81035001890272330) fix: fall back to CNY when DeepSeek USD wallet is zero (#7917)
- 2026-10-20 [e2a0c4b16a](https://github.com/QuantumNous/new-api/commit/e2a0c4b16aa6c2e365f0845cf073c52b933385a2) fix(limiter): recover redis script after flush (#6486)
- 2026-10-20 [a8535fac10](https://github.com/QuantumNous/new-api/commit/a8535fac10139c772eed6182f7c90c123ece7e4b) fix(kling): accept single API keys (#7897)
- 2026-10-20 [783735607d](https://github.com/QuantumNous/new-api/commit/783735607d863bc182a00b939e498cc22e92f2f2) fix(relaykit): keep tool calls when merging adjacent assistant messages for Claude (#7921)
- 2026-10-20 [afe73b08ff](https://github.com/QuantumNous/new-api/commit/afe73b08ff6432aca4749a024391c7f886a5bd2d) feat: 支持阿里多模态向量化模型qwen3-vl-embedding (#5179)
- 2026-10-20 [9a42033a61](https://github.com/QuantumNous/new-api/commit/9a42033a616f7196a5d2d10aad4f56191df021b5) fix(task): 修复任务提交与轮询的请求头覆盖 (#7918)
- 2026-10-20 [603a5ceb55](https://github.com/QuantumNous/new-api/commit/603a5ceb55908cc24b509aca2caa6fc956c9958c) feat(dashboard): show cache usage alongside total tokens (#7773)
- 2026-10-19 [32e587406c](https://github.com/QuantumNous/new-api/commit/32e587406c6ebe891f18ab9aeb2f4a1f6b7b9e14) feat(email): 添加 resend 邮件服务商接口 (#7880)
- 2026-10-19 [25372e30b1](https://github.com/QuantumNous/new-api/commit/25372e30b1f94255b1f818aeaaf006f042572580) fix(relay): Claude 流式请求在准备和重试期间持续保活 (#7857)
- 2026-10-18 [058c8231b6](https://github.com/QuantumNous/new-api/commit/058c8231b68c51ee02abee15befcd720fee9f4a2) fix(audit): add composite index for audit log ordering (#7874)
- 2026-10-18 [55bb6b903c](https://github.com/QuantumNous/new-api/commit/55bb6b903c63fcddbf71740d7267b653fe1dbd47) fix(notify): keep notification limit window from resetting every hour (#7877)
- 2026-10-17 [6cf0d90b5f](https://github.com/QuantumNous/new-api/commit/6cf0d90b5ff27f901697dddd61e0c28aba8efefc) feat(deepseek): support V4.1 Flash, the Search and Fetch APIs, and @ reasoning modifiers (#7867)
- 2026-10-17 [6461f084ff](https://github.com/QuantumNous/new-api/commit/6461f084ffcd1c16e3b31a26678584c888ac7f56) feat(notify): turn each kind of notification on or off (#7868)
- 2026-10-17 [5c31c1fa9e](https://github.com/QuantumNous/new-api/commit/5c31c1fa9ebff24c7ed88005fbab83154b4958f8) feat(channels): improve priority dragging and add onboarding guides
- 2026-10-17 [5e9595190e](https://github.com/QuantumNous/new-api/commit/5e9595190e3006ecc57712f6a365573b5a20de4d) feat(channels): add drag mode for channel priority ordering
- 2026-10-17 [5a59d69153](https://github.com/QuantumNous/new-api/commit/5a59d691535ff84cda9d46a7d84b1be43b18227f) fix(pricing): hide task prices left at zero when another price is set
- 2026-10-17 [3e2ce2c885](https://github.com/QuantumNous/new-api/commit/3e2ce2c8856e32ca7054d673f8cd059ae466e80c) fix: align model card summary with status bar hours (#7847)
- 2026-10-17 [b0a98ac49c](https://github.com/QuantumNous/new-api/commit/b0a98ac49ca138e6274dbc16dc99365a2f639929) fix(web): apply theme class before first paint so Dark Reader detects dark mode (#7571)
- 2026-10-17 [f92c576506](https://github.com/QuantumNous/new-api/commit/f92c5765068552042690917422fe9087b1c27302) fix(web): cover Cloudflare clef and NVIDIA model families in provider rules (#7840)
- 2026-10-17 [ff3684658f](https://github.com/QuantumNous/new-api/commit/ff3684658fc1ab5747bda9a142910a617df8eb10) fix(relayconvert): 保留 Chat/Claude → Responses 转换中 input_image 的 detail 字段 (#7855)
- 2026-10-16 [09109ec315](https://github.com/QuantumNous/new-api/commit/09109ec315a5086366f94bcce00f177ec831dbe1) fix(middleware): 过载哨兵改用容器口径并加入连续性与新鲜度判定 (#7317)
- 2026-10-16 [d032204733](https://github.com/QuantumNous/new-api/commit/d032204733db31208f10bac118b6e683aab3bf29) feat(aws): add Mantle Messages API support
- 2026-10-16 [ca2cae676d](https://github.com/QuantumNous/new-api/commit/ca2cae676db87333a583b6e104d0f1f7366f3258) fix(relaykit): emit summary on reasoning items announced without one (#7833)
- 2026-10-15 [45fbb149f0](https://github.com/QuantumNous/new-api/commit/45fbb149f0c89ff48d805335f22419be392d0350) fix(rate-limit): count each group with its own limit separately
- 2026-10-15 [6549a17eda](https://github.com/QuantumNous/new-api/commit/6549a17eda83eaa270fd8cbb0c03623836b351d6) fix(dashboard): draw flow chart labels in the theme text color (#7821)
- 2026-10-15 [c9179d38b7](https://github.com/QuantumNous/new-api/commit/c9179d38b780c1d6635bcbc50f77b5074f8b49bb) fix(playground): support the reasoning field in chat responses (#7831)
- 2026-10-15 [908f94d65d](https://github.com/QuantumNous/new-api/commit/908f94d65d4f60729db3d8870f505ca3fba943c5) fix(web): hide pages the user cannot open from console search
- 2026-10-15 [19d435e7fd](https://github.com/QuantumNous/new-api/commit/19d435e7fdfdaa2b9a734a76393f112cf980b815) fix(relay): mask only the custom upstream address in error messages (#7822)
- 2026-10-14 [718a4e7f97](https://github.com/QuantumNous/new-api/commit/718a4e7f9751c5bf1958747cd0c2c895e2b937d4) fix(vertex): use the multi-region hostnames for us and eu
- 2026-10-14 [f196934ce6](https://github.com/QuantumNous/new-api/commit/f196934ce6908623cb0ee302cd4ec9a6e11f0228) feat(plugins): bind Hailuo, Vidu and Veo plugins to New API channels
- 2026-10-14 [bfdad293cd](https://github.com/QuantumNous/new-api/commit/bfdad293cdd4fd93c4c80b46ce1baa7a3a593e92) fix(vidu): bill the model, duration and resolution Vidu generates
- 2026-10-14 [150ab3cd43](https://github.com/QuantumNous/new-api/commit/150ab3cd439441e246be7bca9a1ab25e339e1865) chore(licenses): refresh the third-party dependency inventory
- 2026-10-14 [76cd5114ea](https://github.com/QuantumNous/new-api/commit/76cd5114eae2fd933ea0da08f41984f1ff52cd85) build: upgrade moejs to v0.1.0-alpha.7
- 2026-10-14 [1b63c28dcc](https://github.com/QuantumNous/new-api/commit/1b63c28dcc999c4445c70f21ec90103ab49e8be5) fix(claude): preserve explicit zero usage and cache-only billing (#7816)
- 2026-10-14 [c9f7b62081](https://github.com/QuantumNous/new-api/commit/c9f7b6208131089bcf3c36ac554432967b57156d) fix(web): sort pricing models by input, output and per-request price (#7813)
- 2026-10-14 [a0bf6a8a41](https://github.com/QuantumNous/new-api/commit/a0bf6a8a412764bc296a8e27cf8c4aaa3f3f0ed9) fix(vertex): pass project_id in API Key mode to avoid 404 on official endpoints (#6330)
- 2026-10-14 [4013729f6a](https://github.com/QuantumNous/new-api/commit/4013729f6a2a14b9b5988d3c1a529ea97d2b2d8d) fix: disable sensitive word filtering by default and clear the default word list (#7814)
- 2026-10-13 [7de10f2c60](https://github.com/QuantumNous/new-api/commit/7de10f2c602b50122017556e8a09cb7c488c273c) feat(token): let an API key read its owner's quota
- 2026-10-13 [ef60ef27f3](https://github.com/QuantumNous/new-api/commit/ef60ef27f36e418143fa4abf0c9025ba6f17fba7) fix(relay): keep the cause of a failed body read for every later reader
- 2026-10-13 [d10eafb57e](https://github.com/QuantumNous/new-api/commit/d10eafb57e289e971453eb68e882addaa29f8cc9) fix(relay): report the real cause when a request body cannot be read
- 2026-10-13 [63cecab824](https://github.com/QuantumNous/new-api/commit/63cecab824bad1a421d19950004c309bb2054c09) feat(channel): support Sub2API balance queries (#7749)
- 2026-10-13 [b4be2a7847](https://github.com/QuantumNous/new-api/commit/b4be2a7847c4db4a956c3e71ae4bc1f31bc71b60) feat(ui): show newly added pricing groups at the top (#7782)
- 2026-10-12 [633b4c5b19](https://github.com/QuantumNous/new-api/commit/633b4c5b190dc1af99a2814391354d86948e7943) feat(logs): export usage logs from the common log page (#7774)
- 2026-10-12 [dd59854050](https://github.com/QuantumNous/new-api/commit/dd59854050f26f4f67bb3dac5274af5e17013462) chore(deps-dev): bump undici from 6.28.0 to 6.29.0 in /electron (#7624)
- 2026-10-12 [93cf4ba6fb](https://github.com/QuantumNous/new-api/commit/93cf4ba6fb92e3f755fdeb6af9a9484bfa748816) chore(deps-dev): bump js-yaml from 4.3.1 to 4.3.2 in /electron (#7327)
- 2026-10-12 [3c9fb2a0de](https://github.com/QuantumNous/new-api/commit/3c9fb2a0de168d88ce5bffbf44acecdc1cdcdd39) chore(deps-dev): bump brace-expansion from 1.1.16 to 1.1.21 in /electron (#7623)
- 2026-10-12 [9f048107ed](https://github.com/QuantumNous/new-api/commit/9f048107edbba2156691718126de7c4577614533) chore(deps): bump ai from 7.0.31 to 7.0.136 in /web (#7767)
- 2026-10-12 [3fe394cdba](https://github.com/QuantumNous/new-api/commit/3fe394cdba280c00bc566de493b5fc8399b0a7d1) chore(deps): bump github.com/gorilla/websocket from 1.5.0 to 1.5.3 (#7001)
- 2026-10-12 [1c58f3d161](https://github.com/QuantumNous/new-api/commit/1c58f3d161fa32755b372d9449f2d5e077d50a5d) chore(deps-dev): bump fast-uri from 3.1.5 to 3.1.7 in /electron (#7169)
- 2026-10-12 [7f51a97226](https://github.com/QuantumNous/new-api/commit/7f51a97226c43bc082a065bdc8b40b719c4bdcce) chore(deps-dev): bump js-yaml from 4.3.1 to 4.3.2 in /electron (#7327)
- 2026-10-12 [7e25797d14](https://github.com/QuantumNous/new-api/commit/7e25797d14681e4e16c2b46831450e7d6a5ccc74) chore(deps-dev): bump electron from 39.8.10 to 41.10.6 in /electron (#7625)
- 2026-10-12 [bf4e1ebfb8](https://github.com/QuantumNous/new-api/commit/bf4e1ebfb88579930d75d852d2d2bca74cbd5f62) build(deps-dev): bump oxfmt from 0.57.0 to 0.72.0 in /web (#7766)
- 2026-10-12 [82306b3c35](https://github.com/QuantumNous/new-api/commit/82306b3c35945bc0ed2c567688afe28d8a281300) build(deps-dev): bump @tanstack/router-plugin in /web (#7768)
- 2026-10-12 [c730add954](https://github.com/QuantumNous/new-api/commit/c730add954e1b565656a9320444cac5fee12cfe4) build(deps): bump stream-markdown-parser from 1.1.3 to 1.2.18 in /web (#7769)
- 2026-10-12 [b8c2dc1266](https://github.com/QuantumNous/new-api/commit/b8c2dc12660dceb2a905008a672f8bc0260c98c8) build(deps): bump yace from 1.1.0 to 1.1.1 in /web (#7770)
- 2026-10-12 [ba29f06453](https://github.com/QuantumNous/new-api/commit/ba29f06453942d9b5c0b32330340b8a000692c4f) chore(deps-dev): bump @xmldom/xmldom from 0.8.13 to 0.8.15 in /electron (#7187)
- 2026-10-12 [767a82b876](https://github.com/QuantumNous/new-api/commit/767a82b8764778b847462ebecdc231eacd1e282c) fix(claude): forward thinking.block_binding and every anthropic-beta value (#7732)
- 2026-10-12 [2b85a8fc67](https://github.com/QuantumNous/new-api/commit/2b85a8fc67bdb6d8d41a25f60f734dd3b9917618) fix(auth): show legal consent warning on password login (#7771)
- 2026-10-12 [2ed058795b](https://github.com/QuantumNous/new-api/commit/2ed058795bf0ce968134aa8db18b0dd86b54c30e) fix(ci): stabilize checks and configure Bun dependency updates (#7765)
- 2026-10-12 [9948b1ef14](https://github.com/QuantumNous/new-api/commit/9948b1ef14ea17fb23539386c0cb09018671c955) fix(pricing): accept declared endpoint arrays in pricing conversion (#7724)
- 2026-10-12 [4bfcf28121](https://github.com/QuantumNous/new-api/commit/4bfcf2812198913156a3e88a43e746b61a7b6ef5) chore(web): bump @lobehub/icons to 5.22.0 (#7764)
- 2026-10-12 [fc08b76f0f](https://github.com/QuantumNous/new-api/commit/fc08b76f0fb3f95e7cb81667e7bf350200147fb3) fix(web): preserve recorded billing tiers in usage log previews (#7763)
- 2026-10-12 [ba0e478e06](https://github.com/QuantumNous/new-api/commit/ba0e478e061ee0c824922d89c05620de878103cf) fix(relaykit): preserve assistant text with tool calls (#6871)
- 2026-10-12 [34b1942123](https://github.com/QuantumNous/new-api/commit/34b19421237fc4695e919cdd799c1b7813023085) fix(pricing): allow root access when the model square is disabled (#7761)
- 2026-10-12 [b22a43b6e8](https://github.com/QuantumNous/new-api/commit/b22a43b6e89e5349d8cdbddc88d570b16e15fb1f) fix: mark task settlement logs with is_task for tiered billing display (#7446)
- 2026-10-12 [bbf60b49f6](https://github.com/QuantumNous/new-api/commit/bbf60b49f6a4cab709e49e57d45f7911fc7dccae) fix(logs): vertically align billing source icons on mobile (#7760)
- 2026-10-12 [3f926c3ef4](https://github.com/QuantumNous/new-api/commit/3f926c3ef49745be273cd8f753fb691abdd7b836) fix(metrics): count completed streams after context cancellation (#7638)
- 2026-10-12 [3827fd8307](https://github.com/QuantumNous/new-api/commit/3827fd83076b9cbaa3f1d9f9d2957ae653573ed7) fix(web): show cache read price on expression model cards (#7757)
- 2026-10-11 [42e694eb60](https://github.com/QuantumNous/new-api/commit/42e694eb60fe3191795c00d416571ced716eb9c4) fix(web): map Meta Muse Spark models to Meta icon (#7734)
- 2026-10-11 [c10780e054](https://github.com/QuantumNous/new-api/commit/c10780e054e6e518c9d0d8a903526663c9b57d75) fix(relayconvert): keep JSON Schema constraints in tools and structured output sent to Gemini
- 2026-10-11 [801456a061](https://github.com/QuantumNous/new-api/commit/801456a061fb9988957341c4eb5aa8ec20c40de1) fix(relay): send a close frame when a Responses WebSocket session ends
- 2026-10-11 [4b13d9388e](https://github.com/QuantumNous/new-api/commit/4b13d9388ed6f43118707534daad4903e5d22745) fix(relayconvert): drop non-string enum values when converting OpenAI tool schemas (#7729)
- 2026-10-11 [93ea55f3c3](https://github.com/QuantumNous/new-api/commit/93ea55f3c3102f7bc11cd005570c7965f0f22a03) feat(web): pick dates and times down to the second
- 2026-10-11 [96275d224b](https://github.com/QuantumNous/new-api/commit/96275d224bfcf6b5545ca7e57b3bf90f7c3ad396) fix(web): show date pickers and relative times in the interface language
- 2026-10-11 [06dd76a9d3](https://github.com/QuantumNous/new-api/commit/06dd76a9d38eb0eaf8e900b1264b500250156c4d) fix(playground): keep each account's Playground data separate
- 2026-10-11 [1fb6d07fee](https://github.com/QuantumNous/new-api/commit/1fb6d07fee0171d5f4a0bd2d2ea2660572b08389) perf(web): stop loading the whole Simple Icons pack on four pages
- 2026-10-11 [b892a6b2e4](https://github.com/QuantumNous/new-api/commit/b892a6b2e44df572698f6cf731ff381f1977a1f2) fix(web): reload once when a page's code fails to load
- 2026-10-11 [977450a3e8](https://github.com/QuantumNous/new-api/commit/977450a3e810ee9c7d7e98acce1dc4efad90b491) fix(rate-limit): explain model rate-limit rejections without Redis
- 2026-10-11 [5ac7562840](https://github.com/QuantumNous/new-api/commit/5ac75628409de7e3adde7d36771700714af50853) fix(relayconvert): keep answer text that shares a stream chunk with reasoning
- 2026-10-11 [13c8e90b8f](https://github.com/QuantumNous/new-api/commit/13c8e90b8f4b929aa88c2e9616efc462e0068d72) fix(relayconvert): send Responses image URLs to Chat upstreams as objects
- 2026-10-11 [036ba376da](https://github.com/QuantumNous/new-api/commit/036ba376da7d572a4db3e0be8553aadc28c3616a) fix(relaykit): match Claude effort and Gemini 2.5 budgets to each model
- 2026-10-10 [0f38a14a5e](https://github.com/QuantumNous/new-api/commit/0f38a14a5e8e322ea741478b0e49878be3130e1d) fix(web): restore browser sans-serif for Auto font
- 2026-10-10 [1ca79d57e7](https://github.com/QuantumNous/new-api/commit/1ca79d57e77d91c855371b43bb26b151429e18a1) fix(pricing): show recent status samples without gaps
- 2026-10-10 [c6741c36a9](https://github.com/QuantumNous/new-api/commit/c6741c36a9a1a729554ad94ee99f21602a214a23) feat(channel): add a per-channel switch to skip upstream TLS verification
- 2026-10-10 [c8d234baa1](https://github.com/QuantumNous/new-api/commit/c8d234baa18b9090b394fc1b8f384eec94f8bfef) fix(xai): bill video edits and extensions by their output resolution
- 2026-10-10 [ba3b2c015a](https://github.com/QuantumNous/new-api/commit/ba3b2c015ad83e6710c18592b0e3e3ba9df9a67e) refactor(jsplugin): stop requiring json-clone@1 and submit-sse-delta@1
- 2026-10-10 [3f02244783](https://github.com/QuantumNous/new-api/commit/3f02244783e0bb96b1398601f156625936be2f15) feat(web): show the request domains a plugin declares when uploading it
- 2026-10-10 [66c3e14282](https://github.com/QuantumNous/new-api/commit/66c3e142820e513a328def88f2771f73479275bc) fix(hailuo): download MiniMax v1 videos through the file retrieve link
- 2026-10-10 [29509776ef](https://github.com/QuantumNous/new-api/commit/29509776ef855965fb2cff3978578d891005945e) feat(jsplugin): let driver hooks read upstream data with utils.fetch
- 2026-10-10 [b66674ac23](https://github.com/QuantumNous/new-api/commit/b66674ac23d79bfd35a90797ca1ac64fc74a2d8f) feat(task-plugin): let Seedance and wan3.0 pick the video duration with -1
