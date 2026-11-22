package keygen

import (
	"math/big"
	"testing"
)

// 与 Python 版 rsa_key.py 打印出的参数对齐。
func TestKeyParams(t *testing.T) {
	wantE := big.NewInt(50003)
	wantN, _ := new(big.Int).SetString("675cc0f3316cca097356bec356117b8eb44ff18a28d1afeb565d3db4b22b7ef1fc6458075237bc6acc1f86da6248d2c1195ab985bcf2fca8c519a68c7b6b107395efcd87d6773a1c3053156589e69eefb9b2e354851a9639cb950041a35df24f1f544040e56ecccc1945ebace1e2a0104b48d2e4d914a923e0371d650dc1449048980258f1fea01a8c224e442aa4ea33622297b90113d943c2508fb6100478ec8eeb96735c63b22621047c91e4281a72443be1c8c411f72ad3572333af790f81c5b417ad2bcb4c30ed306f9931ab7e4e9622f31bf8f5949cde03305d4faa1877db71d7287dea92b7322aa14d1bb2ab0781a8cd4305d415f97e08936a25047165", 16)
	if key.E.Cmp(wantE) != 0 {
		t.Fatalf("E 不匹配: got %s", key.E)
	}
	if key.N.Cmp(wantN) != 0 {
		t.Fatalf("N 不匹配:\n got %x", key.N)
	}
}

func TestRoundTrip(t *testing.T) {
	req := Request{Username: "Cob", Organization: "CobStudio", SerialNumber: "Abcd-1234", Quantity: 3}
	k, err := Encode(req)
	if err != nil {
		t.Fatal(err)
	}
	info, err := Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	if info.Username != "Cob" || info.Organization != "CobStudio" ||
		info.SerialNumber != "Abcd-1234" || info.Quantity != 3 || info.Version != "0x3d" {
		t.Fatalf("解析结果不匹配: %+v", info)
	}
}

func TestCheckSerial(t *testing.T) {
	ok := []string{"Abcd-1234", "AAAA-0000"}
	bad := []string{"abc-1234", "Abcd_1234", "Abcde-1234", ""}
	for _, s := range ok {
		if !CheckSerial(s) {
			t.Errorf("应通过: %s", s)
		}
	}
	for _, s := range bad {
		if CheckSerial(s) {
			t.Errorf("应拒绝: %s", s)
		}
	}
}
