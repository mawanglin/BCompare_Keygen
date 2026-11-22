// Package keygen 实现 Beyond Compare 5 注册码的生成与解析。
//
// 这是对仓库根目录 Python 版本（const.py / rsa_key.py / lic_manager.py）核心逻辑
// 的 Go 移植，行为与之保持一致：同样的许可数据结构、同样的自定义 base64、
// 按字重排、RSA 私钥签名（msg^D mod N）以及 base58 编码。
//
// 作者/维护者：Cob
package keygen

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// 与 const.py 保持一致的常量。
const (
	standardAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	customAlphabet   = "+-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	publicKey = "++11Ik:7EFlNLs6Yqc3p-LtUOXBElimekQm8e3BTSeGhxhlpmVDeVVrrUAkLTXpZ7mK6jAPAOhyHiokPtYfmokklPELfOxt1s5HJmAnl-5r8YEvsQXY8-dm6EFwYJlXgWOCutNn2+FsvA7EXvM-2xZ1MW8LiGeYuXCA6Yt2wTuU4YWM+ZUBkIGEs1QRNRYIeGB9GB9YsS8U2-Z3uunZPgnA5pF+E8BRwYz9ZE--VFeKCPamspG7tdvjA3AJNRNrCVmJvwq5SqgEQwINdcmwwjmc4JetVK76og5A5sPOIXSwOjlYK+Sm8rvlJZoxh0XFfyioHz48JV3vXbBKjgAlPAc7Npn+wk"

	hexD = "4860d32b474ff398b0058aaf111fe820f8bebad4342cb40b6fd7652b37a92cf077d58ca7374dcf65615fe846e73ababe6a729a59ebdd8b980bbeb47f3ef8041decc465118a40d76293b5fce1271d87865b3f1dc116f2637d8dfa338a5103ef14e9c28f620c325c1e241e2bfa9258d16b1239c5c06ce13ec2fe377fac038a0ff0eb0f5910018724fd4bf429f1c0fac86af083acdab388c18e281a5ea9976b385e6c0383485135f1e68cd7a3c0ab6d36b07aa1404e081083158e523129ace077972fc3bd9424fbe86c64b33e8916e0a15c0f5a346e2260fb565ee00741268e6987b978df646c81bd72b55e0ea94f5f51956bf80ffc4c51f6fcaaab96135c888523"

	licTypeAll = 0x3D // WINDOWS(4)|LINUX(8)|MACOS(0x10)|PRO(0x21)

	keyHeader = "--- BEGIN LICENSE KEY ---\r\n"
	keyFooter = "\r\n--- END LICENSE KEY -----\r\n"
)

const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var serialPattern = regexp.MustCompile(`^[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}$`)

// rsaKey 持有解析出的 RSA 参数。
type rsaKey struct {
	E, D, N *big.Int
}

var key = mustParseKey()

// CheckSerial 校验序列号格式（形如 Abcd-1234）。
func CheckSerial(serial string) bool {
	return serialPattern.MatchString(serial)
}

// Request 是一次注册码生成请求的输入参数。
type Request struct {
	Username     string
	Organization string
	SerialNumber string
	Quantity     int
}

// Info 是从注册码中解析出的信息。
type Info struct {
	Version      string
	Username     string
	Organization string
	SerialNumber string
	Quantity     int
	Random       string
}

// ---- 自定义 base64（与 rsa_key.py 一致）----

func translate(in []byte, from, to string) []byte {
	out := make([]byte, len(in))
	for i, c := range in {
		idx := strings.IndexByte(from, c)
		if idx >= 0 {
			out[i] = to[idx]
		} else {
			out[i] = c
		}
	}
	return out
}

func b64decodeExt(in []byte) ([]byte, error) {
	pad := len(in) % 4
	if pad != 0 {
		in = append(append([]byte{}, in...), bytes.Repeat([]byte("="), pad)...)
	}
	t := translate(in, customAlphabet, standardAlphabet)
	return base64.StdEncoding.DecodeString(string(t))
}

