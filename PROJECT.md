# Flavor Vault 项目设计文档

## 1. 项目概述

**Flavor Vault** 是一个基于 CLI 的菜谱管理工具，通过 **Git + 纯静态托管**（如 GitHub Pages）实现无服务器部署。用户通过 Go 编写的命令行工具维护菜谱（每个菜谱独立 JSON），CLI 在构建时执行 ETL 生成分片静态数据（倒排索引、标签分组等），最终由 Vue 3 + Element Plus 前端消费这些数据，呈现可筛选、可搜索的菜谱展示页。AI 助手（本地或云端）可通过 CLI 命令或直接读取公开 JSON 快照实现智能检索。

### 核心目标
- **极简存储**：菜谱以独立 JSON 文件存放，Git 友好。
- **插件化 ETL**：构建流水线由插件组成，每个插件生成独立数据集，并可注册 CLI 子命令。
- **高效查询**：通过预计算的倒排索引（厨具/主要食材/标签）实现毫秒级多维度交集筛选。
- **AI 就绪**：提供专门的数据快照和 CLI 查询接口，降低 AI 调用延迟。
- **纯静态部署**：所有产物为 HTML/JS/CSS/JSON，可部署到任何静态托管服务。

---

## 2. 技术选型

| 组件 | 选型 | 说明 |
|------|------|------|
| CLI 开发语言 | Go 1.22+ | 跨平台单文件，性能优异，适合构建工具 |
| CLI 框架 | Cobra + Viper | 命令行解析 + 配置管理 |
| 交互式表单 | go-survey 或 bubbletea | 用于 `fv add` 等交互命令 |
| 前端框架 | Vue 3 + TypeScript | 组合式 API，轻量 |
| UI 库 | Element Plus | 成熟组件库，快速搭建管理界面 |
| 包管理 | pnpm | 更快的依赖安装和磁盘节省 |
| 构建工具 | Vite | 极速开发与构建 |
| 静态部署 | GitHub Pages / Cloudflare Pages | 无服务器，全球 CDN |
| CI/CD | GitHub Actions | 自动化 ETL + 部署 |
| 版本控制 | Git | 数据与代码统一管理 |

---

## 3. 项目结构

```
flavor-vault/
├── cmd/
│   └── fv/                     # CLI 主入口
│       └── main.go
├── internal/
│   ├── models/                 # 数据模型
│   │   ├── recipe.go           # 菜谱结构
│   │   └── config.go           # 全局配置结构
│   ├── store/                  # 本地存储层
│   │   ├── loader.go           # 加载所有菜谱，生成 manifest
│   │   └── recipe_file.go      # 单个菜谱的 CRUD（读写 JSON）
│   ├── cache/                  # 缓存管理器
│   │   └── manager.go          # 缓存读写、TTL、依赖哈希校验
│   ├── pipeline/               # ETL 插件系统
│   │   ├── plugin.go           # 插件接口定义
│   │   ├── scheduler.go        # 调度器（顺序执行插件）
│   │   └── context.go          # BuildContext 传递依赖
│   ├── plugins/                # 内置插件实现
│   │   ├── facet_indexer.go    # 厨具/食材/标签倒排索引
│   │   ├── tag_indexer.go      # 按标签分组
│   │   ├── detail_splitter.go  # 拆分详情到独立文件
│   │   ├── stats_collector.go  # 统计信息（总数、常用厨具等）
│   │   ├── ai_exporter.go      # 生成 AI 快照 ai-corpus.json
│   │   └── validator.go        # 校验菜谱必填字段
│   ├── cli/                    # Cobra 命令实现
│   │   ├── add.go
│   │   ├── build.go
│   │   ├── list.go
│   │   ├── show.go
│   │   ├── filter.go           # 调用 facet_indexer 查询
│   │   ├── ask.go              # AI 专用智能检索
│   │   └── push.go             # Git 推送
│   └── utils/                  # 通用工具
│       ├── hash.go             # 文件哈希
│       └── intersect.go        # 有序切片交集算法
├── web/                        # Vue 前端项目
│   ├── src/
│   │   ├── components/         # UI 组件（筛选面板、菜谱列表、详情）
│   │   ├── composables/        # 数据加载（useRecipe, useFilter）
│   │   ├── stores/             # Pinia 状态管理（加载索引、选中的过滤条件）
│   │   ├── App.vue
│   │   └── main.ts
│   ├── package.json
│   ├── pnpm-lock.yaml
│   └── vite.config.ts
├── .flavor-vault/              # 本地数据目录（用户工作区）
│   ├── recipes/                # 菜谱 JSON（用户维护）
│   │   └── *.json
│   ├── config.yaml             # CLI 配置（标签白名单、插件设置）
│   └── cache/                  # 构建缓存（自动生成）
│       └── <plugin>/           # 各插件的缓存数据与 meta.json
├── dist/                       # 构建输出（部署内容）
│   ├── data/                   # ETL 生成的 JSON 数据
│   │   ├── meta.json
│   │   ├── filters.json        # 倒排索引（厨具/食材/标签）
│   │   ├── by-tag/*.json
│   │   ├── details/*.json
│   │   └── ai-corpus.json
│   ├── assets/                 # 前端静态资源（JS/CSS）
│   └── index.html
├── .github/
│   └── workflows/
│       └── deploy.yml          # GitHub Actions 自动构建部署
├── go.mod
└── README.md
```

