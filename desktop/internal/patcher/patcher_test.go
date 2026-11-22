package patcher

import (
	"bytes"
	"os"
	"testing"
)

func TestKeysSameLength(t *testing.T) {
	if len(OldKey) != len(NewKey) {
		t.Fatalf("新旧公钥长度必须一致：old=%d new=%d", len(OldKey), len(NewKey))
	}
	if OldKey == NewKey {
		t.Fatal("新旧公钥不应相同")
	}
}

// 构造一个含原始公钥的假二进制，走完整 Status→Patch→再 Status 流程。
func TestPatchSynthetic(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/BCompare"
	content := append([]byte("\x00\x01random-head"), []byte(OldKey)...)
	content = append(content, []byte("tail-bytes\x00\xff")...)
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatal(err)
	}

	st, oc, nc, err := Status(path)
	if err != nil || st != StateUnpatched || oc != 1 || nc != 0 {
		t.Fatalf("修补前状态异常: st=%v oc=%d nc=%d err=%v", st, oc, nc, err)
	}

	n, err := Patch(path, true)
	if err != nil || n != 1 {
		t.Fatalf("修补失败: n=%d err=%v", n, err)
	}

	data, _ := os.ReadFile(path)
	if !bytes.Contains(data, []byte(NewKey)) || bytes.Contains(data, []byte(OldKey)) {
		t.Fatal("修补后内容不正确")
	}
	if len(data) != len(content) {
		t.Fatalf("修补后长度变化: %d -> %d", len(content), len(data))
	}

	// 备份应存在且等于原始内容
	bak, err := os.ReadFile(path + ".bak")
	if err != nil || !bytes.Equal(bak, content) {
		t.Fatal("备份内容不正确")
	}

	// 幂等：再次修补应为 0 处且已修补状态
	n2, err := Patch(path, true)
	if err != nil || n2 != 0 {
		t.Fatalf("重复修补应为 0: n=%d err=%v", n2, err)
	}
	st2, _, _, _ := Status(path)
	if st2 != StateAlreadyPatched {
		t.Fatalf("应为已修补: %v", st2)
	}
}

// 若设置了环境变量 BC_REAL_BIN，则对真实 Beyond Compare 可执行文件验证。
func TestPatchRealBinary(t *testing.T) {
	src := os.Getenv("BC_REAL_BIN")
	if src == "" {
		t.Skip("未设置 BC_REAL_BIN，跳过真实二进制测试")
	}
	orig, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := dir + "/BCompare"
	if err := os.WriteFile(path, orig, 0o755); err != nil {
		t.Fatal(err)
	}

	st, oc, _, _ := Status(path)
	t.Logf("真实二进制状态: %v，原始公钥出现 %d 处", st, oc)
	if st != StateUnpatched {
		t.Fatalf("真实二进制应为待修补状态，实际 %v", st)
	}

	n, err := Patch(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != oc {
		t.Fatalf("替换处数应等于检测处数: %d != %d", n, oc)
	}
	after, _ := os.ReadFile(path)
	if len(after) != len(orig) {
		t.Fatalf("长度变化: %d -> %d", len(orig), len(after))
	}
	if bytes.Count(after, []byte(NewKey)) != oc {
		t.Fatal("替换后已改公钥数量不符")
	}
	t.Logf("成功替换 %d 处，文件长度不变（%d 字节）", n, len(after))
}
