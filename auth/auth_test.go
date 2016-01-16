// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

package auth

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
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

	// check form auth
	r := &http.Request{PostForm: url.Values{"token": {""}}}
	if _, err := CheckToken(ctx, r, false); err != ErrAnonymous {
		t.Error("invalid behavior")
	}

	r = &http.Request{PostForm: url.Values{"token": {"bad"}}}
	if _, err := CheckToken(ctx, r, false); err == nil || err == ErrAnonymous {
		t.Error("invalid behavior")
	}

	p1, p2, err := genToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r = &http.Request{PostForm: url.Values{"token": {p1 + p2}}}
	ctxToken, err := CheckToken(ctx, r, false)
	if err != nil {
		t.Errorf("invalid behavior: %v", err)
	}
	strToken, err := ExtractTokenKey(ctxToken)
	if err != nil {
		t.Errorf("invalid behavior: %v", err)
	}
	if p2 != strToken {
		t.Errorf("invalid behavior: %v != %v", p2, strToken)
	}

	// check API auth
	r = &http.Request{}
	if _, err := CheckToken(ctx, r, true); err != ErrAnonymous {
		t.Error("invalid behavior")
	}

	r = &http.Request{Header: http.Header{}}
	r.Header.Set("Authorization", "bad")
	if _, err := CheckToken(ctx, r, true); err == nil {
		t.Error("invalid behavior")
	}

	r = &http.Request{Header: http.Header{}}
	r.Header.Set("Authorization", "Bearer")
	if _, err := CheckToken(ctx, r, true); err == nil {
		t.Error("invalid behavior")
	}

	r = &http.Request{Header: http.Header{}}
	r.Header.Set("Authorization", "Bearer1234")
	if _, err := CheckToken(ctx, r, true); err == nil {
		t.Error("invalid behavior")
	}

	r = &http.Request{Header: http.Header{}}
	r.Header.Set("Authorization", "Bearer"+p1+p2)
	if _, err := CheckToken(ctx, r, true); err != nil {
		t.Errorf("invalid behavior: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	const userName = "test"
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

	s, err := db.NewSession(cfg.Conn, true)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx = db.NewContext(ctx, s)

	coll, err := db.Coll(s, "users")
	if err != nil {
		t.Fatal(err)
	}
	_, err = coll.RemoveAll(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = InitUsers(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := coll.Count(); err != nil || n != 2 {
		t.Errorf("n=%v, err=%v", n, err)
	}

	users, err := CreateUsers(ctx, []string{userName})
	if err != nil {
		t.Fatal(err)
	}
	if users, err := CreateUsers(ctx, []string{userName}); err == nil {
		if users[0].Err == "" {
			t.Error("invalid behavior")
		}
	}
	r := &http.Request{PostForm: url.Values{"token": {users[0].T}}}
	ctx, err = CheckToken(ctx, r, false)
	if err != nil {
		t.Errorf("invalid behavior: %v", err)
	}
	ctx, err = Authenticate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	u, err := ExtractUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != userName {
		t.Error("invalid behavior")
	}
	if !u.HasRole("user") {
		t.Error("invalid behavior")
	}
	if u.IsAnonymous() {
		t.Error("invalid behavior")
	}
	_, err = ChangeUsers(ctx, []string{userName})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := ChangeUsers(ctx, []string{"bad"}); err != nil {
		if result[0].Err == "" {
			t.Error("invalid behavior")
		}
	}
	_, err = DisableUsers(ctx, []string{userName})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := DisableUsers(ctx, []string{"bad"}); err != nil {
		if result[0].Err == "" {
			t.Error("invalid behavior")
		}
	}
}
