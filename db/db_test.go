// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package db implements MongoDB database access methods.
package db

import (
	"testing"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/test"
)

func TestNewSession(t *testing.T) {
	cfg, err := conf.Parse(test.TcConfigName())
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	s, err := NewSession(cfg.Conn, true)
	if err != nil {
		t.Fatalf("invalid db init")
	}
	defer s.Close()
	ctx := conf.NewContext(cfg)
	if _, err := CtxSession(ctx); err == nil {
		t.Fatalf("invalid behavior")
	}
	if _, err := C(ctx, "tests"); err == nil {
		t.Fatalf("invalid behavior")
	}
	ctx = NewContext(ctx, s)
	if _, err := CtxSession(ctx); err != nil {
		t.Error(err)
	}
	if _, err := C(ctx, "test-bad"); err == nil {
		t.Fatalf("invalid behavior")
	}
	coll, err := C(ctx, "tests")
	if err != nil {
		t.Error(err)
	}
	_, err = coll.Count()
	if err != nil {
		t.Error(err)
	}
}

func TestCheckID(t *testing.T) {
	suite := map[string]bool{
		"5639dc619c6acd2c8362eba5": true,
		"": false,
		"aaaaaaaaaaaaaaaaaaaaaaaz":  false,
		"aaaaaaaaaaaaaaaaaaaaaaaaa": false,
		"aa": false,
	}
	for k, v := range suite {
		if c, err := CheckID(k); (err != nil && v) || (err == nil && !v) {
			t.Errorf("invalid: %v, %v, %v", c, v, err)
		}
	}
}

func BenchmarkSession(b *testing.B) {
	cfg, err := conf.Parse(test.TcConfigName())
	if err != nil {
		b.Fatal("invalid behavior")
	}
	err = cfg.Validate()
	if err != nil {
		b.Fatal("invalid behavior")
	}
	s, err := NewSession(cfg.Conn, true)
	if err != nil {
		b.Fatal(err)
	}
	s.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, err := NewSession(cfg.Conn, true)
		if err != nil {
			b.Error(err)
		}
		s.Close()
	}
}

func BenchmarkContext(b *testing.B) {
	cfg, err := conf.Parse(test.TcConfigName())
	if err != nil {
		b.Fatal("invalid behavior")
	}
	err = cfg.Validate()
	if err != nil {
		b.Fatal("invalid behavior")
	}
	ctx := conf.NewContext(cfg)
	s, err := NewSession(cfg.Conn, false)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	ctx = NewContext(ctx, s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := C(ctx, "test"); err != nil {
			b.Error(err)
		}
	}
}
