#!/usr/bin/env bash
# 准备分享长图内置字体（DroidSansFallbackFull + DejaVuSans）到 internal/cli/fonts/
#
# 字体文件不提交 git（见 .gitignore），由本脚本在构建前准备：
#   1) 优先复制系统已有字体（最快、无网络；Ubuntu 可先 apt install fonts-droid-fallback fonts-dejavu-core）
#   2) 缺失的从网络下载（Droid 来自 Android 官方 repo，DejaVu 来自官方 release）
# CI（deploy.yml）已自动 apt 安装并运行本脚本。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIR="$ROOT/internal/cli/fonts"
mkdir -p "$DIR"

SYS_DROID=/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf
SYS_DEJAVU=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf

fetch() { # name url
  local name="$1" url="$2" dst="$DIR/$1"
  if [ -s "$dst" ]; then echo "✔ $name 已就绪"; return 0; fi
  echo "→ 下载 $name ..."
  curl -fsSL --retry 3 --connect-timeout 20 -o "$dst.tmp" "$url" || { rm -f "$dst.tmp"; echo "✘ 下载 $name 失败"; return 1; }
  mv "$dst.tmp" "$dst"
  echo "✔ $name -> $dst"
}

# 1) 系统字体复制优先
[ -f "$SYS_DROID" ] && cp "$SYS_DROID" "$DIR/DroidSansFallbackFull.ttf"
[ -f "$SYS_DEJAVU" ] && cp "$SYS_DEJAVU" "$DIR/DejaVuSans.ttf"

# 2) 缺失的从网络下载
[ -s "$DIR/DroidSansFallbackFull.ttf" ] || fetch DroidSansFallbackFull.ttf "https://raw.githubusercontent.com/android/platform_frameworks_base/master/data/fonts/DroidSansFallbackFull.ttf"
if [ ! -s "$DIR/DejaVuSans.ttf" ]; then
  echo "→ 下载 DejaVu 字体包（官方 release）..."
  curl -fsSL --retry 3 --connect-timeout 20 -o /tmp/dejavu-sans.zip "https://github.com/dejavu-fonts/dejavu-fonts/releases/download/version_2_37/dejavu-sans-ttf-2.37.zip"
  unzip -jo /tmp/dejavu-sans.zip "*/DejaVuSans.ttf" -d "$DIR"
  rm -f /tmp/dejavu-sans.zip
fi

# 3) 校验
if [ ! -s "$DIR/DroidSansFallbackFull.ttf" ] || [ ! -s "$DIR/DejaVuSans.ttf" ]; then
  echo "✘ 内置字体缺失，无法构建（请运行 apt install fonts-droid-fallback fonts-dejavu-core 或检查网络）" >&2
  exit 1
fi
echo "✔ 内置字体就绪：$DIR"
ls -la "$DIR"
