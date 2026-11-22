#!/usr/bin/env bash
# 在 macOS 构建机上构建 BCompare Keygen：
#   1) 单文件/免安装版：BCompareKeygen.app（双击即运行），打包成 .zip
#   2) 安装版：.dmg（拖入 Applications 安装）
#
# 前置：Xcode / clang（Fyne 需 CGO）。Go 缺失时脚本自行下载到 ~/go-sdk。
# 手动组 .app 包，不依赖 fyne CLI。用法（在 mac-build 上）：bash scripts/build-macos.sh
#
# 作者/维护者：Cob
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"
DIST="$ROOT/dist/macos"
VERSION="$(sed -n 's/^Version = "\(.*\)"/\1/p' FyneApp.toml)"
APPNAME="BCompareKeygen"
APP="$DIST/$APPNAME.app"
mkdir -p "$DIST"

# --- 准备 Go ---
if ! command -v go >/dev/null 2>&1; then
  if [ ! -x "$HOME/go-sdk/go/bin/go" ]; then
    echo ">> 下载 Go（darwin-amd64）"
    mkdir -p "$HOME/go-sdk"
    curl -sSL -o /tmp/go.tgz https://go.dev/dl/go1.23.6.darwin-amd64.tar.gz
    tar -C "$HOME/go-sdk" -xzf /tmp/go.tgz
  fi
  export PATH="$HOME/go-sdk/go/bin:$PATH"
fi
# 有 vendor/ 则离线构建
if [ -d vendor ]; then export GOFLAGS=-mod=vendor; else export GOFLAGS=-mod=mod; fi
export GOPROXY=off
go version

echo ">> 编译 GUI 二进制（CGO）"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$DIST/$APPNAME-bin" ./cmd/bckeygen

echo ">> 组装 .app 包"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$DIST/$APPNAME-bin" "$APP/Contents/MacOS/$APPNAME"
chmod +x "$APP/Contents/MacOS/$APPNAME"

# 生成 icns
ICONSET="$(mktemp -d)/icon.iconset"
mkdir -p "$ICONSET"
for s in 16 32 64 128 256 512; do
  sips -z $s $s "$ROOT/assets/icon.png" --out "$ICONSET/icon_${s}x${s}.png" >/dev/null
  d=$((s*2)); sips -z $d $d "$ROOT/assets/icon.png" --out "$ICONSET/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/icon.icns"

cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>BCompare Keygen</string>
  <key>CFBundleDisplayName</key><string>BCompare Keygen</string>
  <key>CFBundleExecutable</key><string>$APPNAME</string>
  <key>CFBundleIdentifier</key><string>com.cob.bckeygen</string>
  <key>CFBundleVersion</key><string>$VERSION</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleIconFile</key><string>icon</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>LSMinimumSystemVersion</key><string>10.13</string>
</dict>
</plist>
EOF

echo ">> ad-hoc 签名"
codesign --force --deep --sign - "$APP" || echo "   (codesign 失败可忽略：首次运行右键打开)"

echo ">> 产出 .zip（单文件/免安装版）"
( cd "$DIST" && rm -f "$APPNAME-${VERSION}-macos.zip" \
    && ditto -c -k --keepParent "$APPNAME.app" "$APPNAME-${VERSION}-macos.zip" )

echo ">> 产出 .dmg（安装版）"
DMGTMP="$(mktemp -d)"
cp -R "$APP" "$DMGTMP/"
ln -s /Applications "$DMGTMP/Applications"
rm -f "$DIST/$APPNAME-${VERSION}-macos.dmg"
hdiutil create -volname "BCompare Keygen" -srcfolder "$DMGTMP" -ov -format UDZO \
    "$DIST/$APPNAME-${VERSION}-macos.dmg" >/dev/null
rm -rf "$DMGTMP"

echo ">> CLI 版"
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$DIST/$APPNAME-cli-${VERSION}-macos" ./cmd/bckeygen-cli

rm -f "$DIST/$APPNAME-bin"
echo ""
echo ">> 完成。产物："
ls -la "$DIST"
