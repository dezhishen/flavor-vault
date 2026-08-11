---
name: flavor-vault
description: 'Flavor Vault 菜谱管理 CLI（fv）使用指南。Use when: 需要读取/查询菜谱（list/search/show/stats/filter），编辑/新增/删除菜谱（add/edit/rm），构建静态站点（build），或操作 GitHub 数据分支（gh）。提供命令、参数、配置示例与数据契约，供 AI 助手（如 OpenClaw）调用 fv 命令完成任务。'
---

# Flavor Vault · `fv` CLI 能力指南

Flavor Vault 是一个基于 **GitHub + 纯静态托管** 的菜谱管理系统。CLI 二进制为 `fv`（Go，免 cgo）。

- 数据仓库：菜谱以 JSON 单文件存放于 GitHub 数据分支（默认 `recipes`）
- 静态站点：由 GitHub Actions 构建并部署到 gh-pages（`dist/`）
- 前端：Vue3 + Element Plus，纯静态，读 `./data/*.json`

## 使用模式（两种）

| 模式 | 用途 | 需要什么 |
|---|---|---|
| **只读** | 查询菜谱 | `--endpoint <url>`（或 `FV_ENDPOINT`）；未设则读本地 `dist/data` |
| **读写** | 新增/编辑/删除菜谱 | `--repo <owner/repo>`（或 `FV_REPO`）+ `--branch <branch>`（默认 `recipes`，或 `FV_BRANCH`）+ `GITHUB_TOKEN` |

> 不需要任何配置文件。可选 `config.example.yaml`（项目根）供复制参考；也可放 `.flavor-vault/config.yaml`（endpoint / github 块）。

## 构建 / 安装

```bash
export PATH="$HOME/.local/opt/go/bin:$PATH"   # 若 go 在自定义目录
go build -o fv ./cmd/fv
./fv --help
```

## 命令一览

### 读取（只读，需 endpoint）

| 命令 | 说明 | 示例 |
|---|---|---|
| `fv list [--tag 标签] [--json]` | 列出菜谱 | `fv list --endpoint https://owner.github.io/repo/data` |
| `fv show <id> [--raw]` | 打印单菜谱详情 | `fv show hong-shao-rou --endpoint <url>` |
| `fv search <关键词...> [--json]` | 全文搜索（菜名/食材/步骤等，多词 AND） | `fv search 鸡翅 烤箱 --endpoint <url>` |
| `fv stats [--json]` | 统计（总数/标签/难度等） | `fv stats --endpoint <url>` |
| `fv filter --厨具 炒锅 --标签 凉菜 [--json]` | 按倒排索引求交集 | `fv filter --食材 鸡翅 --endpoint <url>` |
| `fv ask <问题> [--top N]` | AI 语料检索（ai-corpus.json） | `fv ask 有什么快手凉菜 --endpoint <url>` |

### 编辑（读写，需 GITHUB_TOKEN + repo/branch）

| 命令 | 说明 | 示例 |
|---|---|---|
| `fv add [--json '...' | @file] [--action-id X]` | 新增菜谱（交互式或 JSON），经 GitHub API 提交单文件 | `fv add --repo owner/recipes --branch recipes --json @r.json` |
| `fv edit <id> [--json '{"stats":{"difficulty":4}}']` | 编辑（JSON 补丁式，未提供字段保留） | `fv edit hong-shao-rou --repo owner/recipes --json '{"stats":{"difficulty":4}}'` |
| `fv rm <id> [-y]` | 删除菜谱 | `fv rm hong-shao-rou --repo owner/recipes -y` |
| `fv gh push --recipe <id> [--json @file]` | 用 API 推单个菜谱文件（含图片） | `fv gh push --recipe hong-shao-rou --json @r.json` |

### 构建 / 其他

| 命令 | 说明 |
|---|---|
| `fv build [--force] [--output ./dist] [--asset-dir .flavor-vault/assets] [--ai-snapshot] [--endpoint <url>]` | ETL 生成静态站点（build 配置由 CI/workflow 传入；本地用于预览） |
| `fv init [--endpoint <url>]` | 生成 `config.example.yaml` 与 `.flavor-vault/`（可选） |
| `fv config get / set <key> <val>` | 查看/修改可选配置（endpoint / asset_dir / github.repo / github.branch） |
| `fv action list/show/clear` | 管理 `--action-id` 操作缓存（草稿续写） |
| `fv gh status/pr/release/workflow` | GitHub API 操作（只读/追加式，防冲突） |

## 菜谱 JSON 结构

```json
{
  "id": "hong-shao-rou",
  "name": "红烧肉",
  "description": "",
  "tags": ["热菜", "家常"],
  "kitchenware": ["炒锅"],
  "ingredients": { "main": [{"name": "五花肉", "amount": "500g"}], "side": [{"name": "冰糖"}] },
  "steps": [{"order": 1, "description": "焯水", "image_ref": "images/红烧肉-1-1.png"}],
  "media": { "cover": "", "images": [] },
  "stats": { "prep_time": 20, "cook_time": 70, "difficulty": 3 }
}
```
- 图片：`image_ref`/`cover`/`images` 可为本地路径（随单文件经 API 提交）或外部 URL
- 步骤图命名：`<菜谱名>-<步骤>-<序号>`（如 `红烧肉-1-1.png`）

## 数据契约（构建产物 `dist/data/`）

| 文件 | 内容 |
|---|---|
| `all.json` | 菜谱轻量列表（id/name/tags/kitchenware/ingredients/cover/prep_time/cook_time/difficulty） |
| `details/<id>.json` | 完整菜谱 |
| `filters.json` | 倒排索引（kitchenware/tags/ingredients → id 列表） |
| `search.json` | 搜索索引（全文拼合） |
| `meta.json` | 统计 + `endpoint`（构建时注入的默认读取地址） |
| `ai-corpus.json` | AI 语料（JSON Lines） |

## 自动化（GitHub Actions）

- `recipes` 分支推送 → 构建并部署 gh-pages（只更新 pages）
- `v*` tag → 构建 + 发布（多架构客户端 `fv-<os>-<arch>`，changelog 由 commit 生成；tag 含 `-` 为预览版 prerelease）
- `main` 不触发部署
- build 配置在 workflow 内：`fv build --force --output ./dist --asset-dir .flavor-vault/assets --ai-snapshot --endpoint $PAGES_BASE/data`

## AI 助手常用任务示例

```bash
# 1. 查菜谱
fv list --endpoint <url>
fv search 红烧 --endpoint <url>
fv show <id> --endpoint <url>

# 2. 新增菜谱（先构造合法 JSON：name/ingredients.main/steps/stats.difficulty 1-5 必填）
export GITHUB_TOKEN=ghp_xxx
fv add --repo <owner>/<repo> --branch recipes --json '{"name":"...", ...}'

# 3. 编辑（补丁式）
fv edit <id> --repo <owner>/<repo> --branch recipes --json '{"stats":{"difficulty":4}}'

# 4. 删除
fv rm <id> --repo <owner>/<repo> --branch recipes -y

# 5. 本地构建预览
fv build --force --output ./dist --asset-dir .flavor-vault/assets --ai-snapshot --endpoint <url>
```

> 校验规则：`name` 非空、`ingredients.main` 至少一项、`steps` 至少一步、`difficulty` 1–5。失败会提示并用 `--action-id` 缓存草稿供修正后重试。
