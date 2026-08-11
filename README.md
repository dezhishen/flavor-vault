# 🍲 Flavor Vault

基于 **Git + 纯静态托管** 的菜谱管理工具。使用 Go 编写的 CLI 维护菜谱（每道菜谱一个独立 JSON），构建时执行 **ETL 流水线** 生成分片静态数据（倒排索引、标签分组、AI 快照等），最终由 **Vue 3 + Element Plus** 前端消费，呈现可筛选、可搜索的菜谱页面。无服务器、无数据库、无成本。

---

## ✨ 特性

- **极简存储**：菜谱为独立 JSON 文件，Git 友好，天然支持版本控制与协作。
- **插件化 ETL**：构建流水线由插件组成，每个插件生成独立数据集，可注册 CLI 子命令。
- **毫秒级查询**：预计算 `厨具 / 主要食材 / 标签` 倒排索引，多维条件内存交集。
- **AI 就绪**：`fv ask` 自然语言检索；构建生成 `ai-corpus.json`（JSON Lines）供云端 AI 拉取。
- **增量构建**：基于文件哈希 + TTL 的缓存系统，重复构建毫秒完成。
- **纯静态部署**：产物为 HTML/JS/CSS/JSON，一键部署 GitHub Pages / Cloudflare Pages。

---

## 🏗 技术栈

| 组件 | 选型 |
|------|------|
| CLI | Go 1.22+ · Cobra · Viper |
| 前端 | Vue 3 + TypeScript · Element Plus · Pinia · Vite · npm/pnpm |
| 部署 | GitHub Actions · GitHub Pages |
| 存储 | Git（菜谱 JSON） |

---

## 📦 快速开始

### 1. 构建 CLI

```bash
go build -o fv ./cmd/fv
```

### 2. 初始化

```bash
fv init
```

会在当前目录生成：

```
.flavor-vault/
├── config.yaml   # 标签白名单、厨具建议、缓存与插件配置
├── recipes/      # 菜谱 JSON（含 6 道示例菜谱）
└── cache/        # 构建缓存（自动生成，已被 .gitignore 忽略）
```

### 3. 添加菜谱（交互式）

```bash
fv add
```

### 4. 构建静态站点

```bash
fv build                # 增量构建（使用缓存）
fv build --force        # 强制全量重建
```

### 5. 本地预览

```bash
cd web
npm install
npm run dev            # 开发服务器（默认 5173 端口）
```

> 开发服务器已内置 `/data` 代理，会自动读取项目根 `dist/data` 下的构建产物。
> 使用 pnpm 亦可（`pnpm install && pnpm dev`）。

---

## 🖥 CLI 命令

| 命令 | 用途 | 示例 |
|------|------|------|
| `fv init [-c <path>] [-f]` | 初始化 `.flavor-vault/` 及默认 config | `fv init -c /path/to/config.yaml` |
| `fv add [--json <json>] [--action-id <id>]` | 创建菜谱（交互式或 JSON，支持草稿缓存） | `fv add --json @recipe.json --action-id a1` |
| `fv edit <id> [--json <patch>] [--action-id <id>]` | 编辑菜谱（$EDITOR 或 JSON 补丁，支持草稿缓存） | `fv edit hong-shao-rou --json '{"stats":{"difficulty":4}}'` |
| `fv rm <id> [--action-id <id>]` | 删除菜谱 | `fv rm hong-shao-rou` |
| `fv list [--tag <tag>] [--json]` | 列出菜谱 | `fv list --tag 凉菜` |
| `fv show <id>` | 打印完整菜谱（JSON） | `fv show hong-shao-rou` |
| `fv build [--force] [--output]` | 执行 ETL 流水线 | `fv build --force` |
| `fv filter --厨具 <kw> --标签 <tag> --食材 <ing>` | 倒排索引交集筛选 | `fv filter --厨具 炒锅 --标签 凉菜` |
| `fv stats [--json]` | 显示统计信息 | `fv stats` |
| `fv ask <query>` | AI 自然语言检索（本地缓存优先） | `fv ask "不用炒锅的凉菜"` |
| `fv push <msg> [--no-rebase] [-f]` | git 推送（自动 fetch/rebase 防冲突） | `fv push "添加红烧肉"` |
| `fv gh push <msg> [--dir <d>] [--branch <b>]` | 通过 GitHub API 推送（快进守卫） | `fv gh push "发布" --dir dist --branch gh-pages` |
| `fv gh status / pr / release / workflow` | GitHub 只读/追加式操作 | `fv gh status` |
| `fv action list/show/clear <id>` | 管理基于 action-id 的操作缓存 | `fv action list` |
| `fv version` | 显示版本 | `fv version` |

