# 内置字体（构建时下载，不提交仓库）

`fv share --img` 渲染分享长图需要字体：优先用系统字体，系统缺失时用内置字体兜底
（保证任何环境都能渲染、不出方块字）。

本目录的 `.ttf` **不提交到 git**（见 `.gitignore`），由
[`scripts/fetch-fonts.sh`](../../scripts/fetch-fonts.sh) 在构建前准备：
1. 优先复制系统已有字体（Ubuntu/Debian 可先 `apt install fonts-droid-fallback fonts-dejavu-core`）
2. 缺失的从网络下载（Droid 来自 Android 官方 repo、DejaVu 来自官方 release）

| 文件 | 用途 | 来源 | License |
|---|---|---|---|
| `DroidSansFallbackFull.ttf` | 全量中文（CJK） | [android/platform_frameworks_base](https://raw.githubusercontent.com/android/platform_frameworks_base/master/data/fonts/DroidSansFallbackFull.ttf) | Apache License 2.0 |
| `DejaVuSans.ttf` | ASCII/数字/符号（含 `★·`） | [dejavu-fonts release 2.37](https://github.com/dejavu-fonts/dejavu-fonts/releases/tag/version_2_37) | Bitstream Vera + 公有领域 |

本地构建：`bash scripts/fetch-fonts.sh`
CI：`deploy.yml` 已自动 `apt` 安装并运行本脚本。
