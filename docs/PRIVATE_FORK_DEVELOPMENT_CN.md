# 私有下游仓库开发、上游同步与页面一键更新

本文档适用于当前私有仓库 `X-manist/EntangledAPI`、生产分支 `custom/main`，以及 GreenVPS 上的 `sub2api`。

## 1. 整体原则

GitHub 的普通 Fork 会继承上游仓库的可见性：公开仓库的 Fork 仍然公开。因此这里所说的“私有 fork”，实际采用的是一个**私有下游镜像仓库**：

- `upstream`：官方仓库 `Wei-Shaw/sub2api`，只拉取，不直接推送。
- `origin`：私有仓库 `X-manist/EntangledAPI`，保存自己的功能和 Release。
- `custom/main`：生产主分支，始终基于最新可用的 `upstream/main`。
- `feature/*`：GLM、Copilot 等独立功能分支。

相关 GitHub 官方说明：

- [复制仓库并推送到新仓库](https://docs.github.com/en/repositories/creating-and-managing-repositories/duplicating-a-repository)
- [配置 upstream remote](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/configuring-a-remote-repository-for-a-fork)

更新链路如下：

```text
官方 upstream/main
        ↓ rebase / merge
私有 custom/main + 自定义功能
        ↓ 合并后自动测试、自动打 tag
私有 GitHub Release（二进制 + checksums.txt）
        ↓ 管理页面“立即更新”
/app/data/runtime/sub2api
        ↓ 页面点击“重启”
Docker 使用新版本，且容器重建后仍保留
```

## 2. 当前分支状态

当前本地分支已经整理为：

```text
custom/main
  └─ fix: route OAuth Codex image bridge through HTTP
      └─ upstream/main
```

rebase 前的旧提交保存在本地备份分支：

```text
backup/green-ws-before-upstream-20260712
```

现有 `origin/main` 与官方仓库没有共同提交祖先，不要直接把 `custom/main` 强推覆盖到 `origin/main`。第一次推送应创建新分支：

```bash
git switch custom/main
git push -u origin custom/main
```

推送成功后，在私有仓库设置中：

1. 将默认分支改为 `custom/main`。
2. 为 `custom/main` 开启分支保护和 Pull Request 合并要求。
3. 在 Actions 设置中允许 Workflow 获得 `Read and write permissions`，用于创建 Release tag 和同步 VERSION 文件。
4. 确认 Actions 可以创建和上传私有 Release 资产。

确认运行稳定后，再决定是否归档旧的 `origin/main`；不要在确认前删除旧分支。

## 3. GreenVPS 更新配置

GreenVPS 的 `.env` 需要加入：

```dotenv
UPDATE_REPOSITORY=X-manist/EntangledAPI
UPDATE_GITHUB_TOKEN=github_pat_xxxxxxxxxxxxxxxxxxxx
UPDATE_DOCKER_IMAGE=ghcr.io/x-manist/sub2api

SUB2API_RUNTIME_SEED_POLICY=if-missing
SUB2API_RUNTIME_DIR=/app/data/runtime
```

`UPDATE_GITHUB_TOKEN` 建议使用 GitHub Fine-grained PAT：

- Repository access：只选择 `X-manist/EntangledAPI`。
- Repository permissions：`Contents: Read-only`。
- 不需要写权限，不要使用个人全仓库管理 Token。

GitHub Release 和 Release asset 的读取需要 `Contents: read`。官方权限说明见 [REST Releases API](https://docs.github.com/en/rest/releases/releases) 和 [REST Release Assets API](https://docs.github.com/en/rest/releases/assets)。

Token 只放在服务器 `.env` 中：

- 不提交到 Git。
- 不放入前端环境变量。
- 不写入 `config.example.yaml` 的真实值。
- 泄露后立即吊销并重新生成。

## 4. Docker 持久化更新机制

镜像里的 `/app/sub2api` 现在只是启动种子。容器第一次启动时，入口脚本会把它复制到：

```text
/app/data/runtime/sub2api
```

管理页面更新时，后端替换的是这个持久化二进制。由于 `/app/data` 是 Docker volume，因此：

- `docker restart` 后更新仍然存在。
- 主机重启后更新仍然存在。
- `docker compose up -d` 重建应用容器后更新仍然存在。
- 删除 `sub2api_data` volume 后，运行版本会回到镜像种子。

`SUB2API_RUNTIME_SEED_POLICY` 支持：

| 值 | 用途 |
|---|---|
| `if-missing` | 推荐。仅首次复制镜像二进制，之后保留页面更新版本。 |
| `always` | 故障恢复。每次启动都用镜像覆盖持久化版本。只能临时使用。 |
| `never` | 禁用持久化自更新，直接运行镜像内二进制。 |

首次启用该机制仍需要部署一次新镜像。完成这次 bootstrap 后，后续正常版本不需要手工重新部署。

## 5. 日常功能开发

以 GLM 订阅为例：

```bash
git switch custom/main
git pull --ff-only origin custom/main
git switch -c feature/glm-subscription

# 开发、测试、提交
git add <files>
git commit -m "feat(glm): add subscription account support"
git push -u origin feature/glm-subscription
```

然后在私有仓库创建 PR，合并到 `custom/main`。

Copilot 使用独立分支：

```bash
git switch custom/main
git switch -c feature/copilot-subscription
```

建议保持 Provider 功能独立：

- 登录、Token 导入和刷新。
- 配额探测与账号健康状态。
- 模型发现与模型映射。
- 请求/响应协议适配。
- 调度、计费和审计。
- 前端账号编辑和状态展示。

不要把 GLM 和 Copilot 混在一个超大提交里，否则以后同步官方上游时冲突会明显增加。

## 6. 同步官方上游

先配置 remotes：

```bash
git remote -v
git remote add upstream https://github.com/Wei-Shaw/sub2api.git  # 已存在则跳过
```

单人维护、允许重写 `custom/main` 历史时：

```bash
git fetch upstream --prune --tags
git fetch origin
git switch custom/main
git pull --ff-only origin custom/main
git rebase upstream/main

# 有冲突时逐个解决
git add <resolved-files>
git rebase --continue

cd backend
go test -tags=unit ./internal/config ./internal/repository ./internal/service
cd ../frontend
pnpm install --frozen-lockfile
pnpm run build
cd ..
sh deploy/docker-entrypoint_test.sh

git push --force-with-lease origin custom/main
```

多人协作或不允许重写主分支时，使用同步分支：

```bash
git fetch upstream --prune --tags
git switch custom/main
git switch -c sync/upstream-YYYYMMDD
git merge --no-ff upstream/main

# 解决冲突并测试后
git push -u origin sync/upstream-YYYYMMDD
```

然后通过 PR 合并到 `custom/main`。不要在多人共同开发的分支上直接 `push --force`。

## 7. 自动发布与页面更新

合并到 `custom/main` 后，工作流分两段执行：

1. `.github/workflows/custom-release-on-merge.yml`
   - 后端单元测试。
   - 前端构建。
   - Docker 持久化入口测试。
   - 自动生成下一版本 tag，例如 `v0.1.151-entangled.2`。
2. `.github/workflows/release.yml`
   - 构建 Release 二进制。
   - 上传 Linux archive 和 `checksums.txt`。
   - 发布为稳定 Release，使 `/releases/latest` 能找到它。
   - 同时构建私有 GHCR 镜像。

版本规则：

```text
v<官方版本>-entangled.<私有修订号>

示例：
v0.1.151-entangled.1
v0.1.151-entangled.2
v0.1.152-entangled.1
```

不要执行：

```bash
git push origin --tags
```

否则可能把大量官方 tag 推进私有仓库。只推送明确的自定义 tag；正常情况下自动工作流会完成打 tag。

Release 完成后，GreenVPS 管理页面会显示新版本：

1. 点击版本徽章。
2. 点击刷新。
3. 确认版本号是 `*-entangled.*`，且 Release 链接属于私有仓库。
4. 点击“立即更新”。
5. 校验下载文件 SHA-256 后替换持久化二进制。
6. 点击“立即重启”。Docker 的 `restart: unless-stopped` 会拉起新版本。

## 8. 故障恢复

### 页面检查更新返回 404

依次检查：

- `UPDATE_REPOSITORY` 是否严格为 `owner/repository`。
- PAT 是否可访问该私有仓库。
- PAT 是否有 `Contents: Read-only`。
- Release 是否已经发布，而不是 Draft。

### 页面显示没有更新

- Release 必须是稳定 Release，不能标记为 prerelease。
- tag 必须符合 `vX.Y.Z-entangled.N`。
- 新版本必须大于当前版本。
- 点击版本面板中的刷新按钮，绕过 20 分钟缓存。

### 更新提示缺少兼容二进制

Release 必须至少包含：

```text
sub2api_<version>_linux_amd64.tar.gz
checksums.txt
```

不要开启会跳过二进制资产的自定义发布配置。本仓库的 simple release 已保留 Linux amd64 archive 和 checksum。

### 新版本无法启动

优先使用页面中的版本回退。必要时执行一次镜像恢复：

1. 临时把 `SUB2API_RUNTIME_SEED_POLICY` 改成 `always`。
2. 重建或重启应用容器，使镜像种子覆盖运行二进制。
3. 确认服务健康后，立即改回 `if-missing`。

不要长期保留 `always`，否则每次页面更新后的重启都会被旧镜像覆盖。

## 9. 发布前检查清单

- [ ] `custom/main` 已同步目标 `upstream/main`。
- [ ] 自定义功能按 Provider 拆分并经过 PR。
- [ ] 后端单元测试通过。
- [ ] 前端构建通过。
- [ ] `deploy/docker-entrypoint_test.sh` 通过。
- [ ] Release 不是 Draft 或 prerelease。
- [ ] Linux amd64 archive 和 `checksums.txt` 已上传。
- [ ] GreenVPS 的 Token 只有私有仓库只读权限。
- [ ] 管理页面展示的是私有仓库 Release，而不是 `Wei-Shaw/sub2api`。