所有命令均支持 `--json` 输出，方便 AI 或脚本解析。

---

## 🎯 基于 `action-id` 的可续写操作

菜谱维护（`add` / `edit` / `rm`）支持全局参数 `--action-id`：把**操作参数**（菜谱草稿/补丁）缓存到 `/tmp/flavor-vaults/action-<id>.json`，**校验无误后才真正完成动作**（写入/删除）并自动清除缓存。这大幅降低 AI 或人进行多轮菜谱维护的难度——不必每次重发全部数据。

### 工作流

```bash
# 1. AI/人 提交一次操作（数据不完整，校验失败）
fv add --action-id a1 --json '{"name":"红烧肉"}'
#    → ⚠ 校验失败，草稿已缓存到 /tmp/flavor-vaults/action-a1.json

# 2. 查看/检查缓存
fv action list
fv action show a1

# 3. 修正后以相同 action-id 重试（自动恢复草稿并合并新数据）
fv add --action-id a1 --json @recipe-full.json
#    → ✔ 动作完成，已清除缓存；菜谱已创建
```

### 特性

- **草稿续写**：交互式 `fv add` 中断后，以相同 action-id 重新执行可基于缓存草稿续写（预填默认值）。
- **失败留痕**：校验未通过时参数保留在缓存中，并提示重试命令。
- **完成即清**：动作成功后自动清除缓存，避免残留。
- **JSON 直接输入**：`--json` 支持内联 JSON 或 `@文件路径`，适合 AI 批量提交。
- **编辑补丁**：`fv edit --json` 未提供的字段保持原值（补丁式更新）。
- **缓存管理**：`fv action list` 查看待处理项，`fv action show` 查看参数，`fv action clear` 手动清除。
- 缓存目录可通过环境变量 `FV_ACTION_DIR` 覆盖（默认 `/tmp/flavor-vaults`）。

---

## ⚙️ 配置文件与 `--config`

所有命令支持全局参数 `--config`/`-c` 指定配置文件路径，默认读取 `<项目根>/.flavor-vault/config.yaml`（从当前目录向上自动查找）。

```bash
fv list                      # 自动查找 .flavor-vault/config.yaml
fv -c /path/to/config.yaml list    # 使用指定配置文件（可跨目录工作）
fv --config .flavor-vault/config.yaml build
```

**路径解析规则**：
- 配置为标准布局 `<root>/.flavor-vault/config.yaml` → 项目根为 `.flavor-vault` 的上级目录；
- 其他自定义路径（如 `/data/vault/config.yaml`）→ 项目根为配置文件所在目录，菜谱/缓存位于 `<root>/.flavor-vault/`。

**初始化配置**：

```bash
fv init                          # 在当前目录初始化 .flavor-vault/ + 默认配置 + 示例菜谱
fv init -c /path/to/config.yaml  # 在指定位置初始化配置
fv init -f                       # 覆盖已存在的配置文件
fv init --separate-recipes       # 菜谱数据放到独立 recipes 分支（见下文）
fv init --separate-recipes --recipes-branch data   # 自定义分支名
```

---

## 🌿 独立菜谱分支（`--separate-recipes`）

默认菜谱与代码同在 `main`。若希望**菜谱数据独立成一个分支**（只含 `recipes/*.json` 与 `config.yaml`，不含代码），初始化时一条命令即可完成全部配置：

```bash
fv init --separate-recipes
```

初始化会：
1. 写入配置 `github.recipes_branch: recipes`；
2. 用 git 创建**只含菜谱与配置**的孤立 `recipes` 分支（不打扰当前分支）；
3. 建立本地 worktree `<root>/.recipes`（该分支的检出）；
4. 让 `main` 忽略 `.flavor-vault/recipes/` 与 `.recipes/`（菜谱只在独立分支维护）。

**之后的维护完全透明**（CLI 自动从 worktree 读写）：

