// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

package conf

import (
	"strings"
	"testing"

	"github.com/z0rr0/luss/test"
)

func TestParseConfig(t *testing.T) {
	name := "bad"
	cfg, err := Parse(name)
	if err == nil {
		t.Fatal("incorrect behavior")
	}
	name = test.TcConfigName()
	cfg, err = Parse(name + "  ")
	if err != nil {
		t.Fatalf("invalid [%v]: %v", name, err)
	}
	if cfg == nil {
		t.Errorf("incorrect behavior")
	}
	// check mongo addresses
	if len(cfg.Db.Addrs()) == 0 {
		t.Errorf("incorrect behavior")
	}
	if u := cfg.Address(""); !strings.HasPrefix(u, "http") {
		t.Errorf("incorrect behavior")
	}
	// validate parameters
	oldDomain := cfg.Domain.Name
	cfg.Domain.Name = ""
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Domain.Name = oldDomain

	oldPort := cfg.Listener.Port
	cfg.Listener.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Listener.Port = oldPort

	oldTimeout := cfg.Listener.Timeout
	cfg.Listener.Timeout = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Listener.Timeout = oldTimeout

	oldSecurityTokenLen := cfg.Listener.Security.TokenLen
	cfg.Listener.Security.TokenLen = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Listener.Security.TokenLen = oldSecurityTokenLen

	oldSecuritySalt := cfg.Listener.Security.Salt
	cfg.Listener.Security.Salt = "abc"
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Listener.Security.Salt = oldSecuritySalt

	oldMaxSpam := cfg.Settings.MaxSpam
	cfg.Settings.MaxSpam = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Settings.MaxSpam = oldMaxSpam

	oldCleanMin := cfg.Settings.CleanMin
	cfg.Settings.CleanMin = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Settings.CleanMin = oldCleanMin

	oldCbNum := cfg.Settings.CbNum
	cfg.Settings.CbNum = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Settings.CbNum = oldCbNum

	oldCbBuf := cfg.Settings.CbBuf
	cfg.Settings.CbBuf = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Settings.CbBuf = oldCbBuf

	oldCbLength := cfg.Settings.CbLength
	cfg.Settings.CbLength = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Settings.CbLength = oldCbLength

	oldMaxName := cfg.Settings.MaxName
	cfg.Settings.MaxName = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Settings.MaxName = oldMaxName

	oldMaxPack := cfg.Settings.MaxPack
	cfg.Settings.MaxPack = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Settings.MaxPack = oldMaxPack

	cfg, err = Parse(name)
	if err != nil {
		t.Fatal("incorrect behavior")
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("incorrect behavior")
	}
}

func TestNewContext(t *testing.T) {
	cfg, err := Parse(test.TcConfigName())
	if err != nil {
		t.Fatalf("invalid")
	}
	if cfg == nil {
		t.Errorf("incorrect behavior")
	}
	ctx := NewContext(cfg)
	if cfg2, err := FromContext(ctx); err != nil || cfg2 != cfg {
		t.Errorf("incorrect behavior")
	}
}
