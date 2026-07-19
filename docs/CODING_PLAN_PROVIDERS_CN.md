# Coding Plan 账号接入与验证

本文说明如何在管理网站添加 GLM Coding Plan、Kimi Coding Plan 和 GitHub Copilot 账号，并通过 sub2api 的下游 API 使用它们。下列端点与模型截至 **2026-07-19** 与当前实现、官方文档一致；上游后续可能调整。

## 1. 分组与协议

三个 Coding Plan 账号都归入 **OpenAI 协议**，网站会将其保存为 `platform=openai`、`type=apikey`。建议为每个供应商单独创建一个 OpenAI 分组，并将下游 API Key 绑定到对应分组，避免不同套餐的模型和额度混用。

Coding Plan 上游均按 Chat Completions 调用，下游支持：

| 下游端点 | 处理方式 |
|---|---|
| `/v1/chat/completions` | 转发到供应商的 Chat Completions 接口。 |
| `/v1/responses` | 将 Responses 请求转换为 Chat Completions，并把响应转换回 Responses 格式。 |
| `/v1/messages` | 将 Anthropic Messages 请求转换为 Chat Completions，并把响应转换回 Messages 格式。 |
| `/v1/messages/count_tokens` | 不向上游发送凭据，使用本地 tokenizer 返回估算值。 |
| `/v1/models?client_version=...` | 从账号模型映射生成稳定的本地 Codex 模型清单。 |

使用 `/v1/messages` 前，必须在对应 OpenAI 分组中开启 **允许 `/v1/messages` 调度**。如果客户端发送 Claude 模型名，还应在分组的“OpenAI Messages 调度配置”中，将 Opus、Sonnet、Haiku 系列映射到该账号实际支持的模型。

Coding Plan 不实现完整 OpenAI 平台协议，因此不会被调度到独立 `/v1/alpha/search`、`/v1/responses/compact` 或 Responses WebSocket 请求；HTTP Chat Completions、Responses 和 Messages 桥接不受影响。

## 2. 当前端点与模型

| 供应商 | 上游 Base URL | 当前预置模型 |
|---|---|---|
| GLM Coding Plan | `https://open.bigmodel.cn/api/coding/paas/v4` | `GLM-5.2`、`GLM-5-Turbo`、`GLM-4.7` |
| Kimi Coding Plan | `https://api.kimi.com/coding/v1` | `k3`、`kimi-for-coding`、`kimi-for-coding-highspeed` |
| GitHub Copilot | 由 GitHub 令牌交换响应下发；当前默认回退为 `https://api.githubcopilot.com` | `claude-sonnet-4.6`、`claude-haiku-4.5`、`gpt-5.4`、`gpt-5.3-codex`、`gemini-3.1-pro-preview`、`gemini-3.5-flash`、`mai-code-1-flash` |

GLM 同时接受表中的小写别名，例如 `glm-5.2`、`glm-5-turbo`、`glm-4.7`，转发时会映射到官方模型名。模型是否可用仍取决于账号套餐、组织策略和上游实时授权。

官方参考：