```bash
fv add / fv edit / fv rm          # 直接操作 .recipes 里的菜谱文件
fv list / fv show / fv build      # 从 worktree 读取，行为不变
fv gh push --recipe hong-shao-rou # 自动提交/更新到 recipes 分支（默认分支已切换）
```

**发布流程**：
- 首次：`git push -u origin recipes`（把菜谱分支推上去）
- 之后：`fv gh push --recipe <id>` 直接改远端 recipes 分支；代码仍走 `main`
- CI 构建时会把 `recipes` 分支合并进工作区再 `fv build`，代码与数据互不干扰、各自独立演进

> 取舍：独立分支带来数据/代码关注点分离，代价是本地多一个 worktree、CI 多一次分支合并。单用户也可继续用默认"同分支"模式，两种方式 CLI 行为一致。

**配置项**（`config.yaml`）：

```yaml
tags: [凉菜, 热菜, 川菜, ...]        # 标签白名单（fv add 校验）
kitchenware: [炒锅, 砂锅, ...]      # 厨具建议列表
cache:
  enabled: true
  ttl_seconds: 86400
  plugins: { facet_indexer: 3600 }
output_dir: ./dist                 # 构建输出目录（可自定义）
ai_snapshot: true                  # 是否生成 AI 快照

# GitHub 集成（fv gh / fv push）
github:
  token: ""                        # 访问令牌（推荐用环境变量 GITHUB_TOKEN）
  owner: ""                        # 仓库属主（默认从 git remote 推断）
  repo: ""                         # 仓库名（默认从 git remote 推断）
  default_branch: main             # 默认分支
  auto_rebase: true                # push 前自动 fetch + rebase 防冲突
```

---

## 🔐 推送与冲突避免

集成 GitHub 客户端（`go-github`）后，推送面临 5 类冲突风险。Flavor Vault 通过**单一写者 + 快进守卫**来规避：

| 冲突来源 | 说明 | 规避机制 |
|---------|------|---------|
| **双写通道** | git transport 与 GitHub API 是两套写远端的通道，同时用会"分叉大脑" | 职责分离：`fv push`（git 通道）负责同步本地 git 对象；`fv gh` 只做只读/追加式操作；`fv gh push` 走 API 但严格快进 |
| **非快进** | 远端已被他人推进，直接 push 被拒或 force 覆盖他人提交 | `fv push` 先 `git fetch` 检测落后 → 自动 `git pull --rebase`（`--no-rebase` 则安全中止）；`-f` 时用 `--force-with-lease`（带预期 SHA 校验）而非裸 `--force` |
| **并发推送** | 多个 CLI 进程同时推送互相覆盖 | 推送锁 `.flavor-vault/push.lock`（独占文件 + 过期接管），实现单一写者 |
| **API 竞态** | `fv gh push` 创建提交期间远端被推进 | 快进守卫：updateRef 前重新校验父提交 SHA == 远端 tip，不等则返回 `ErrNonFastForward` 中止（乐观锁），绝不强推 |
| **凭据冲突** | git 走系统凭据/SSH，API 走 PAT，账号/scope 不一致 | 统一 token 来源（`GITHUB_TOKEN` 环境变量或 `config.github.token`），owner/repo 从配置或 `git remote` 自动推断 |

### 两种推送方式怎么选

```bash
# 方式一：git 通道（推荐日常使用，同步整个仓库）
fv push "添加红烧肉"

# 方式二：GitHub API 通道（无本地 git 或只推产物到 gh-pages）
fv gh push "发布站点" --dir dist --branch gh-pages --author "Name <email>"
```

`fv gh` 其余命令均为只读或追加式，不写分支 ref，天然无冲突：

```bash
fv gh status                            # 仓库信息 / 最新提交 / CI 状态（只读）
fv gh pr --title "..." --head feat --base main   # 创建 PR（追加）
fv gh release --tag v1.0 --name "v1.0"           # 创建 Release（追加）
fv gh workflow --workflow deploy.yml             # 触发 CI（追加）
```

### 按"文件思路"用 gh 提交/更新菜谱

`fv gh push --recipe <id>` 只提交**单个菜谱文件** `recipes/<id>.json`（快进守卫，不碰其他文件），内容可来自 `--json` 或本地文件，提交前自动做与本地 `fv add` 相同的校验：

