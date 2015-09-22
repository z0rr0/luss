// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package db implements MongoDB database access methods.
package db

import (
    "testing"
    // "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/test"
)

func TestNewConnPool(t *testing.T) {
    cfg, err := conf.ParseConfig(test.TcConfigName())
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    pool, err := NewConnPool(nil)
    if err == nil {
        t.Errorf("incorrect behavior")
        return
    }
    cfg.Cache.DbPoolSize = 0
    pool, err = NewConnPool(cfg)
    if err == nil {
        t.Errorf("incorrect behavior")
        return
    }
    cfg.Cache.DbPoolSize = 1
    pool, err = NewConnPool(cfg)
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    t.Log(pool)
    oldPass := cfg.Db.Password
    cfg.Db.Password = "bad password"
    conn, err := GetConn(cfg)
    if err == nil {
        t.Errorf("incorrect behavior")
        return
    }
    ReleaseConn(conn)
    Logger.Println("check valid value")
    cfg.Db.Password, cfg.Db.MongoCred = oldPass, nil
    // all is ok, check connection
    conn, err = GetConn(cfg)
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    defer conn.Close(0)
    defer ReleaseConn(conn)
    if conn.Session == nil {
        t.Errorf("incorrect behavior")
    }
}
