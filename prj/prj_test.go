// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package prj implements methods to handler projects/users activities.
package prj

import (
    "testing"

    "github.com/z0rr0/luss/db"
    "github.com/z0rr0/luss/test"
    "github.com/z0rr0/luss/utils"
)

func TestGenRndBytes(t *testing.T) {
    values := []int{0, 1, 1, 2, 3, 5, 8, 13}
    for _, v := range values {
        r, err := getRndBytes(v)
        if err != nil {
            t.Errorf("invalid value: %v", err)
        }
        if len(r) != v {
            t.Error("wrong behavior")
        }
    }
}

func TestEqualBytes(t *testing.T) {
    s := "test string-試験の文字列-тестовая строка"
    sb := []byte{
        116, 101, 115, 116, 32, 115, 116, 114, 105, 110, 103, 45, 232, 169,
        166, 233, 168, 147, 227, 129, 174, 230, 150, 135, 229, 173, 151, 229,
        136, 151, 45, 209, 130, 208, 181, 209, 129, 209, 130, 208, 190, 208, 178,
        208, 176, 209, 143, 32, 209, 129, 209, 130, 209, 128, 208, 190, 208, 186, 208, 176}
    if !EqualBytes([]byte(s), sb) {
        t.Error("wrong behavior")
    }
    n := len(sb)
    if EqualBytes([]byte(s), sb[:n-1]) {
        t.Error("wrong behavior")
    }
    sb[n-1] = 177
    if EqualBytes([]byte(s), sb) {
        t.Error("wrong behavior")
    }
}

func TestCheckToken(t *testing.T) {
    err := utils.InitConfig(test.TcConfigName(), true)
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    cfg := utils.Cfg
    p1, p2, err := genToken(cfg.Conf)
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    if _, err := CheckToken("", cfg.Conf); err == nil {
        t.Error("wrong behavior")
    }
    if _, err := CheckToken("bad", cfg.Conf); err == nil {
        t.Error("wrong behavior")
    }
    if _, err := CheckToken("1234", cfg.Conf); err == nil {
        t.Error("wrong behavior")
    }
    if _, err := CheckToken(p1+p2, cfg.Conf); err != nil {
        t.Errorf("invalid: %v", err)
    }
}

func TestCreateAdmin(t *testing.T) {
    adminName := "test-admin"
    err := utils.InitConfig(test.TcConfigName(), true)
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    cfg := utils.Cfg
    u, err := CreateAdmin(adminName, cfg.Conf)
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    defer db.CleanCollection(cfg.Conf, db.Colls["projects"])
    if (u.Name != adminName) || (u.Role != "admin") || (u.Secret == "") {
        t.Errorf("invalid: %v", u)
    }
}
