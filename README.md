# Flavor Vault · 菜谱数据仓库

本分支是独立的**菜谱数据仓库**，仅包含数据，不含程序代码：

- .flavor-vault/config.yaml — 数据侧配置（标签白名单、资源目录等）
- .flavor-vault/recipes/ — 菜谱源文件（每道菜一个 JSON）
- .flavor-vault/assets/ — 图片等资源

## 使用方式

- **独立使用**：克隆本分支即为一个完整数据源，配合 Flavor Vault CLI 读取：
  `git clone -b .flavor-vault/assets <repo-url> data && cd data && fv list`
- **fork / 私有化**：直接 fork 本分支，修改 config.yaml 与 recipes 即可，程序端零改动。
- **代码侧**：程序代码在默认分支（main），本分支只维护数据。

## 维护

    fv add                          # 新增菜谱（写入本仓库 recipes/）
    fv gh push --recipe <id>        # 把单个菜谱连同图片提交到本分支

> 当前含测试图片（封面/步骤图 SVG 占位，见 .flavor-vault/assets/images/）。
