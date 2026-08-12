# 🍲 Flavor Vault

基于 **Git + 纯静态托管** 的菜谱管理工具。使用 Go 编写的 CLI 维护菜谱（每道菜谱一个独立 JSON，经 GitHub API 单文件操作），构建时执行 **ETL 流水线** 生成分片静态数据（倒排索引、标签分组、搜索、AI 快照等），最终由 **Vue 3 + Element Plus** 前端消费，呈现可筛选、可搜索的菜谱页面。无服务器、无数据库、无成本。

---

## ✨ 特性

- **极简存储**：菜谱为独立 JSON 文件，Git 友好，天然支持版本控制与协作；编辑经 GitHub API 单文件提交，无需本地克隆。
- **插件化 ETL**：构建流水线由插件组成，每个插件生成独立数据集，可注册 CLI 子命令。
- **毫秒级查询**：预计算 `厨具 / 主要食材 / 标签` 倒排索引，多维条件内存交集。
- **全文搜索**：构建生成 `search.json`，前端搜索框与 `fv search` 共用同一份索引。
- **AI 就绪**：`fv ask` 自然语言检索；构建生成 `ai-corpus.json`（JSON Lines）供云端 AI 拉取。
- **增量构建**：基于文件哈希 + TTL 的缓存系统，重复构建毫秒完成。
- **纯静态部署**：产物为 HTML/JS/CSS/JSON，由 GitHub Actions 部署到 gh-pages。
- **自更新**：`fv update` 从 GitHub Releases 拉取当前平台二进制并原子替换；`--pre` 可更新到最新预览版（含预发布，便于测试）。

---

## 🏗 技术栈

| 组件 | 选型 |
|------|------|
| CLI | Go 1.25+ · Cobra · Viper · go-github |
| 前端 | Vue 3 + TypeScript · Element Plus · Pinia · Vite |
| 部署 | GitHub Actions · GitHub Pages |
| 存储 | Git（菜谱 JSON 于独立 `recipes` 分支） |

---

## 📦 快速开始

### 一条命令安装（基于 Release 二进制，OpenClaw 等任何环境可用）

```bash
mkdir -p ~/.local/bin && curl -fsSL "https://github.com/dezhishen/flavor-vault/releases/latest/download/fv-$(uname -s|tr A-Z a-z)-$(uname -m|sed 's/x86_64/amd64/;s/aarch64/arm64/')" -o ~/.local/bin/fv && chmod +x ~/.local/bin/fv && ~/.local/bin/fv --help
```

> 自动识别系统与架构（`fv-linux-amd64` / `fv-darwin-arm64` / `fv-windows-amd64.exe` 等，见 Releases 页）。若 `~/.local/bin` 不在 PATH，先执行 `export PATH="$HOME/.local/bin:$PATH"`。之后用 `fv update` 自更新到最新版；源码构建为 `go build -o fv ./cmd/fv`。

### 初始化（对话式，回车即用默认）

```bash
fv init                                    # 对话式：回车即用默认配置（只读即可用），生成 ~/.flavor-vault/config.yaml
fv init --github --repo <owner>/<repo>     # 非交互：一并配置编辑仓库（-c 指定位置，-f 覆盖）
```

> 只需只读时，`fv init` 回车到底即可；以后要新增/编辑/删除菜谱时，`fv add`/`fv edit`/`fv rm` 会按提示补全 GitHub 信息（或 `fv init --github` 一次性配置）。

### 两种使用模式

| 模式 | 用途 | 需要什么 |
|---|---|---|
| **只读** | 查询菜谱 | 无需配置（默认端点 `https://fv.sdniu.top/data`；本地有 `dist/data` 时读本地） |
| **读写** | 新增/编辑/删除菜谱 | `GITHUB_TOKEN` + `--repo owner/repo` + `--branch recipes`（或写入配置） |

**多版本**：同一道菜可含多个 `versions`（每版本独立食材/调料/步骤/统计）。`fv add` 交互式选择“添加其他版本”逐个录入，或用 `--json` 直接提供 `versions` 数组；`fv edit --json` 默认编辑第一个版本（补丁含 `versions` 则整体替换）。版本内食材分 `main`（必选主料）/`side`（配菜辅料），非必须通过条目 `note` 备注表达（如"可省略"）；调料 `seasonings[].alternatives` 为备选方案（如香菜代替香葱）。

> **AI 助手（OpenClaw 等）请统一 `fv init` 生成配置，所有命令显式 `-c ~/.flavor-vault/config.yaml`**，不要依赖工作目录自动查找。

