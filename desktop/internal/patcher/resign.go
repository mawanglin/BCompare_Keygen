package patcher

import (
	"fmt"
	"os/exec"
	"runtime"
)

// ResignMacho 在 macOS 上对补丁后的可执行文件做 ad-hoc 重签名，并去掉隔离属性，
// 否则修改字节后签名失效，Gatekeeper 会拒绝运行。非 macOS 平台为无操作。
//
// 需要系统自带的 codesign / xattr（随 Xcode Command Line Tools 提供）。
func ResignMacho(path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	// 去隔离（忽略失败：文件可能本就没有该属性）
	_ = exec.Command("xattr", "-d", "com.apple.quarantine", path).Run()
	// ad-hoc 重签名
	out, err := exec.Command("codesign", "--force", "--deep", "--sign", "-", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign 重签名失败：%v：%s", err, string(out))
	}
	return nil
}
