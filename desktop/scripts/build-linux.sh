#!/usr/bin/env bash
# 构建 Linux 版 BCompare Keygen：
#   1) 单文件版：免安装、双击即运行的独立可执行文件
#   2) 安装版：.deb 安装包（带菜单项与图标）
#
# 依赖：go、gcc、X11/GL 开发库；可选 dpkg-deb（打 .deb 用）、rsvg-convert（生成图标）。
# 用法：bash scripts/build-linux.sh
#
# 作者/维护者：Cob
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"
DIST="$ROOT/dist/linux"
VERSION="$(sed -n 's/^Version = "\(.*\)"/\1/p' FyneApp.toml)"
ARCH="amd64"
APPNAME="bckeygen"
PRETTY="BCompareKeygen"

mkdir -p "$DIST"

echo ">> [1/3] 生成图标 PNG"
if command -v rsvg-convert >/dev/null 2>&1; then
  rsvg-convert -w 512 -h 512 assets/icon.svg -o assets/icon.png
fi

echo ">> [2/3] 编译单文件版（CGO on）"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$DIST/${PRETTY}-${VERSION}-linux-${ARCH}" ./cmd/bckeygen
# CLI 版一并产出
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$DIST/${PRETTY}-cli-${VERSION}-linux-${ARCH}" ./cmd/bckeygen-cli
echo "   单文件版: $DIST/${PRETTY}-${VERSION}-linux-${ARCH}"

echo ">> [3/3] 打 .deb 安装包"
if ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "   [跳过] 未找到 dpkg-deb，仅产出单文件版"
  exit 0
fi

PKGROOT="$(mktemp -d)"
trap 'rm -rf "$PKGROOT"' EXIT
mkdir -p "$PKGROOT/DEBIAN" \
         "$PKGROOT/usr/bin" \
         "$PKGROOT/usr/share/applications" \
         "$PKGROOT/usr/share/icons/hicolor/512x512/apps" \
         "$PKGROOT/usr/share/doc/${APPNAME}"

install -m755 "$DIST/${PRETTY}-${VERSION}-linux-${ARCH}" "$PKGROOT/usr/bin/${APPNAME}"
cp assets/icon.png "$PKGROOT/usr/share/icons/hicolor/512x512/apps/${APPNAME}.png"

cat > "$PKGROOT/usr/share/applications/${APPNAME}.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=BCompare Keygen
Comment=Beyond Compare 5 注册码生成器
Exec=${APPNAME}
Icon=${APPNAME}
Terminal=false
Categories=Utility;Development;
EOF

cat > "$PKGROOT/DEBIAN/control" <<EOF
Package: ${APPNAME}
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: Cob <Cob@88.com>
Description: Beyond Compare 5 注册码生成器（桌面版）
 基于 Go + Fyne 的桌面小工具，为 Beyond Compare 5 生成注册密钥，仅供学习研究使用。
EOF

chmod -R u=rwX,go=rX "$PKGROOT"
DEB="$DIST/${PRETTY}-${VERSION}-linux-${ARCH}.deb"
dpkg-deb --build --root-owner-group "$PKGROOT" "$DEB" >/dev/null
echo "   安装版(deb): $DEB"

# ---- RPM 安装版（需 rpmbuild）----
if command -v rpmbuild >/dev/null 2>&1; then
  echo ">> [4/5] 打 .rpm 安装包"
  RB="$(mktemp -d)"
  mkdir -p "$RB/SOURCES" "$RB/SPECS"
  cp "$DIST/${PRETTY}-${VERSION}-linux-${ARCH}" "$RB/SOURCES/${APPNAME}"
  cp assets/icon.png "$RB/SOURCES/${APPNAME}.png"
  cp "$PKGROOT/usr/share/applications/${APPNAME}.desktop" "$RB/SOURCES/${APPNAME}.desktop"
  cat > "$RB/SPECS/${APPNAME}.spec" <<SPEC
%global debug_package %{nil}
%global __strip /bin/true
Name:    ${APPNAME}
Version: ${VERSION}
Release: 1
Summary: Beyond Compare 5 注册码生成器（桌面版）
License: MIT
URL:     https://github.com/mawanglin/BCompare_Keygen
BuildArch: ${ARCH}
%description
基于 Go + Fyne 的桌面小工具，为 Beyond Compare 5 生成注册密钥并修补内置公钥。Maintainer: Cob.
%install
install -Dm755 %{_sourcedir}/${APPNAME} %{buildroot}%{_bindir}/${APPNAME}
install -Dm644 %{_sourcedir}/${APPNAME}.desktop %{buildroot}%{_datadir}/applications/${APPNAME}.desktop
install -Dm644 %{_sourcedir}/${APPNAME}.png %{buildroot}%{_datadir}/icons/hicolor/512x512/apps/${APPNAME}.png
%files
%{_bindir}/${APPNAME}
%{_datadir}/applications/${APPNAME}.desktop
%{_datadir}/icons/hicolor/512x512/apps/${APPNAME}.png
SPEC
  rpmbuild --define "_topdir $RB" -bb "$RB/SPECS/${APPNAME}.spec" >/dev/null 2>&1
  cp "$RB"/RPMS/*/*.rpm "$DIST/${PRETTY}-${VERSION}-linux-${ARCH2:-x86_64}.rpm"
  rm -rf "$RB"
  echo "   安装版(rpm): $DIST/${PRETTY}-${VERSION}-linux-${ARCH2:-x86_64}.rpm"
else
  echo "   [跳过] 未找到 rpmbuild，不打 .rpm"
fi

# ---- AppImage 单文件免安装 ----
APPIMAGETOOL="${APPIMAGETOOL:-$(command -v appimagetool || true)}"
if [ -n "$APPIMAGETOOL" ]; then
  echo ">> [5/5] 打 AppImage"
  AD="$(mktemp -d)/AppDir"
  mkdir -p "$AD/usr/bin" "$AD/usr/share/applications" "$AD/usr/share/icons/hicolor/512x512/apps"
  cp "$DIST/${PRETTY}-${VERSION}-linux-${ARCH}" "$AD/usr/bin/${APPNAME}"
  cp assets/icon.png "$AD/usr/share/icons/hicolor/512x512/apps/${APPNAME}.png"
  cp assets/icon.png "$AD/${APPNAME}.png"; cp assets/icon.png "$AD/.DirIcon"
  cp "$PKGROOT/usr/share/applications/${APPNAME}.desktop" "$AD/${APPNAME}.desktop"
  cp "$AD/${APPNAME}.desktop" "$AD/usr/share/applications/${APPNAME}.desktop"
  printf '#!/bin/sh\nHERE="$(dirname "$(readlink -f "$0")")"\nexec "$HERE/usr/bin/%s" "$@"\n' "$APPNAME" > "$AD/AppRun"
  chmod +x "$AD/AppRun"
  ( cd "$DIST" && ARCH=x86_64 "$APPIMAGETOOL" --appimage-extract-and-run "$AD" "${PRETTY}-${VERSION}-x86_64.AppImage" >/dev/null 2>&1 )
  rm -rf "$(dirname "$AD")"
  echo "   单文件(AppImage): $DIST/${PRETTY}-${VERSION}-x86_64.AppImage"
else
  echo "   [跳过] 未找到 appimagetool，不打 AppImage（设 APPIMAGETOOL=/path 指定）"
fi

echo ""
echo ">> 完成。产物："
ls -la "$DIST"