- [GLM Coding Plan 接入工具](https://docs.bigmodel.cn/cn/coding-plan/tool/others)
- [GLM Coding Plan 常见问题](https://docs.bigmodel.cn/cn/coding-plan/faq)
- [Kimi Code 概览](https://www.kimi.com/code/docs/)
- [GitHub Copilot CLI 模型与命令参考](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference)

## 3. 准备凭据

### GLM Coding Plan

1. 在智谱 GLM Coding Plan 页面确认套餐有效。
2. 创建该 Coding Plan 专用 API Key。不要使用普通按量 API 的 Key 代替。
3. 网站会固定使用 Coding Plan Base URL，无需手工修改。

### Kimi Coding Plan

1. 在 Kimi Code Console 确认会员权益有效并创建 API Key。
2. 使用 `api.kimi.com` 账号体系的 Key；`api.moonshot.cn` 开放平台 Key 与其不互通。
3. `k3` 和高速模型可能要求更高套餐，普通会员可先使用 `kimi-for-coding`。

### GitHub Copilot

推荐使用 GitHub **fine-grained personal access token**：

1. Token 的 Resource owner 选择拥有 Copilot 权益的**个人账号**，不要选择组织。
2. 在 **Account permissions** 中添加 **Copilot Requests: Read and write**，对应权限参数为 `copilot_requests=write`。
3. Repository access 按实际需要选择最小范围，不要附加无关的仓库写入或管理权限。
4. 生成 `github_pat_...` Token。GitHub Copilot CLI 官方不支持 classic `ghp_...` PAT。

GitHub 官方说明见 [Copilot CLI 认证](https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/install-copilot-cli#authenticating-with-a-personal-access-token) 和 [fine-grained PAT 权限表](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#permissions)。Token 应按密码管理，泄露后立即吊销。

## 4. 在网站添加账号

1. 进入 **管理后台 -> 分组**，创建或选择平台为 **OpenAI** 的分组。需要兼容 Claude SDK/Claude Code 时，开启“允许 `/v1/messages` 调度”并配置模型映射。
2. 进入 **管理后台 -> 账号 -> 添加账号**。
3. 填写账号名称，然后在 **Coding Plan** 区域选择 GLM、Kimi 或 GitHub Copilot。
4. GLM/Kimi 填入对应 Coding Plan API Key；Copilot 填入上一步创建的 GitHub fine-grained PAT。
5. 选择代理、并发数和刚创建的 OpenAI 分组，然后提交。官方 Base URL、Chat Completions 能力和严格模型映射会自动写入。
6. 在账号列表点击 **测试账号**，选择该供应商支持的模型完成连接测试。若测试失败，先核对套餐状态、Token 权限、代理和模型授权。

随后创建一个下游 API Key，并将它绑定到同一个 OpenAI 分组。下游调用使用的是 sub2api API Key，不是上游 Coding Plan Key 或 GitHub Token。

## 5. 验证下游桥接

先设置部署地址、下游 API Key 和一个已授权模型：

```bash
export SUB2API_BASE_URL="https://sub2api.example.com"
export SUB2API_API_KEY="sk-your-sub2api-key"
export MODEL="glm-5.2"
```

验证 Chat Completions：

```bash
curl -sS "$SUB2API_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $SUB2API_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK\"}],\"stream\":false}"
```

验证 Responses 桥接：

```bash
curl -sS "$SUB2API_BASE_URL/v1/responses" \
  -H "Authorization: Bearer $SUB2API_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$MODEL\",\"input\":\"Reply with OK\",\"stream\":false}"
```

验证 Messages 桥接：

```bash
curl -sS "$SUB2API_BASE_URL/v1/messages" \
  -H "x-api-key: $SUB2API_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$MODEL\",\"max_tokens\":64,\"messages\":[{\"role\":\"user\",\"content\":\"Reply with OK\"}],\"stream\":false}"
```

切换供应商时，将 `MODEL` 改为该账号预置且实际获授权的模型，并使用绑定到相应分组的下游 API Key。

## 6. Copilot 协议与使用边界

Copilot 账号中保存的是 GitHub 客户端 Token。服务端会用它向 GitHub 交换短期 Copilot Token，并采用交换响应下发的 API 地址；短期 Token 临近过期或遇到认证失效时会重新交换。该交换端点、请求头和模型列表可能随 GitHub 客户端协议变化，不属于稳定承诺的公共 OpenAI API，升级后应重新执行账号测试和三个下游验证。

使用这些账号时必须遵守 GLM Coding Plan、Kimi Code 和 GitHub Copilot 的套餐条款、授权工具范围、额度、计费及组织策略。本项目不承诺套餐一定允许通过中转站使用，也不承诺或提供规避客户端身份、User-Agent、授权工具名单、额度或风控规则的能力。上游拒绝某种客户端身份或使用方式时，应停止调用并按供应商规则处理。
