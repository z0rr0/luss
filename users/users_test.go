// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package users implements users handling methods.
package users

import (
    "log"
    "testing"

    "github.com/z0rr0/luss/test"
    "github.com/z0rr0/luss/utils"
)

func TestGenRndBytes(t *testing.T) {
    values := []int{0, 1, 1, 2, 3, 5, 8, 13}
    for _, v := range values {
        r, err := GenRndBytes(v)
        if err != nil {
            t.Errorf("invalid value: %v", err)
        }
        if len(r) != v {
            t.Error("wrong behavior")
        }
    }
}

func TestCheckToken(t *testing.T) {
    name := test.TcConfigName()
    err := utils.InitConfig(name, true)
    if err != nil {
        t.Errorf("invalid value: %v", err)
        return
    }
    c := utils.Cfg
    p1, p2, err := genToken(c.Conf)
    if err != nil {
        t.Errorf("invalid value: %v", err)
    }
    val := p1 + p2
    if len(val) == 0 {
        t.Error("wrong behavior")
    }
    // bad tokens
    if err := CheckToken("", c.Conf); err == nil {
        t.Error("wrong behavior")
    }
    if err := CheckToken("abc", c.Conf); err == nil {
        t.Error("wrong behavior")
    }
    if err := CheckToken("abcdefabcdefabcdefabcdefabcdefabcdefab", c.Conf); err == nil {
        t.Error("wrong behavior")
    }
    if err := CheckToken("M<.", c.Conf); err == nil {
        t.Error("wrong behavior")
    }
    // good token
    err = CheckToken(val, c.Conf)
    if err != nil {
        t.Errorf("invalid value: %v", err)
    }
}

func TestCreateUser(t *testing.T) {
    name := test.TcConfigName()
    err := utils.InitConfig(name, true)
    if err != nil {
        t.Errorf("invalid value: %v", err)
        return
    }
    c := utils.Cfg.Conf
    DeleteUser("test", c)

    bName, err := GenRndBytes(260)
    if _, err := CreateUser(string(bName), "test", c); err == nil {
        t.Error("wrong behavior")
    }
    u, err := CreateUser("test", "test", c)
    if err != nil {
        t.Errorf("invalid value: %v", err)
        return
    }
    if u.Secret == "" {
        t.Error("wrong behavior")
    }
    if _, err := CreateUser("test", "", c); err != nil {
        t.Errorf("invalid value: %v", err)
    }
    if err := DeleteUser("test", c); err != nil {
        t.Errorf("invalid value: %v", err)
    }
}

func BenchmarkCheckToken(b *testing.B) {
    name := test.TcConfigName()
    err := utils.InitConfig(name, true)
    if err != nil {
        log.Println("error: %v", err)
        return
    }
    c := utils.Cfg.Conf
    p1, p2, err := genToken(c)
    if err != nil {
        log.Println("invalid value: %v", err)
    }
    val := p1 + p2
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        err = CheckToken(val, c)
        if err != nil {
            log.Println("error: %v", err)
            break
        }
    }
}
