// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package conf implements MongoDB database access method.
package conf

import (
    "testing"

    "github.com/z0rr0/luss/test"
)

func TestParseConfig(t *testing.T) {
    name := "bad"
    cfg, err := ParseConfig(name)
    if err == nil {
        t.Errorf("incorrect behavior")
    }
    name = test.TcConfigName()
    cfg, err = ParseConfig(name + "  ")
    if err != nil {
        t.Errorf("invalid [%v]: %v", name, err)
    }
    if cfg == nil {
        t.Errorf("incorrect behavior")
    }
    // check mongo addresses
    if len(cfg.Db.Addrs()) == 0 {
        t.Errorf("incorrect behavior")
    }
    // salt test
    if !cfg.GoodSalts() {
        t.Errorf("incorrect behavior")
    }
    // conn cap
    values := map[int]int{2: 2, 10: 8, 70: 16, 500: 32}
    for k, v := range values {
        cfg.Cache.DbPoolSize = k
        if c := cfg.ConnCap(); c != v {
            t.Errorf("invalid: %v != %v", c, v)
        }
    }
    if cfg.Address("domain.com") == "" {
        t.Errorf("incorrect behavior")
    }
}