```bash
# 新增：提供菜谱内容（内联 JSON 或 @文件）
fv gh push "新增红烧肉" --recipe hong-shao-rou --json @recipe.json

# 更新：复用本地 recipes/<id>.json（远端已存在则自动识别为"更新"）
fv gh push "调整难度" --recipe hong-shao-rou
```

每次 `--recipe` 推送都只产生一个文件的 commit，`fv build` 在 CI 上自动重建派生数据，与本地维护完全同构。

---

## 🔌 插件系统

每个插件实现统一接口，在 `fv build` 时顺序执行：

```go
type Plugin interface {
    Name() string
    Build(ctx *BuildContext) error
    RegisterCommands(root *cobra.Command) error
}
```

| 插件 | 职责 | 输出 |
|------|------|------|
| `validator` | 校验必填字段与标签白名单 | — |
| `facet_indexer` | 厨具/食材/标签倒排索引 | `data/filters.json` · 注册 `fv filter` |
| `tag_indexer` | 按标签分组 | `data/by-tag/*.json` |
| `detail_splitter` | 详情拆分到独立文件 | `data/details/{id}.json` |
| `stats_collector` | 统计与轻量清单 | `data/meta.json` · `data/all.json` · 注册 `fv stats` |
| `ai_exporter` | AI 精简快照（JSON Lines） | `data/ai-corpus.json` |

新增插件只需实现接口并在 `cmd/fv/main.go` 或 `internal/cli/root.go` 中注册，无需改动既有代码。

### 缓存机制

每个插件通过 `cache.CacheManager` 集成缓存：

1. 计算依赖文件（菜谱 + config.yaml）的 MD5 哈希。
2. 命中缓存（哈希一致 && 未超 TTL）则直接复用，跳过重建。
3. 未命中或 `--force` 时全量重建并写回缓存。

TTL 可全局或按插件配置（见 `config.yaml` 的 `cache.plugins`）。

---

## 🗂 构建产物（dist/）

```
dist/
├── index.html          # 前端入口
├── assets/             # JS/CSS（前端构建产物）
└── data/               # ETL 生成数据
    ├── meta.json       # 统计信息
    ├── all.json        # 全部菜谱轻量清单
    ├── filters.json    # 倒排索引
    ├── by-tag/*.json   # 标签分组
    ├── details/*.json  # 菜谱详情（懒加载）
    └── ai-corpus.json  # AI 快照
```

---

## 🤖 AI 使用场景

### 本地 AI（Cursor / Claude Desktop 等）

```bash
# 结构化筛选
fv filter --厨具 蒸箱 --标签 凉菜 --json

# 自然语言检索（关键词打分 + 否定词过滤）
fv ask "不用炒锅的凉菜"
fv ask "烤箱能做的甜点" --json
```

### 云端 AI（无法访问本地文件）

构建时生成的 `ai-corpus.json`（JSON Lines）可部署到静态托管。AI 通过 System Prompt 获取该 URL，首次对话时拉取到上下文，后续检索全部在内存中完成。

---

## 🚀 部署（GitHub Pages）

推送到 `main` 分支后，GitHub Actions 自动完成：构建 CLI → 构建 Vue → 执行 ETL → 发布到 `gh-pages` 分支。

1. 在仓库 **Settings → Pages** 中选择 `gh-pages` 分支作为源。
2. 推送代码即可，无需手动操作。

手动部署：

```bash
fv build --force
git add -A && git commit -m "build" && git push origin main:gh-pages
```

---

## 🧪 开发与测试

```bash
# Go 单元测试（store / cache / pipeline / plugins / utils）
go test ./...

# 前端类型检查
cd web && npm run typecheck

# 前端构建
cd web && npm run build
```

端到端测试：在临时目录 `fv init` → `fv add` → `fv build`，检查 `dist/` 输出。

---

## 📁 项目结构

```
flavor-vault/
├── cmd/fv/                  # CLI 入口
├── internal/
│   ├── models/              # 数据模型
│   ├── store/               # 存储层（加载/CRUD）
│   ├── cache/               # 缓存管理器
│   ├── pipeline/            # 插件框架与调度器
│   ├── plugins/             # 6 个内置插件
│   ├── cli/                 # Cobra 命令
│   ├── utils/               # 哈希、交集算法
│   └── vault/               # 项目根定位与配置
├── web/                     # Vue 3 前端
├── .github/workflows/       # 自动构建部署
└── .flavor-vault/           # 本地数据目录
```

---

## 📄 License

MIT
