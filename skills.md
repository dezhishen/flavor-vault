---
name: flavor-vault
description: 'Flavor Vault 菜谱管理 CLI（fv）使用指南。Use when: 需要读取/查询菜谱（list/search/show/share/stats/filter），编辑/新增/删除菜谱（add/edit/rm），构建静态站点（build），或操作 GitHub 数据分支（gh）。提供命令与参数用法、配置与示例，供 AI 助手（如 OpenClaw）调用 fv 命令完成任务。'
---

# Flavor Vault · `fv` CLI 能力指南

Flavor Vault 是一个基于 **GitHub + 纯静态托管** 的菜谱管理系统。CLI 二进制为 `fv`（Go，免 cgo）。

- 数据仓库：菜谱以 JSON 单文件存放于 GitHub 数据分支（默认 `recipes`）
- 静态站点：由 GitHub Actions 构建并部署到 gh-pages（`dist/`）
- 前端：Vue3 + Element Plus，纯静态，读 `./data/*.json`

## 快速开始（如何开始）

> **AI 助手（OpenClaw 等）注意：所有 `fv` 命令都必须显式加 `-c <配置路径>`**，配置文件默认初始化在用户主目录 `~/.flavor-vault/config.yaml`（`fv init` 生成）。不要依赖当前工作目录的自动查找。

### 1. 安装 CLI（一条命令，基于 GitHub Release 二进制）

```bash
# Linux / macOS（自动识别系统与架构，下载最新 release 二进制）
mkdir -p ~/.local/bin && curl -fsSL "https://github.com/dezhishen/flavor-vault/releases/latest/download/fv-$(uname -s|tr A-Z a-z)-$(uname -m|sed 's/x86_64/amd64/;s/aarch64/arm64/')" -o ~/.local/bin/fv && chmod +x ~/.local/bin/fv && ~/.local/bin/fv --help

# Windows PowerShell
# Invoke-WebRequest -Uri "https://github.com/dezhishen/flavor-vault/releases/latest/download/fv-windows-amd64.exe" -OutFile "$HOME\.local\bin\fv.exe"
```

> 若 `~/.local/bin` 不在 PATH，请先 `export PATH="$HOME/.local/bin:$PATH"`。之后可用 `fv update` 自更新到最新版；源码构建为 `go build -o fv ./cmd/fv`。

### 2. 初始化（对话式，回车即用默认）

```bash
# 对话式：回车即用默认配置（只读即可用），生成 ~/.flavor-vault/config.yaml
fv init
# 需要编辑菜谱时再配置 GitHub：
fv init --github --repo <owner>/<repo> --branch recipes
# 或直接运行 fv add/edit/rm，按提示补全 GitHub 信息（-c 可指定配置位置）
```

### 3. 开始使用（务必带 `-c ~/.flavor-vault/config.yaml`）

```bash
CONFIG=~/.flavor-vault/config.yaml

# 只读：查询菜谱
fv list -c "$CONFIG"
fv search 红烧 -c "$CONFIG"
fv show <id> -c "$CONFIG"

# 读写（编辑菜谱）：需 GitHub 权限（GITHUB_TOKEN + 数据仓库）
export GITHUB_TOKEN=ghp_xxx
fv add -c "$CONFIG" --repo <owner>/<repo> --branch recipes                          # 交互式新增
fv edit -c "$CONFIG" --repo <owner>/<repo> --branch recipes --json '{"stats":{"difficulty":4}}'
fv rm -c "$CONFIG" --repo <owner>/<repo> --branch recipes -y
```

## 使用模式（两种）

| 模式 | 用途 | 需要什么 |
|---|---|---|
| **只读** | 查询菜谱 | `--endpoint <url>`（或 `FV_ENDPOINT`）；未配置时用默认端点 `https://fv.sdniu.top/data`；本地有 `dist/data` 时读本地 |
| **读写** | 新增/编辑/删除菜谱 | `--repo <owner/repo>`（或 `FV_REPO`）+ `--branch <branch>`（默认 `recipes`，或 `FV_BRANCH`）+ `GITHUB_TOKEN` |

