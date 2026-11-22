#!/usr/bin/env bash
# 在 Linux 上交叉编译 Windows 版 BCompare Keygen：
#   1) 单文件版：BCompareKeygen.exe（免安装、双击即运行，GUI、无控制台、含图标）
#   2) 安装版：BCompareKeygen-Setup-<ver>.exe（NSIS，开始菜单/桌面快捷方式 + 卸载器）
#
# 依赖：go、mingw-w64（x86_64-w64-mingw32-gcc）、rsrc、makensis、rsvg-convert+imagemagick（生成图标）。
# 用法：bash scripts/build-windows.sh
#
# 作者/维护者：Cob
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"
DIST="$ROOT/dist/windows"
VERSION="$(sed -n 's/^Version = "\(.*\)"/\1/p' FyneApp.toml)"
export GOFLAGS=-mod=mod
export PATH="$PATH:$HOME/go/bin"
mkdir -p "$DIST"

echo ">> [1/4] 生成图标 .ico"
if command -v rsvg-convert >/dev/null 2>&1 && command -v convert >/dev/null 2>&1; then
  tmp=$(mktemp -d)
  for s in 16 32 64 256; do rsvg-convert -w $s -h $s assets/icon.svg -o "$tmp/i$s.png"; done
  convert "$tmp/i256.png" "$tmp/i64.png" "$tmp/i32.png" "$tmp/i16.png" assets/icon.ico
  rm -rf "$tmp"
fi

echo ">> [2/4] 生成图标资源 .syso"
if ! command -v rsrc >/dev/null 2>&1; then go install github.com/akavel/rsrc@latest; fi
rsrc -ico assets/icon.ico -arch amd64 -o cmd/bckeygen/rsrc_windows_amd64.syso

echo ">> [3/4] 交叉编译 .exe"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  go build -trimpath -ldflags "-H windowsgui -s -w" -o "$DIST/BCompareKeygen.exe" ./cmd/bckeygen
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -trimpath -ldflags "-s -w" -o "$DIST/BCompareKeygen-cli.exe" ./cmd/bckeygen-cli
echo "   单文件版: $DIST/BCompareKeygen.exe"

echo ">> [4/4] 打 NSIS 安装包"
if command -v makensis >/dev/null 2>&1; then
  makensis -V2 \
    -DAPPVER="$VERSION" \
    -DSRCEXE="$DIST/BCompareKeygen.exe" \
    -DSRCCLI="$DIST/BCompareKeygen-cli.exe" \
    -DICON="$ROOT/assets/icon.ico" \
    -DOUTFILE="$DIST/BCompareKeygen-Setup-$VERSION.exe" \
    scripts/windows-installer.nsi
  echo "   安装版:   $DIST/BCompareKeygen-Setup-$VERSION.exe"
else
  echo "   [跳过] 未找到 makensis"
fi

echo ""
echo ">> 完成。产物："
ls -la "$DIST"