---

## 4. 数据模型

### 4.1 菜谱模型（`models/recipe.go`）

```go
type Recipe struct {
    ID          string       `json:"id"`           // 唯一标识（建议拼音或 UUID）
    Name        string       `json:"name"`         // 菜名
    Description string       `json:"description"`  // 简介
    Tags        []string     `json:"tags"`         // 标签（如 "凉菜","川菜"）
    Kitchenware []string     `json:"kitchenware"`  // 厨具（如 "炒锅","砂锅"）
    Ingredients Ingredients `json:"ingredients"`
    Steps       []Step       `json:"steps"`
    Media       Media        `json:"media"`
    Stats       Stats        `json:"stats"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    // 内部使用，不序列化
    FilePath    string       `json:"-"`            // 源文件路径
}

type Ingredients struct {
    Main []Ingredient `json:"main"`
    Side []Ingredient `json:"side"` // 配菜/辅料
}

type Ingredient struct {
    Name   string `json:"name"`
    Amount string `json:"amount"` // 如 "500g", "适量"
}

type Step struct {
    Order       int    `json:"order"`
    Description string `json:"description"`
    ImageRef    string `json:"image_ref,omitempty"` // 步骤图（可选）
}

type Media struct {
    Cover    string   `json:"cover"`               // 封面图路径
    Images   []string `json:"images"`              // 过程图列表
    VideoURL string   `json:"video_url,omitempty"` // 外部视频链接
}

type Stats struct {
    PrepTime   int `json:"prep_time"`   // 准备分钟
    CookTime   int `json:"cook_time"`   // 烹饪分钟
    Difficulty int `json:"difficulty"`  // 1-5
}
```

### 4.2 全局配置（`.flavor-vault/config.yaml`）

```yaml
# 标签白名单（fv add 时校验）
tags:
  - 凉菜
  - 热菜
  - 川菜
  - 粤菜
  - 家常
  - 下饭菜
  - 快手
  - 宴客

# 厨具建议列表（用于交互式提示）
kitchenware:
  - 炒锅
  - 砂锅
  - 蒸箱
  - 烤箱
  - 平底锅

# 缓存配置
cache:
  enabled: true
  ttl_seconds: 86400            # 全局 TTL（24h）
  # 可单独覆盖某个插件
  plugins:
    facet_indexer:
      ttl_seconds: 3600         # 1h

# 构建输出目录（默认为 ./dist）
output_dir: ./dist

# 是否生成 AI 快照
ai_snapshot: true
```

---

## 5. CLI 命令设计

所有命令基于 Cobra，根命令为 `fv`。

| 命令 | 用途 | 示例 |
|------|------|------|
| `fv init` | 在当前目录初始化 `.flavor-vault/` 及默认 config | `fv init` |
| `fv add` | 交互式创建新菜谱，自动生成 ID 并写入 `recipes/{id}.json` | `fv add` |
| `fv edit <id>` | 编辑已有菜谱（通过 $EDITOR 或交互式修改） | `fv edit hong-shao-rou` |
| `fv rm <id>` | 删除菜谱 JSON | `fv rm hong-shao-rou` |
| `fv list [--tag <tag>]` | 列出所有菜谱，支持按标签过滤 | `fv list --tag 凉菜` |
| `fv show <id>` | 打印完整菜谱（格式化 JSON） | `fv show hong-shao-rou` |
| `fv build [--force] [--incremental]` | 执行 ETL 流水线，生成 `dist/` | `fv build --force` |
| `fv filter --厨具 <kw> --标签 <tag> --食材 <ing>` | 基于倒排索引求交集，返回匹配的菜谱 ID | `fv filter --厨具 炒锅 --标签 凉菜` |
| `fv ask <query>` | AI 智能查询（自然语言），输出匹配菜谱（本地缓存优先） | `fv ask "不用烤箱的甜点"` |
| `fv push <message>` | 执行 `git add .` + `git commit` + `git push` | `fv push "添加红烧肉"` |
| `fv version` | 显示版本信息 | `fv version` |

所有命令支持 `--json` 输出，方便 AI 解析（例如 `fv list --json` 输出 JSON 数组）。

---

## 6. 插件系统

### 6.1 插件接口（`pipeline/plugin.go`）

```go
package pipeline

