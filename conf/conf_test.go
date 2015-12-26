// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package conf implements MongoDB database access method.
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

	oldMaxSpam := cfg.Projects.MaxSpam
	cfg.Projects.MaxSpam = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Projects.MaxSpam = oldMaxSpam

	oldCleanMin := cfg.Projects.CleanMin
	cfg.Projects.CleanMin = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Projects.CleanMin = oldCleanMin

	oldCbNum := cfg.Projects.CbNum
	cfg.Projects.CbNum = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Projects.CbNum = oldCbNum

	oldCbBuf := cfg.Projects.CbBuf
	cfg.Projects.CbBuf = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Projects.CbBuf = oldCbBuf

	oldCbLength := cfg.Projects.CbLength
	cfg.Projects.CbLength = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Projects.CbLength = oldCbLength

	oldMaxName := cfg.Projects.MaxName
	cfg.Projects.MaxName = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Projects.MaxName = oldMaxName

	oldMaxPack := cfg.Projects.MaxPack
	cfg.Projects.MaxPack = 0
	if err := cfg.Validate(); err == nil {
		t.Errorf("incorrect behavior")
	}
	cfg.Projects.MaxPack = oldMaxPack

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