// reverseByWord 把每 4 字节一组做翻转。
func reverseByWord(data []byte) []byte {
	ret := make([]byte, 0, len(data))
	for i := 0; i < len(data); i += 4 {
		end := i + 4
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]
		for j := len(chunk) - 1; j >= 0; j-- {
			ret = append(ret, chunk[j])
		}
	}
	return ret
}

func leBytesToBig(data []byte) *big.Int {
	// little-endian -> big.Int
	be := make([]byte, len(data))
	for i := range data {
		be[len(data)-1-i] = data[i]
	}
	return new(big.Int).SetBytes(be)
}

func bigToLEBytes(n *big.Int) []byte {
	be := n.Bytes() // big-endian, minimal
	le := make([]byte, len(be))
	for i := range be {
		le[len(be)-1-i] = be[i]
	}
	return le
}

func mustParseKey() rsaKey {
	k, err := parseKey()
	if err != nil {
		panic(err)
	}
	return k
}

func parseKey() (rsaKey, error) {
	parts := strings.SplitN(publicKey, ":", 2)
	if len(parts) != 2 {
		return rsaKey{}, errors.New("公钥格式错误")
	}
	bsE, err := b64decodeExt([]byte(parts[0]))
	if err != nil {
		return rsaKey{}, err
	}
	bsN, err := b64decodeExt([]byte(parts[1]))
	if err != nil {
		return rsaKey{}, err
	}
	d, ok := new(big.Int).SetString(hexD, 16)
	if !ok {
		return rsaKey{}, errors.New("私钥指数解析失败")
	}
	return rsaKey{
		E: leBytesToBig(reverseByWord(bsE)),
		N: leBytesToBig(reverseByWord(bsN)),
		D: d,
	}, nil
}

// ---- base58（比特币字母表，与 python base58.b58encode 一致）----