> 提交作者：默认自动取**你的 `GITHUB_TOKEN` 对应 GitHub 账户**（`GET /user`，无公开邮箱时用 GitHub noreply 邮箱）。如需覆盖，在配置 `author.name` / `author.email`（`fv config set author.name ...`）。

> **提交确认**：`fv add` / `fv edit` / `fv gh push --recipe` 在真正推送前会**预览**菜谱内容（含待上传图片数）并请求确认（回车=提交，输入 `n` 取消）。**AI 助手非交互请统一加 `-y`/`--yes` 跳过确认**（避免卡在提示；无 stdin 时默认提交，但显式 `-y` 更明确）。

> 配置可选（也可全用参数/环境变量），但 **AI 助手请统一用 `fv init` 生成配置并以 `-c` 显式指定**（默认用户主目录 `~/.flavor-vault/config.yaml`），不要依赖工作目录自动查找。项目根另有 `config.example.yaml` 供参考。

## 命令一览

### 读取（只读，需 endpoint）

| 命令 | 说明 | 示例 |
|---|---|---|
| `fv list [-c <cfg>] [--tag 标签] [--json]` | 列出菜谱 | `fv list -c ~/.flavor-vault/config.yaml` |
| `fv show <id> [-c <cfg>] [--raw]` | 打印单菜谱详情 | `fv show hong-shao-rou -c ~/.flavor-vault/config.yaml` |
| `fv share <id> [-c <cfg>] [--out <file>] [--img <png>] [--no-img]` | 生成分享内容：Markdown（默认带封面/步骤图+菜谱链接，可直接发 IM/AI 助手；`--no-img` 纯文字）或 `--img` 导出 PNG 分享长图（适合不支持带图 Markdown 的平台） | `fv share hong-shao-rou -c ~/.flavor-vault/config.yaml --img ~/share.png` |
| `fv search <关键词...> [-c <cfg>] [--json]` | 全文搜索（菜名/食材/步骤等，多词 AND） | `fv search 鸡翅 烤箱 -c ~/.flavor-vault/config.yaml` |
| `fv stats [-c <cfg>] [--json]` | 统计（总数/标签/难度等） | `fv stats -c ~/.flavor-vault/config.yaml` |
| `fv filter --厨具 炒锅 --标签 凉菜 [-c <cfg>] [--json]` | 按倒排索引求交集 | `fv filter --食材 鸡翅 -c ~/.flavor-vault/config.yaml` |
| `fv ask <问题> [-c <cfg>] [--top N]` | AI 语料检索（ai-corpus.json） | `fv ask 有什么快手凉菜 -c ~/.flavor-vault/config.yaml` |

### 编辑（读写，需 GITHUB_TOKEN + repo/branch）

| 命令 | 说明 | 示例 |
|---|---|---|
| `fv add [-c <cfg>] [--json '...'\|@file] [--action-id X] [-y]` | 新增菜谱（交互式或 JSON），提交前预览+确认（`-y` 跳过） | `fv add -c ~/.flavor-vault/config.yaml --repo owner/recipes --branch recipes --json @r.json -y` |
| `fv edit <id> [-c <cfg>] [--json '{"stats":{"difficulty":4}}'] [-y]` | 编辑（JSON 补丁式，未提供字段保留），提交前预览+确认 | `fv edit hong-shao-rou -c ~/.flavor-vault/config.yaml --repo owner/recipes --json '{"stats":{"difficulty":4}}' -y` |
| `fv rm <id> [-c <cfg>] [-y]` | 删除菜谱（**同时清理其图片资产 `images/<id>/`**，无图静默跳过） | `fv rm hong-shao-rou -c ~/.flavor-vault/config.yaml --repo owner/recipes -y` |
| `fv gh push --recipe <id> [-c <cfg>] [--json @file] [-y]` | 用 API 推单个菜谱文件（含图片），提交前预览+确认（`-y` 跳过） | `fv gh push --recipe hong-shao-rou -c ~/.flavor-vault/config.yaml --json @r.json -y` |

### 构建 / 其他

