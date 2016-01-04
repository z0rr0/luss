// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package test contains additional methods for testing.
package test

import (
    "os"
    "testing"
)

func TestTcBuildDir(t *testing.T) {
    v := TcBuildDir()
    if v == "" {
        t.Errorf("icorrect behavior")
    }
    _, err := os.Stat(v)
    if err != nil {
        t.Errorf("invalid: %v", err)
    }
}

func TestTcConfigName(t *testing.T) {
    v := TcConfigName()
    if len(v) <= len(Config) {
        t.Errorf("icorrect behavior")
    }
    _, err := os.Stat(v)
    if err != nil {
        t.Errorf("invalid: %v", err)
    }
}