func base58Encode(input []byte) string {
	zeros := 0
	for zeros < len(input) && input[zeros] == 0 {
		zeros++
	}
	x := new(big.Int).SetBytes(input)
	radix := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)
	var out []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, radix, mod)
		out = append(out, b58Alphabet[mod.Int64()])
	}
	for i := 0; i < zeros; i++ {
		out = append(out, b58Alphabet[0])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func base58Decode(s string) ([]byte, error) {
	x := big.NewInt(0)
	radix := big.NewInt(58)
	for i := 0; i < len(s); i++ {
		idx := strings.IndexByte(b58Alphabet, s[i])
		if idx < 0 {
			return nil, fmt.Errorf("非法的 base58 字符: %q", s[i])
		}
		x.Mul(x, radix)
		x.Add(x, big.NewInt(int64(idx)))
	}
	be := x.Bytes()
	zeros := 0
	for zeros < len(s) && s[zeros] == b58Alphabet[0] {
		zeros++
	}
	return append(make([]byte, zeros), be...), nil
}

// ---- 许可数据构造（与 lic_manager.py 一致）----

func genPaddingLic(data []byte) []byte {
	if len(data) == 0 {
		return []byte{0}
	}
	return append([]byte{byte(len(data))}, data...)
}

func pad255(data []byte) []byte {
	const bs = 255
	padLen := bs - len(data)%bs
	return append(data, bytes.Repeat([]byte{byte(padLen)}, padLen)...)
}

func genLic(req Request) ([]byte, error) {
	random5 := make([]byte, 5)
	if _, err := rand.Read(random5); err != nil {
		return nil, err
	}

	var lic []byte
	lic = append(lic, 0x04)
	lic = append(lic, []byte("SCTR")...)
	for i := 0; i < 5; i++ {
		lic = append(lic, genPaddingLic(nil)...)
	}
	lic = append(lic, 0x01)
	lic = append(lic, genPaddingLic([]byte("73051"))...)
	lic = append(lic, genPaddingLic([]byte(fmt.Sprintf("%d|%s", req.Quantity, req.Organization)))...)
	lic = append(lic, 0x06)
	lic = append(lic, byte(licTypeAll))
	lic = append(lic, random5...)
	lic = append(lic, 0x09)
	lic = append(lic, []byte(req.SerialNumber)...)
	lic = append(lic, genPaddingLic([]byte("0"))...)
	lic = append(lic, genPaddingLic([]byte("30"))...)
	lic = append(lic, genPaddingLic([]byte("15"))...)
	lic = append(lic, genPaddingLic([]byte(req.Username))...)
	lic = append(lic, genPaddingLic([]byte("0"))...)
	lic = append(lic, genPaddingLic([]byte("0"))...)
	return pad255(lic), nil
}

// Encode 生成注册码。序列号非法时回退为 Abcd-Efgh（与 Python 行为一致）。
func Encode(req Request) (string, error) {
	if !CheckSerial(req.SerialNumber) {
		req.SerialNumber = "Abcd-Efgh"
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	lic, err := genLic(req)
	if err != nil {
		return "", err
	}
	licData := leBytesToBig(lic)
	enc := new(big.Int).Exp(licData, key.D, key.N)
	data := bigToLEBytes(enc)
	return keyHeader + base58Encode(data) + keyFooter, nil
}

// Decode 解析注册码，取回其中的用户名/组织/序列号等信息。
func Decode(licKey string) (Info, error) {
	body := licKey
	body = strings.ReplaceAll(body, keyHeader, "")
	body = strings.ReplaceAll(body, keyFooter, "")
	body = strings.TrimSpace(body)

	encBytes, err := base58Decode(body)
	if err != nil {
		return Info{}, err
	}
	encData := leBytesToBig(encBytes)
	raw := new(big.Int).Exp(encData, key.E, key.N)
	data := bigToLEBytes(raw)
	return parseLic(data)
}

func parseLic(data []byte) (Info, error) {
	var info Info

	// 组织信息：\x05 "73051" 之后 1 字节长度，再到 \x06
	preOrg := []byte("\x0573051")
	i := bytes.Index(data, preOrg)
	if i < 0 {
		return info, errors.New("解析失败：找不到组织段")
	}
	cur := data[i+len(preOrg)+1:]
	j := bytes.IndexByte(cur, 0x06)
	if j < 0 {
		return info, errors.New("解析失败：组织段结尾缺失")
	}
	orgField := strings.TrimSpace(string(cur[:j]))
	segs := strings.SplitN(orgField, "|", 2)
	if len(segs) != 2 {
		return info, errors.New("解析失败：组织字段格式错误")
	}
	var num int
	if _, err := fmt.Sscanf(segs[0], "%d", &num); err != nil {
		return info, err
	}
	info.Quantity = num
	info.Organization = segs[1]
	cur = cur[j:]

	// 版本：\x06 之后 1 字节
	k := bytes.IndexByte(cur, 0x06)
	if k < 0 || k+1 >= len(cur) {
		return info, errors.New("解析失败：版本段缺失")
	}
	cur = cur[k+1:]
	info.Version = fmt.Sprintf("0x%x", cur[0])

	// 随机数 + 序列号：\x09 ... \x01 0 \x02 30 \x02 15
	r := bytes.IndexByte(cur, 0x09)
	if r < 0 {
		return info, errors.New("解析失败：随机段缺失")
	}
	info.Random = fmt.Sprintf("0x%x", new(big.Int).SetBytes(cur[1:r]))
	cur = cur[r:]
	postRand := []byte("\x010\x0230\x0215")
	p := bytes.Index(cur, postRand)
	if p < 0 {
		return info, errors.New("解析失败：序列号段结尾缺失")
	}
	serialParts := strings.SplitN(string(cur[1:p]), "-", 3)
	if len(serialParts) < 2 {
		return info, errors.New("解析失败：序列号格式错误")
	}
	info.SerialNumber = serialParts[0] + "-" + serialParts[1]

	// 用户名：\x01 0 \x02 30 \x02 15 之后 1 字节长度，再到 \x01 0 \x01 0
	preUname := []byte("\x01\x30\x0230\x0215")
	u := bytes.Index(cur, preUname)
	if u < 0 {
		return info, errors.New("解析失败：用户名段缺失")
	}
	cur = cur[u+len(preUname)+1:]
	postUname := []byte("\x010\x010")
	q := bytes.Index(cur, postUname)
	if q < 0 {
		return info, errors.New("解析失败：用户名段结尾缺失")
	}
	info.Username = string(cur[:q])

	return info, nil
}
