// Package patcher 负责「处理 RSA 密钥」这一步：把 Beyond Compare 可执行文件中
// 内置的 RSA 公钥（结尾 ...Np1+wk）就地改成与 keygen 匹配的版本（结尾 ...Npn+wk）。
//
// 这是对 README 中那条 sed 命令的跨平台封装：新旧公钥**长度相同**（仅一个字符
// 不同），因此是安全的等长就地字节替换，Windows / macOS / Linux 通用。
//
// 作者/维护者：Cob
package patcher

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// 完整公钥。NewKey 与 keygen 使用的公钥一致（结尾 Npn+wk）。
// OldKey 是 Beyond Compare 原始可执行文件中的版本（结尾 Np1+wk）。
const (
	NewKey = "++11Ik:7EFlNLs6Yqc3p-LtUOXBElimekQm8e3BTSeGhxhlpmVDeVVrrUAkLTXpZ7mK6jAPAOhyHiokPtYfmokklPELfOxt1s5HJmAnl-5r8YEvsQXY8-dm6EFwYJlXgWOCutNn2+FsvA7EXvM-2xZ1MW8LiGeYuXCA6Yt2wTuU4YWM+ZUBkIGEs1QRNRYIeGB9GB9YsS8U2-Z3uunZPgnA5pF+E8BRwYz9ZE--VFeKCPamspG7tdvjA3AJNRNrCVmJvwq5SqgEQwINdcmwwjmc4JetVK76og5A5sPOIXSwOjlYK+Sm8rvlJZoxh0XFfyioHz48JV3vXbBKjgAlPAc7Npn+wk"
	OldKey = "++11Ik:7EFlNLs6Yqc3p-LtUOXBElimekQm8e3BTSeGhxhlpmVDeVVrrUAkLTXpZ7mK6jAPAOhyHiokPtYfmokklPELfOxt1s5HJmAnl-5r8YEvsQXY8-dm6EFwYJlXgWOCutNn2+FsvA7EXvM-2xZ1MW8LiGeYuXCA6Yt2wTuU4YWM+ZUBkIGEs1QRNRYIeGB9GB9YsS8U2-Z3uunZPgnA5pF+E8BRwYz9ZE--VFeKCPamspG7tdvjA3AJNRNrCVmJvwq5SqgEQwINdcmwwjmc4JetVK76og5A5sPOIXSwOjlYK+Sm8rvlJZoxh0XFfyioHz48JV3vXbBKjgAlPAc7Np1+wk"
)

// State 描述目标文件当前的公钥状态。
type State int

const (
	StateNotFound       State = iota // 既找不到原始公钥也找不到已改公钥
	StateUnpatched                   // 含原始公钥，待修补
	StateAlreadyPatched              // 已是修补后的公钥
	StateMixed                       // 同时存在（异常，一般不会出现）
)

func (s State) String() string {
	switch s {
	case StateUnpatched:
		return "待修补（检测到原始公钥）"
	case StateAlreadyPatched:
		return "已修补"
	case StateMixed:
		return "混合状态（同时存在原始/已改公钥）"
	default:
		return "未找到公钥（可能不是 Beyond Compare 可执行文件）"
	}
}

// Status 返回目标文件的修补状态及原始/已改公钥各自的出现次数。
func Status(path string) (state State, oldCount, newCount int, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return StateNotFound, 0, 0, err
	}
	oldCount = bytes.Count(data, []byte(OldKey))
	newCount = bytes.Count(data, []byte(NewKey))
	switch {
	case oldCount > 0 && newCount > 0:
		return StateMixed, oldCount, newCount, nil
	case oldCount > 0:
		return StateUnpatched, oldCount, newCount, nil
	case newCount > 0:
		return StateAlreadyPatched, oldCount, newCount, nil
	default:
		return StateNotFound, 0, 0, nil
	}
}

// Patch 把目标文件中的原始公钥就地替换为已改公钥。
//   - backup=true 时先写一份 <path>.bak 备份（若已存在则不覆盖）。
//   - 返回替换的处数。已修补（0 处原始公钥但有已改公钥）时返回 0 且 err=nil。
//   - 写入采用「临时文件 + 原子改名」，避免写坏原文件。
//
// 注意：修改可执行文件会使其数字签名失效。macOS 上需随后重新 ad-hoc 签名
// （见 ResignMacho），否则 Gatekeeper 可能拒绝运行；Windows 签名失效但仍可运行。
func Patch(path string, backup bool) (replaced int, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	n := bytes.Count(data, []byte(OldKey))
	if n == 0 {
		if bytes.Contains(data, []byte(NewKey)) {
			return 0, nil // 已修补
		}
		return 0, fmt.Errorf("未在 %s 中找到 Beyond Compare 的内置公钥", filepath.Base(path))
	}
	if len(OldKey) != len(NewKey) {
		return 0, fmt.Errorf("内部错误：新旧公钥长度不一致")
	}

	patched := bytes.ReplaceAll(data, []byte(OldKey), []byte(NewKey))

	if backup {
		bak := path + ".bak"
		if _, e := os.Stat(bak); os.IsNotExist(e) {
			info, _ := os.Stat(path)
			mode := os.FileMode(0o644)
			if info != nil {
				mode = info.Mode()
			}
			if e := os.WriteFile(bak, data, mode); e != nil {
				return 0, fmt.Errorf("写备份失败：%w", e)
			}
		}
	}

	if err := atomicWrite(path, patched); err != nil {
		return 0, err
	}
	return n, nil
}

// atomicWrite 把内容写到同目录临时文件再改名覆盖，尽量保留原权限。
func atomicWrite(path string, data []byte) error {
	info, statErr := os.Stat(path)
	mode := os.FileMode(0o755)
	if statErr == nil {
		mode = info.Mode()
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".bcpatch-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// DefaultExecutablePaths 返回当前系统上 Beyond Compare 可执行文件的常见位置。
func DefaultExecutablePaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			`C:\Program Files\Beyond Compare 5\BCompare.exe`,
			`C:\Program Files (x86)\Beyond Compare 5\BCompare.exe`,
			`C:\Program Files\Beyond Compare 4\BCompare.exe`,
		}
	case "darwin":
		return []string{
			"/Applications/Beyond Compare.app/Contents/MacOS/BCompare",
		}
	default: // linux
		return []string{
			"/usr/lib/beyondcompare/BCompare",
			"/usr/local/lib/beyondcompare/BCompare",
			"/opt/beyondcompare/BCompare",
		}
	}
}

// FindExecutable 返回第一个实际存在的默认路径，找不到则返回空串。
func FindExecutable() string {
	for _, p := range DefaultExecutablePaths() {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}