```bash
# 只读：查询菜谱
fv list -c ~/.flavor-vault/config.yaml
fv search 红烧 -c ~/.flavor-vault/config.yaml
fv show <id> -c ~/.flavor-vault/config.yaml
fv share <id> -c ~/.flavor-vault/config.yaml --out ~/share.md   # 生成分享消息（Markdown 带图 + 菜谱链接，可直接发 IM/AI 助手）
fv share <id> -c ~/.flavor-vault/config.yaml --img ~/share.png   # 导出 PNG 分享长图（非所有平台都支持带图 Markdown）

# 读写：编辑菜谱（提交作者自动取自你的 GITHUB_TOKEN 对应账户）
export GITHUB_TOKEN=ghp_xxx
fv add  -c ~/.flavor-vault/config.yaml --repo <owner>/<repo> --branch recipes
fv edit <id> -c ~/.flavor-vault/config.yaml --repo <owner>/<repo> --branch recipes --json '{"stats":{"difficulty":4}}'
fv rm   <id> -c ~/.flavor-vault/config.yaml --repo <owner>/<repo> --branch recipes -y
```

---

## 🖥 CLI 命令

| 命令 | 用途 |
|------|------|
| `fv init [-c <path>] [-f] [--endpoint <url>]` | 生成配置到 `~/.flavor-vault/config.yaml` |
| `fv list [-c <cfg>] [--tag 标签] [--json]` | 列出菜谱 |
| `fv show <id> [-c <cfg>] [--raw]` | 打印单菜谱详情 |
| `fv share <id> [-c <cfg>] [--format md\|png\|plain\|all] [--out <file>] [--img <png>] [--no-img]` | 生成分享内容：Markdown（默认，带封面/步骤图+菜谱链接）、`--format png` PNG 长图（含步骤图+底部菜谱二维码）、`--format plain` 纯文本（无 Markdown 标记）、`--format all` md+png；`--no-img` 纯文字 |
| `fv search <关键词...> [-c <cfg>] [--json]` | 全文搜索（多词 AND） |
| `fv filter --厨具 炒锅 --标签 凉菜 [-c <cfg>] [--json]` | 倒排索引交集筛选 |
| `fv stats [-c <cfg>] [--json]` | 统计信息 |
| `fv ask <问题> [-c <cfg>] [--top N]` | AI 语料检索 |
| `fv add [-c <cfg>] [--json '...'\|@file] [--action-id X] [-y]` | 创建菜谱（交互式/JSON，提交前预览+确认，`-y` 跳过） |
| `fv edit <id> [-c <cfg>] [--json <patch>] [--action-id X] [-y]` | 编辑菜谱（JSON 补丁式，提交前预览+确认） |
| `fv rm <id> [-c <cfg>] [-y]` | 删除菜谱（同时清理其图片资产 `assets/images/<id>/`） |
| `fv gh push --recipe <id> [-y]` | 用 API 推送单个菜谱文件（含图片，提交前预览+确认） |
| `fv gh status / pr / release / workflow` | GitHub 只读/追加式操作 |
| `fv config get/set <key> <val>` | 查看/修改配置（endpoint / asset_dir / author.* / github.*） |
| `fv action list/show/clear` | 管理 `--action-id` 操作缓存（草稿续写） |
| `fv update [--check] [--pre] [--version vX]` | 自更新到 GitHub Releases 最新版；`--pre` 更新到预览版（含预发布，便于测试） |
| `fv version` | 显示版本 |

**全局参数**：`-c/--config`（配置路径）、`--endpoint`（读，亦 `FV_ENDPOINT`）、`--repo/--branch`（写，亦 `FV_REPO/FV_BRANCH`）、`--action-id`（操作缓存 ID）。所有命令支持 `--json` 输出，方便 AI/脚本解析。

---

## 🎯 基于 `action-id` 的可续写操作

菜谱维护（`add` / `edit` / `rm`）支持全局参数 `--action-id`：把**操作参数**（菜谱草稿/补丁）缓存到 `<系统临时目录>/flavor-vaults/action-<id>.json`，**校验无误后才真正完成动作**（写入/删除）并自动清除缓存。这大幅降低 AI 或人进行多轮菜谱维护的难度——不必每次重发全部数据。

```bash
# 1. AI/人 提交一次操作（数据不完整，校验失败）
fv add --action-id a1 --json '{"name":"红烧肉"}'
#    → ⚠ 校验失败，草稿已缓存

# 2. 查看/检查缓存
fv action list
fv action show a1

# 3. 修正后以相同 action-id 重试（自动恢复草稿并合并新数据）
fv add --action-id a1 --json @recipe-full.json
#    → ✔ 动作完成，已清除缓存
```

