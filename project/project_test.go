// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

package project

import (
	"testing"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/test"
	"golang.org/x/net/context"
)

func TestEqualBytes(t *testing.T) {
	s := "test string-試験の文字列-тестовая строка"
	sb := []byte{
		116, 101, 115, 116, 32, 115, 116, 114, 105, 110, 103, 45, 232, 169,
		166, 233, 168, 147, 227, 129, 174, 230, 150, 135, 229, 173, 151, 229,
		136, 151, 45, 209, 130, 208, 181, 209, 129, 209, 130, 208, 190, 208, 178,
		208, 176, 209, 143, 32, 209, 129, 209, 130, 209, 128, 208, 190, 208, 186, 208, 176}
	if !EqualBytes([]byte(s), sb) {
		t.Error("invalid behavior")
	}
	n := len(sb)
	if EqualBytes([]byte(s), sb[:n-1]) {
		t.Error("invalid behavior")
	}
	sb[n-1] = 177
	if EqualBytes([]byte(s), sb) {
		t.Error("invalid behavior")
	}
}

func TestToken(t *testing.T) {
	cfg, err := conf.Parse(test.TcConfigName())
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	ctx, cancel := context.WithCancel(conf.NewContext(cfg))
	defer cancel()

	if _, err := CheckToken(ctx, ""); err != ErrAnonymous {
		t.Error("invalid behavior")
	}
	if _, err := CheckToken(ctx, "a"); err == nil {
		t.Error("invalid behavior")
	}
	if _, err := CheckToken(ctx, "abc"); err == nil {
		t.Error("invalid behavior")
	}
	p1, p2, err := genToken(cfg)
	if err != nil {
		t.Error(err)
	}
	ctx, err = CheckToken(ctx, p1+p2)
	if err != nil {
		t.Error(err)
	}
	strToken, err := ExtractTokenKey(ctx)
	if err != nil {
		t.Error(err)
	}
	if p2 != strToken {
		t.Errorf("invalid behavior: %v != %v", p2, strToken)
	}
}
