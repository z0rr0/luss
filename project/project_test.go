// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

package project

import (
	"testing"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/test"
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
	if _, err := checkToken("", cfg); err == nil {
		t.Error("invalid behavior")
	}
	if _, err := checkToken("a", cfg); err == nil {
		t.Error("invalid behavior")
	}
	if _, err := checkToken("abc", cfg); err == nil {
		t.Error("invalid behavior")
	}
	p1, p2, err := genToken(cfg)
	if err != nil {
		t.Error(err)
	}
	strToken, err := checkToken(p1+p2, cfg)
	if err != nil {
		t.Error(err)
	}
	if p2 != strToken {
		t.Errorf("invalid behavior: %v != %v", p2, strToken)
	}
}

func TestIsToken(t *testing.T) {
	cfg, err := conf.Parse(test.TcConfigName())
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	// values for tokenlen=20
	examples := map[string]bool{
		"":    false,
		"a":   false,
		"abc": false,
		"10f955da505cc4293e418c2a69b5e5c296a8e961743b6f4b9a320602df977687d61743b6f4b9a32J":  false,
		"10f955da505cc4293e418c2a69b5e5c296a8e961743b6f4b9a320602df977687d61743b6f4b9a320":  true,
		"10f955da505cc4293e418c2a69b5e5c296a8e961743b6f4b9a320602df977687d61743b6f4b9a3J0":  false,
		"J1f955da505cc4293e418c2a69b5e5c296a8e961743b6f4b9a320602df977687d61743b6f4b9a320":  false,
		"10J955da505cc4293e418c2a69b5e5c296a8e961743b6f4b9a320602df977687d61743b6f4b9a320":  false,
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef":  true,
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdefa": false,
	}
	for token, result := range examples {
		if IsToken(token, cfg) != result {
			t.Errorf("invalid result: %v", token)
		}
	}
}
