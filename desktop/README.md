# BCompare Keygen · 桌面版（Go + Fyne）

这是仓库根目录 Python 版的**桌面重写**：用 Go 重新实现完整流程，并用
[Fyne](https://fyne.io) 做纯桌面窗口界面，可编译成 **Windows / macOS / Linux**
三端的**单文件免安装程序**与**安装包**。

完整流程是**两步**，桌面版都已内置（两个标签页）：

1. **处理 RSA 公钥**：把 Beyond Compare 可执行文件里内置的 RSA 公钥（结尾 `...Np1+wk`）
   就地改成与注册机匹配的版本（结尾 `...Npn+wk`）。这是 README 里那条 `sed` 命令的
   跨平台封装——新旧公钥**长度相同**，是安全的等长就地替换，自动备份为 `.bak`。
2. **生成注册码**：填入用户名/组织/序列号/数量生成注册码并复制。

> 现有 Python 版（`app.py` / `keygen.py` 等）原样保留，二者功能等价、可共存。
> 仅供学习研究使用。维护者：Cob。

## 目录结构

```
desktop/
├── cmd/
│   ├── bckeygen/          # 桌面窗口版（Fyne GUI，两个标签页：修补 + 生成）
│   └── bckeygen-cli/      # 命令行版（对应 keygen.py，并支持修补）
├── internal/keygen/       # 注册码生成/解析核心（移植自 const.py / rsa_key.py / lic_manager.py）
├── internal/patcher/      # 公钥修补（跨平台等长字节替换 + macOS 自动重签名）
├── assets/icon.svg        # 应用图标（源文件；icon.png 由构建脚本栅格化）
├── scripts/               # 各平台打包脚本
└── FyneApp.toml           # 应用元数据（名称 / ID / 版本 / 图标）
```

## 核心兼容性

Go 核心与 Python 版**字节级兼容**：相同的许可数据结构、自定义 base64、按字重排、
RSA 私钥签名（`msg^D mod N`）与 base58 编码。单元测试校验了 RSA 参数 `E/N/D`，
并与 Python 的 `LicenseDecoder` 做了跨语言往返验证（Go 生成 → Python 解析一致）。

```bash
go test ./internal/keygen/
```

## 本地构建

### Linux

```bash
bash scripts/build-linux.sh
# 产物在 dist/linux/：
#   BCompareKeygen-<ver>-x86_64.AppImage    单文件免安装（双击即运行，跨发行版）
#   BCompareKeygen-<ver>-linux-amd64        单文件 GUI（裸二进制）
#   BCompareKeygen-cli-<ver>-linux-amd64    单文件 CLI
#   BCompareKeygen-<ver>-linux-amd64.deb    安装版（Debian/Ubuntu/飞牛）
#   BCompareKeygen-<ver>-linux-x86_64.rpm   安装版（Fedora/RHEL/openSUSE）
```

构建依赖：Go 1.23+、gcc、X11/GL 开发库（`xorg-dev libgl1-mesa-dev`）；
可选 `dpkg-deb`（.deb）、`rpmbuild`（.rpm）、`appimagetool`（AppImage，经
`APPIMAGETOOL=/path` 或 PATH 提供）、`rsvg-convert`（栅格化图标）。
AppImage 运行需系统有 FUSE。

### Windows

在 Linux 上用 mingw-w64 交叉编译：

```bash
bash scripts/build-windows.sh
# 产物在 dist/windows/：
#   BCompareKeygen.exe                    单文件版 GUI（免安装、双击即运行、含图标）
#   BCompareKeygen-cli.exe                单文件版 CLI
#   BCompareKeygen-Setup-<ver>.exe        安装版（NSIS，开始菜单/桌面快捷方式 + 卸载器）
```

依赖：`gcc-mingw-w64-x86-64`、`rsrc`、`makensis`（NSIS）、`rsvg-convert`+`imagemagick`（图标）。

### macOS

因 Fyne 依赖 Apple 框架的 CGO，需在 macOS 上原生构建（本仓库用 NAS 上的 mac 构建机）：

```bash
bash scripts/build-macos.sh   # 在 macOS 上执行
# 产物在 dist/macos/：
#   BCompareKeygen.app                    .app 包（双击即运行）
#   BCompareKeygen-<ver>-macos.zip        单文件/免安装版（.app 压缩）
#   BCompareKeygen-<ver>-macos.dmg        安装版（拖入 Applications）
#   BCompareKeygen-cli-<ver>-macos        单文件版 CLI
```

> 修改可执行文件会使签名失效：脚本已对产物做 ad-hoc 签名；用户首次运行若被
> Gatekeeper 拦截，右键「打开」一次即可。

## 命令行用法

```bash
# 生成 / 解析注册码
bckeygen-cli -u 用户名 -c 组织名 -s Abcd-1234 -n 1   # 生成
bckeygen-cli -d "$(cat key.txt)"                      # 解析已有注册码

# 修补 Beyond Compare 可执行文件（第一步）
bckeygen-cli -patch-status auto                       # 查看修补状态（auto=自动探测路径）
bckeygen-cli -patch auto                              # 修补（自动备份 .bak；macOS 自动重签名）
bckeygen-cli -patch /path/to/BCompare                 # 指定路径
```

## 兼容性验证

补丁已针对**官网 Beyond Compare 5.2.8.33052** 真实安装包、在三端真实环境逐一验证：

| 平台 | 可执行文件 | 原始公钥命中 | 结果 |
|---|---|---|---|
| Linux (.deb) | `usr/lib/beyondcompare/BCompare` | 1 处 | ✅ 替换成功，长度不变、幂等、备份一致 |
| macOS (.zip) | `Beyond Compare.app/Contents/MacOS/BCompare` | 2 处（通用二进制两切片） | ✅ 替换成功，长度不变 |
| Windows (.exe) | 安装后的 `BCompare.exe`（50.5 MB） | 1 处 | ✅ 在真实 Windows 上用交叉编译的 CLI 修补成功，长度不变、备份一致 |

> 公钥是全平台共用的常量，三端实测字符串完全一致；补丁为与平台无关的等长字节替换。
> Windows 安装包是 Inno Setup，可用 `/VERYSILENT /CURRENTUSER /DIR=…` 装到用户目录做无人值守验证。