| 命令 | 说明 |
|---|---|
| `go run ./cmd/build --sync --force` | 构建静态站点数据 dist/data+dist/assets（构建只在 CI 完成，fv 不含 build 命令；本地复刻 CI 用此入口） |
| `fv init [-c <path>] [-f] [--github] [--repo <owner/repo>] [--branch <b>]` | 对话式生成配置到 `~/.flavor-vault/config.yaml`（回车即用默认；`--github` 一并配置编辑仓库） |
| `fv config get / set <key> <val>` | 查看/修改可选配置（endpoint / asset_dir / github.repo / github.branch） |
| `fv action list/show/clear` | 管理 `--action-id` 操作缓存（草稿续写） |
| `fv update [--check] [--pre] [--version vX] [--repo owner/repo]` | 自更新到 GitHub Releases 最新版（公开仓库免 token；Windows 自动延迟替换）；`--pre` 更新/检查最新预览版（含预发布，方便测试） |
| `fv gh status/pr/release/workflow` | GitHub API 操作（只读/追加式，防冲突） |

## 菜谱 JSON 结构

```json
{
  "id": "hong-shao-rou",
  "name": "红烧肉",
  "description": "",
  "tags": ["热菜", "家常"],
  "kitchenware": ["炒锅"],
  "versions": [
    {
      "name": "经典版",
      "ingredients": {
        "main": [{"name": "五花肉", "amount": "500g", "alternatives": [{"name": "梅花肉", "amount": "500g", "note": "代替五花肉"}]}, {"name": "鹌鹑蛋", "amount": "10个", "note": "没有可省略（可选）"}],
        "side": [{"name": "冰糖", "amount": "20g"}]
      },
      "seasonings": [
        {"name": "香葱", "amount": "2根", "alternatives": [{"name": "香菜", "amount": "1把", "note": "代替香葱"}]}
      ],
      "steps": [{"order": 1, "description": "焯水", "image_ref": "images/hong-shao-rou/红烧肉-1-1.png"}],
      "media": { "cover": "images/hong-shao-rou/红烧肉-cover.jpg", "images": [] },
      "stats": { "prep_time": 20, "cook_time": 70, "difficulty": 3 }
    },
    {
      "name": "少油版",
      "ingredients": { "main": [{"name": "白萝卜", "amount": "1根"}], "side": [] },
      "seasonings": [],
      "steps": [{"order": 1, "description": "先煎出油"}],
      "media": { "cover": "", "images": [] },
      "stats": { "prep_time": 15, "cook_time": 45, "difficulty": 2 }
    }
  ]
}
```

- **统一多版本结构**：菜谱内容（食材/调料/步骤/media/统计）一律放 `versions`（至少 1 个版本），**顶层只保留元数据**（`id/name/description/tags/kitchenware`）；`fv add`/`fv edit` 统一输出多版本，历史单版本结构会自动迁移（`normalizeMultiVersion`），前端按 tab 展示多版本
- **食材**：`ingredients.main` 必选主料 / `side` 配菜辅料；每项可带 `note` 备注（非必须如"可省略/可选"写在备注里，无独立 `optional` 分组），可带 `alternatives` 可替换方案（如 梅花肉 代替 五花肉）
- **调料**：`seasonings` 每项 `name` 为方案一，`alternatives` 为备选方案（方案二/三，如用香菜代替香葱）
- **图片**：`image_ref`/`cover`/`images` 可为**本地路径**或外部 URL。给步骤配图：`fv edit <id> --json '{"steps":[{"order":1,"description":"...","image_ref":"/本地/图片.png"}]}'`——本地路径会被自动复制到 `images/<菜谱ID>/`（命名 `<菜谱名>-<步骤>-<序号>`，如 `images/hong-shao-rou/红烧肉-1-1.png`）并随菜谱经 API 上传；已分组引用（`images/<id>/...` 且本地有文件）幂等保留；`images/` 前缀本地无文件视为分支已有资产不重复上传；外部 URL 原样。同一菜谱的封面/过程图/步骤图集中存放于 `images/<菜谱ID>/` 目录。

## 构建（`cmd/build`，在 CI 中完成）

