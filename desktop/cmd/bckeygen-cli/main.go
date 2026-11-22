// bckeygen-cli 是命令行版 Beyond Compare 5 注册机，对应仓库中的 keygen.py，
// 并额外提供「修补公钥」功能（对应 README 里那条 sed 命令，跨平台）。
//
// 作者/维护者：Cob
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/mawanglin/BCompare_Keygen/desktop/internal/keygen"
	"github.com/mawanglin/BCompare_Keygen/desktop/internal/patcher"
)

func main() {
	user := flag.String("u", "Test", "用户名 Username")
	company := flag.String("c", "Home", "组织名 Company")
	serial := flag.String("s", "Abcd-Efgh", "序列号 Serial number (xxxx-xxxx)")
	num := flag.Int("n", 1, "最大用户数 Max user number")
	showVer := flag.Bool("v", false, "显示版本 Show version")
	decode := flag.String("d", "", "解码一个已有注册码（整段文本）")

	patchPath := flag.String("patch", "", "修补指定的 Beyond Compare 可执行文件（传 auto 自动探测）")
	statusPath := flag.String("patch-status", "", "查看指定可执行文件的修补状态（传 auto 自动探测）")
	noBackup := flag.Bool("no-backup", false, "修补时不生成 .bak 备份")
	flag.Parse()

	switch {
	case *showVer:
		fmt.Println("BCompare_Keygen (Go) 1.1 — by Cob")
	case *statusPath != "":
		doStatus(resolvePath(*statusPath))
	case *patchPath != "":
		doPatch(resolvePath(*patchPath), !*noBackup)
	case *decode != "":
		doDecode(*decode)
	default:
		doEncode(*user, *company, *serial, *num)
	}
}

func resolvePath(p string) string {
	if p == "auto" || p == "" {
		if found := patcher.FindExecutable(); found != "" {
			return found
		}
		fmt.Fprintln(os.Stderr, "未能自动探测到 Beyond Compare 可执行文件，请显式指定路径。")
		os.Exit(1)
	}
	return p
}

func doStatus(path string) {
	st, oc, nc, err := patcher.Status(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取失败:", err)
		os.Exit(1)
	}
	fmt.Printf("文件: %s\n状态: %s\n原始公钥: %d 处，已改公钥: %d 处\n", path, st, oc, nc)
}

func doPatch(path string, backup bool) {
	n, err := patcher.Patch(path, backup)
	if err != nil {
		fmt.Fprintln(os.Stderr, "修补失败:", err)
		os.Exit(1)
	}
	if n == 0 {
		fmt.Println("该文件已是修补后的公钥，无需修补。")
		return
	}
	fmt.Printf("修补成功：替换 %d 处公钥。", n)
	if backup {
		fmt.Printf(" 备份: %s.bak", path)
	}
	fmt.Println()
	if runtime.GOOS == "darwin" {
		if err := patcher.ResignMacho(path); err != nil {
			fmt.Fprintln(os.Stderr, "⚠️ 自动重签名失败:", err)
			fmt.Fprintf(os.Stderr, "请手动执行: codesign -f -s - %q\n", path)
		} else {
			fmt.Println("已自动 ad-hoc 重签名。")
		}
	}
}

func doDecode(key string) {
	info, err := keygen.Decode(key)
	if err != nil {
		fmt.Fprintln(os.Stderr, "解码失败:", err)
		os.Exit(1)
	}
	fmt.Printf("Version: %s\nSerial: %s\nUsername: %s\nCompany: %s\nMax users: %d\nRandom: %s\n",
		info.Version, info.SerialNumber, info.Username, info.Organization, info.Quantity, info.Random)
}

func doEncode(user, company, serial string, num int) {
	s := serial
	if !keygen.CheckSerial(s) {
		fmt.Fprintf(os.Stderr, "Serial num [%s] invalid, use [Abcd-Efgh] as default.\n", s)
		s = "Abcd-Efgh"
	}
	key, err := keygen.Encode(keygen.Request{
		Username:     user,
		Organization: company,
		SerialNumber: s,
		Quantity:     num,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "生成失败:", err)
		os.Exit(1)
	}
	fmt.Print(key)
}