import (
    "github.com/spf13/cobra"
    "flavor-vault/internal/models"
)

type Plugin interface {
    // 插件唯一标识
    Name() string
    // 构建阶段执行（所有插件顺序执行）
    Build(ctx *BuildContext) error
    // 向根命令注册子命令（可选）
    RegisterCommands(root *cobra.Command) error
}

// 构建上下文
type BuildContext struct {
    Recipes   []*models.Recipe         // 所有菜谱
    OutputDir string                   // 输出根目录（如 ./dist）
    Force     bool                     // 是否强制重建
    Options   map[string]interface{}   // 额外参数
}
```

### 6.2 内置插件清单

| 插件名 | 职责 | 输出产物 | 是否注册命令 |
|--------|------|----------|-------------|
| `validator` | 校验每个菜谱的必填字段、标签合法性，不合法则中断构建 | 无（仅日志） | 否 |
| `facet_indexer` | 构建 `厨具/主要食材/标签` 的倒排索引，供 `fv filter` 查询 | `dist/data/filters.json` | `fv filter` |
| `tag_indexer` | 按标签分组，生成每个标签下的菜谱列表（轻量） | `dist/data/by-tag/*.json` | 否 |
| `detail_splitter` | 将完整 Steps 拆分到独立文件，按 ID 命名 | `dist/data/details/{id}.json` | 否 |
| `stats_collector` | 统计总数、常用厨具、难度分布等 | `dist/data/meta.json` | `fv stats` |
| `ai_exporter` | 生成 AI 专用的精简快照（每行 JSON） | `dist/data/ai-corpus.json` | 否 |

**执行顺序**：`validator` → `facet_indexer` → `tag_indexer` → `detail_splitter` → `stats_collector` → `ai_exporter`（可根据依赖调整）。

### 6.3 插件开发示例（`facet_indexer`）

参见后续“缓存集成”章节，插件需集成缓存管理器，支持增量构建和 TTL。

---

## 7. 缓存机制（`cache/manager.go`）

### 7.1 缓存存储结构

```
.flavor-vault/cache/
├── manifest.json                      # 全局缓存索引（记录各插件缓存状态）
├── facet_indexer/
│   ├── data.gob                       # 插件序列化数据（推荐 Gob）
│   └── meta.json                      # 元数据（生成时间、TTL、依赖哈希）
├── tag_indexer/
│   └── ...
└── ...
```

### 7.2 缓存管理接口

```go
type CacheManager struct { rootDir string }

func NewCacheManager(rootDir string) *CacheManager

// 检查缓存是否有效：依赖哈希一致 && 未超过 TTL
func (cm *CacheManager) IsValid(pluginName string, deps map[string]string, ttlSeconds int) bool

// 保存缓存数据（data 可为 []byte）
func (cm *CacheManager) Save(pluginName string, data []byte, deps map[string]string) error

// 加载缓存数据
func (cm *CacheManager) Load(pluginName string) ([]byte, error)

// 强制清除某个插件的缓存
func (cm *CacheManager) Clear(pluginName string) error
```

### 7.3 插件集成模式

每个插件在 `Build()` 中：
1. 计算当前依赖文件的哈希（菜谱文件 + config.yaml）。
2. 调用 `cm.IsValid()`，若有效则直接加载缓存数据，跳过重建。
3. 若无效或 `BuildContext.Force == true`，则执行完整构建逻辑。
4. 将结果序列化（JSON/Gob），调用 `cm.Save()` 写入缓存。

### 7.4 依赖哈希生成

- 对每个菜谱文件：计算 `md5` 或读取 `mtime`（`mtime` 更轻量但可能有误判，建议用 `md5`）。
- 对 `config.yaml`：同样计算 `md5`。
- 将结果存入 `deps` map 并与缓存中的记录比对。

---

## 8. ETL 构建流程（`fv build`）

1. **加载所有菜谱**：遍历 `.flavor-vault/recipes/*.json`，解析为 `[]models.Recipe`，同时记录文件路径与哈希。
2. **构造 BuildContext**：传入 Recipes、OutputDir、Force 等。
3. **顺序执行插件**（按预定义顺序）：
   - 每个插件检查缓存，若有效则跳过重建，直接复制缓存产物到 `dist/`。
   - 若失效则运行插件逻辑，输出到 `dist/`，并更新缓存。
4. **复制前端静态资源**：将 `web/dist/` 中的 `index.html` 和 `assets/` 复制到 `dist/`（或直接由前端构建输出到 `dist/` 根目录）。
5. **完成构建**：`dist/` 目录即为最终可部署的静态站点。

### 增量构建支持
- 利用 `manifest.json` 记录上次构建时每个文件的哈希。
- 本次构建时，仅将变化的菜谱传入插件（但插件若需全量索引，则仍需全量重建）。
- `facet_indexer` 等全量索引插件，可先判断变化数量，若变化极少（如 <5%），可考虑重建整个索引（因为 1000 道菜的重建耗时 < 50ms，无需增量优化）。

---

## 9. 前端集成（Vue 3 + Element Plus）

### 9.1 页面设计
- **首页**：展示筛选面板（厨具、标签、食材多选），下方显示符合条件的菜谱卡片列表（封面、名称、标签）。
- **详情页**：点击卡片进入，显示完整步骤、食材、视频、图片等。

### 9.2 数据加载策略
- **启动时**：`fetch('/data/meta.json')` 获取统计信息和所有菜谱 ID 列表（轻量）。
- **筛选时**：前端加载 `filters.json`（倒排索引），本地内存计算交集，得到 ID 列表。
- **列表渲染**：根据 ID 列表，从 `all.json`（或 `by-tag/` 分片）获取轻量菜谱信息（名称、标签、封面）。
- **详情**：`fetch('/data/details/{id}.json')` 懒加载完整数据。

### 9.3 状态管理（Pinia）
- `filterStore`：存储当前选中的厨具、标签、食材。
- `recipeStore`：存储当前列表数据，及加载状态。
- `computed` 自动响应筛选条件变化，触发内存过滤。

### 9.4 构建集成
- 前端项目通过 Vite 构建，输出到 `web/dist`。
- CLI 构建时，将 `web/dist` 内容复制到根 `dist`，并确保 `data/` 在根 `dist` 下（可配置符号链接或直接输出到 `web/dist/data`）。

---

## 10. AI 使用场景

### 10.1 本地 AI（Cursor/Claude Desktop）
- AI 通过 Shell 调用 `fv ask "..."`，CLI 快速从缓存加载索引并返回匹配结果（纯文本或 JSON）。
- 也可调用 `fv filter --...` 进行结构化筛选。

### 10.2 云端 AI（无法访问本地文件）
- 构建时生成 `dist/data/ai-corpus.json`（JSON Lines 格式），每条记录包含 `id, name, tags, main_ingredients, prep_time, cook_time`。
- AI 通过 System Prompt 获取该 URL，首次对话时 `fetch` 拉取到上下文，后续所有检索在 LLM 内存中完成。

### 10.3 智能问答示例
- 用户提问：“我想做一道不用炒锅的凉菜，有什么推荐？”
- 本地 AI：执行 `fv filter --厨具 蒸箱 --标签 凉菜`（或通过 `fv ask` 自然语言），得到 ID 列表，再 `fv show <id>` 获取详情。
- 云端 AI：从已加载的 `ai-corpus.json` 中过滤 `kitchenware` 不包含“炒锅”且 `tags` 包含“凉菜”的条目，返回名称和简介。

---

## 11. 部署方案（GitHub Pages）

### 11.1 手动部署
1. 用户运行 `fv build` 生成 `dist/`。
2. 将 `dist/` 推送到 GitHub 仓库的 `gh-pages` 分支。
3. 在仓库 Settings > Pages 中选择 `gh-pages` 分支作为源。

### 11.2 自动化部署（GitHub Actions）
创建 `.github/workflows/deploy.yml`：

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [ main ]
  workflow_dispatch:

jobs:
  build-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build CLI
        run: go build -o fv ./cmd/fv
      - name: Run ETL
        run: ./fv build --output=./dist --force
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'pnpm'
      - name: Build Vue
        run: |
          cd web
          pnpm install
          pnpm run build
          cp -r dist/* ../dist/   # 将前端产物合并到根 dist
      - name: Deploy to gh-pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./dist
          publish_branch: gh-pages
```

---

## 12. 非功能需求

### 12.1 性能要求
- `fv build` 在 100 道菜以内应 < 2s（含所有插件）。
- `fv filter` 查询响应 < 50ms。
- 前端首屏加载 < 1s（基于 CDN 缓存）。

### 12.2 可靠性
- 插件执行失败应输出清晰错误，并中断构建（`validator` 必须通过）。
- 缓存损坏时自动降级到全量重建。
- 菜谱文件格式错误时，应跳过该文件并给出警告，不中断整体构建（除非 `validator` 强制要求）。

### 12.3 扩展性
- 新增插件只需实现接口，并在 `main.go` 中注册，无需修改已有代码。
- 插件可声明对其他插件数据的依赖（通过 `BuildContext` 传递），但目前暂不实现 DAG，按固定顺序执行即可。

---

## 13. 开发与测试

### 13.1 开发环境
- Go 1.22+
- Node.js 18+ + pnpm
- Git

### 13.2 单元测试
- 为 `store`、`cache`、`pipeline` 编写单元测试，使用 `testing` 包和 `testdata/` 模拟菜谱。
- 前端使用 Vitest + Vue Test Utils。

### 13.3 集成测试
- 端到端测试：在临时目录运行 `fv init`，添加菜谱，执行 `fv build`，验证 `dist/` 输出内容。

---

## 14. 未来扩展建议
- **多用户协作**：当前基于 Git 共享，可考虑增加 Git 子模块或多仓库聚合。
- **支持导入其他格式**：如从网络爬取菜谱，通过插件实现导入器。
- **图片处理**：将图片压缩或转换为 WebP，减少前端加载体积（可作为插件）。
- **Recipe Schema 版本管理**：通过 `$schema` 字段支持 JSON Schema 校验。

---

## 15. 开发任务分解（供 AI IDE 使用）

1. **初始化项目结构**：创建目录，初始化 `go.mod`，安装 Cobra/Viper。
2. **实现数据模型与存储层**：`models/recipe.go`、`store/loader.go`。
3. **实现缓存管理器**：`cache/manager.go`（含单元测试）。
4. **实现插件框架**：`pipeline/plugin.go`、`scheduler.go`。
5. **实现 `facet_indexer` 插件**（含缓存集成、`fv filter` 命令）。
6. **实现其他插件**：`tag_indexer`、`detail_splitter`、`stats_collector`、`ai_exporter`、`validator`。
7. **实现 CLI 核心命令**：`add`、`edit`、`list`、`show`、`build`、`push`、`ask`。
8. **构建 Vue 前端**：设计页面、数据加载逻辑、筛选组件。
9. **配置 GitHub Actions**：自动化构建与部署。
10. **编写文档与示例**：README、使用教程。

---

## 16. 附录：关键代码片段参考

### 16.1 `facet_indexer` 核心逻辑（含缓存）
```go
func (p *FacetIndexer) Build(ctx *BuildContext) error {
    cm := cache.NewCacheManager(".flavor-vault/cache")
    deps := make(map[string]string)
    for _, r := range ctx.Recipes {
        hash, _ := cache.FileHash(r.FilePath)
        deps[r.FilePath] = hash
    }
    if !ctx.Force && cm.IsValid(p.Name(), deps, 86400) {
        data, _ := cm.Load(p.Name())
        return os.WriteFile(filepath.Join(ctx.OutputDir, "data/filters.json"), data, 0644)
    }
    // 构建索引...
    jsonData, _ := json.Marshal(index)
    cm.Save(p.Name(), jsonData, deps)
    return os.WriteFile(outPath, jsonData, 0644)
}
```

### 16.2 `fv filter` 命令
```go
// 在 RegisterCommands 中注册
filterCmd := &cobra.Command{
    Use:   "filter",
    RunE: func(cmd *cobra.Command, args []string) error {
        kitchenware, _ := cmd.Flags().GetString("厨具")
        tag, _ := cmd.Flags().GetString("标签")
        ingredient, _ := cmd.Flags().GetString("食材")
        // 从缓存或 dist 加载 filters.json
        index := loadFacetIndex()
        // 求交集...
        fmt.Println(strings.Join(result, "\n"))
        return nil
    },
}
```

### 16.3 前端筛选示例（Vue 组合式）
```ts
const filters = ref({ kitchenware: [], tags: [], ingredients: [] })
const filteredIds = computed(() => {
  // 从 filters.json 中根据选中条件求交集（使用 lodash/intersection 或手写）
})
```

---