`fv` 制品不包含 build 命令；静态站点数据由独立构建器 `cmd/build` 在 CI 中执行，生成 ETL 产物 `dist/data/*.json`（`all.json` / `details/*.json` / `filters.json` / `search.json` / `meta.json` / `ai-corpus.json`）供前端与只读命令消费。参数由 CI/workflow 传入（`--output --asset-dir --ai-snapshot --endpoint`）。本地复刻 CI：`go run ./cmd/build --sync --force --output ./dist --asset-dir .flavor-vault/assets --ai-snapshot`（`--sync` 先从 recipes 数据分支拉取 recipes+assets 到本地布局）。

## AI 助手常用任务示例

> 全部命令都带 `-c ~/.flavor-vault/config.yaml`（配置在用户主目录；`fv init` 已生成）。

```bash
CONFIG=~/.flavor-vault/config.yaml

# 1. 查菜谱
fv list -c "$CONFIG"
fv search 红烧 -c "$CONFIG"
fv show <id> -c "$CONFIG"

# 2. 新增菜谱（先构造合法 JSON：name + 每版本 ingredients.main/steps/stats.difficulty 1-5 必填；提交前会预览+确认，-y 跳过）
export GITHUB_TOKEN=ghp_xxx
# 单版本（顶层字段即默认版本）
fv add -c "$CONFIG" --repo <owner>/<repo> --branch recipes -y --json '{"name":"...", "ingredients":{"main":[{"name":"材料","amount":"1"}]}, "steps":[{"order":1,"description":"做"}], "stats":{"prep_time":10,"cook_time":20,"difficulty":2}}'
# 多版本（versions 数组；或交互式 fv add 时选择“添加其他版本”逐个录入）
fv add -c "$CONFIG" --repo <owner>/<repo> --branch recipes -y --json '{"name":"...", "versions":[{"name":"经典版","ingredients":{"main":[{"name":"五花肉","amount":"500g"}]},"steps":[{"order":1,"description":"焯水"}],"stats":{"difficulty":3}},{"name":"少油版","ingredients":{"main":[{"name":"白萝卜","amount":"1根"}]},"steps":[{"order":1,"description":"煎"}],"stats":{"difficulty":2}}]}'

# 3. 编辑（补丁式；多版本菜谱默认编辑第一个版本，补丁含 versions 则整体替换）
fv edit -c "$CONFIG" <id> --repo <owner>/<repo> --branch recipes -y --json '{"stats":{"difficulty":4}}'

# 3b. 给步骤配图（image_ref 填本地图片路径，自动复制到 images/<id>/ 并按 <菜谱名>-<步骤>-<序号> 命名后上传；注意 steps 为整体替换，需带上全部步骤）
fv edit -c "$CONFIG" chao-jue-zi-su-xia --repo <owner>/<repo> --branch recipes -y --json '{"steps":[{"order":1,"description":"处理虾","image_ref":"/tmp/step1.jpg"},{"order":2,"description":"炒"}]}'

# 4. 删除（会同时清理该菜谱的图片资产 images/<id>/）
fv rm -c "$CONFIG" <id> --repo <owner>/<repo> --branch recipes -y

# 5. 自更新 / 检查预览版
fv update --check
fv update --check --pre    # 检查最新预览版（含预发布）
fv update --pre            # 更新到最新预览版（方便测试）

# 6. 生成分享内容（Markdown 带封面/步骤图 + 菜谱链接，可直接发 IM / AI 助手）
fv share <id> -c "$CONFIG" --out ~/share.md
fv share <id> -c "$CONFIG" --no-img          # 纯文字不带图
fv share <id> -c "$CONFIG" --img ~/share.png   # 导出 PNG 分享长图（非所有平台支持带图 Markdown）

# 7. 本地复刻 CI 构建（fv 不含 build 命令，用独立构建器）
go run ./cmd/build --sync --force --output ./dist --asset-dir .flavor-vault/assets --ai-snapshot --endpoint <url>
```

> 校验规则：`name` 非空；每个版本需 `ingredients.main` 至少一项、`steps` 至少一步、`difficulty` 1–5；调料与备选方案需有 `name`。失败会提示并用 `--action-id` 缓存草稿供修正后重试。