- **草稿续写**：交互式 `fv add` 中断后以相同 action-id 重试可基于缓存续写。
- **失败留痕**：校验未通过时参数保留在缓存中并提示重试命令；**成功即清**。
- **JSON 直接输入**：`--json` 支持内联 JSON 或 `@文件路径`；`fv edit --json` 为补丁式（未提供字段保留）。
- 缓存目录默认 `<os.TempDir>/flavor-vaults`（Windows 为 `%TEMP%`），可用环境变量 `FV_ACTION_DIR` 覆盖。

---

## ⚙️ 配置（用户主目录）

默认配置在 **`~/.flavor-vault/config.yaml`**（`fv init` 生成；`-c` 指定其他位置）。项目根另有 `config.example.yaml` 示例。也可直接用参数/环境变量，不写配置文件。

```yaml
endpoint: ""            # 只读数据地址；留空用默认 https://fv.sdniu.top/data
asset_dir: .flavor-vault/assets
author:
  name: ""              # 提交作者；留空自动取 GITHUB_TOKEN 对应账户
  email: ""
github:
  token: ""             # 或环境变量 GITHUB_TOKEN
  repo: ""              # 菜谱仓库 owner/repo
  branch: recipes       # 数据分支
```

`fv config get` 查看当前生效配置，`fv config set <key> <value>` 修改（endpoint / asset_dir / author.name / author.email / github.repo / github.branch）。

---

## 🌿 数据与部署

- **数据分支**：菜谱存于独立 `recipes` 分支（`recipes/<id>.json` 单文件 + `assets/`），相当于可 fork / 私有化的数据仓库；同一菜谱的图片集中存放于 `assets/images/<id>/`（如 `assets/images/hong-shao-rou/`）。
- **触发**：`recipes` 分支有变动 → GitHub Actions 构建并部署 gh-pages（只更新页面）；推送 `v*` tag → 构建 + 发布多架构客户端 Release 并更新页面（tag 含 `-` 为预览版 prerelease）；`main` 不触发部署。
- **构建**：静态站点数据由独立构建器 `cmd/build` 在 CI 中完成（`go build -o build ./cmd/build && ./build --force --output ./dist --asset-dir .flavor-vault/assets --ai-snapshot --endpoint https://fv.sdniu.top/data`），配置默认值 = 当前运行仓库的 GitHub 信息；`fv` CLI 制品不包含 build 命令。

---

## 🗂 构建产物（dist/）

```
dist/
├── index.html / assets/   # 前端产物
└── data/                  # ETL 数据
    ├── meta.json          # 统计 + 默认 endpoint
    ├── all.json           # 菜谱轻量清单（含全部版本的主要食材）
    ├── filters.json       # 倒排索引
    ├── search.json        # 全文搜索索引（聚合所有版本食材/调料/步骤）
    ├── by-tag/*.json      # 标签分组
    ├── details/*.json     # 菜谱详情（含多版本 versions/调料 seasonings）
    └── ai-corpus.json     # AI 快照（JSON Lines）
```

> 菜谱数据模型：**统一多版本**——内容（食材/调料/步骤/media/统计）一律放 `versions`（至少 1 个版本），顶层只保留元数据（`id/name/description/tags/kitchenware`）；`fv add`/`fv edit` 统一输出多版本，历史单版本结构自动迁移；食材分 `main`（必选主料）/`side`（配菜辅料），非必须通过 `note` 备注表达；调料 `seasonings` 每项可带 `alternatives` 备选方案（如香菜代替香葱）。

---

## 🤖 AI 使用场景

```bash
# 本地：结构化筛选 / 自然语言检索
fv filter --厨具 蒸箱 --标签 凉菜 --json
fv ask "不用炒锅的凉菜" --json
```

- **云端 AI**：构建生成的 `ai-corpus.json`（JSON Lines）部署到静态托管，AI 通过 System Prompt 获取 URL，首次对话拉取到上下文后全部内存检索。

---

## 🧪 开发与测试

```bash
# Go 单元测试（store / cache / pipeline / plugins / utils）
go test ./...

# 前端类型检查 + 构建（pnpm 11）
cd web && pnpm run typecheck
cd web && pnpm run build
```

**本地预览**：

```bash
go run ./cmd/build --sync --force     # 从 recipes 数据分支拉取数据 + 生成 dist/data 与 dist/assets（--sync 可省略）
cd web && pnpm install && pnpm run dev  # 开发服务器（5173，/data 与 /assets 代理到 dist/）
```
