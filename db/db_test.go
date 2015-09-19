// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package db implements MongoDB database access methods.
package db

import (
    "testing"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/test"
)

func checkConn(cfg *conf.MongoCfg, ch chan *Conn) error {
    conn, err := GetConn(cfg, ch)
    if err != nil {
        return err
    }
    defer conn.Release()
    Logger.Println("DB connection checked")
    return err
}

func TestNewConnPool(t *testing.T) {
    cf, err := conf.ParseConfig(test.TcConfigName())
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    // bad pool
    pool := NewConnPool(0)
    ch, errch := make(chan *Conn, pool.Cap()), make(chan error)
    go pool.Produce(ch, errch)
    err = <-errch
    if err == nil {
        t.Errorf("icorrect behavior")
        return
    }
    // good pool
    pool = NewConnPool(5)
    ch, errch = make(chan *Conn, pool.Cap()), make(chan error)
    go pool.Produce(ch, errch)
    err = <-errch
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    d := 250 * time.Millisecond
    go pool.Monitor(d)
    for i := 0; i < 10; i++ {
        err = checkConn(&cf.Db, ch)
        if err != nil {
            t.Errorf("invalid: %v", err)
        }
    }
    time.Sleep(d)
}

func TestCap(t *testing.T) {
    sizes := map[int]int{0: 0, 1: 1, 31: 31, 48: 16, 64: 16, 120: 16, 130: 32, 250: 32, 1000: 32}
    for k, v := range sizes {
        p := NewConnPool(k)
        if p.Cap() != v {
            t.Errorf("invalid: %v != %v", p.Cap(), v)
        }
    }

}
