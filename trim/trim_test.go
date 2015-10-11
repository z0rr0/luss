// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package trim implements methods and structures to convert/de-convert
// users' URLs. Also it controls their consistency in the database.
package trim

import (
    "testing"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "github.com/z0rr0/luss/test"
    "github.com/z0rr0/luss/utils"
)

func TestEncode(t *testing.T) {
    suite := map[int64]string{
        -1:  "-1",
        -9:  "-9",
        -11: "-B",
        0:   "0",
        5:   "5",
        10:  "A",
        61:  "z",
        62:  "10",
        72:  "1A",
        124: "20",
        129: "25",
    }
    for k, v := range suite {
        if s := Encode(k); s != v {
            t.Errorf("incorrect values: %v != %v", s, v)
        }
        if num, err := Decode(v); (err != nil) || (num != k) {
            t.Errorf("incorrect values: %v, %v, %v", err, num, k)
        }
    }
    if _, err := Decode("34.56"); err == nil {
        t.Error("unexpected behavior")
    }
}

func TestInc(t *testing.T) {
    suite := map[string]string{
        "":                   "0",
        "1":                  "2",
        "a":                  "b",
        "A":                  "B",
        "z":                  "10",
        "11":                 "12",
        "AZ":                 "Aa",
        "Az":                 "B0",
        "zz":                 "100",
        "1zz":                "200",
        "ABC123zzzzzzzzzzzz": "ABC124000000000000",
    }
    for k, v := range suite {
        if i := Inc(k); i != v {
            t.Errorf("incorrect values: %v != %v", i, v)
        }
    }
    for i := 1; i < 100000; i++ {
        num := int64(i - 1)
        si := Encode(num)
        if s := Encode(num + 1); s != Inc(si) {
            t.Errorf("bad result: %v %v", s, Inc(si))
        }
    }
}

func TestGetShort(t *testing.T) {
    const (
        longURL = "http://mydomain.com"
    )
    err := utils.InitConfig(test.TcConfigName(), true)
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    cfg := utils.Cfg
    err = db.CleanCollection(cfg.Conf, db.Colls["urls"])
    if err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    now := time.Now().UTC()
    cu := &CustomURL{
        Active:    true,
        Project:   conf.DefaultProject,
        Original:  longURL,
        User:      conf.AnonName,
        TTL:       nil,
        NotDirect: false,
        Spam:      0,
        Created:   now,
        Modified:  now,
    }
    if err := GetShort(cfg.Conf, cu); err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    defer db.CleanCollection(cfg.Conf, db.Colls["urls"])
    if (cu.Original != longURL) || (cu.String() == "") {
        t.Errorf("incorrect behavior")
    }
    // add many
    cus := make([]*CustomURL, cfg.Conf.Projects.MaxPack+1)
    for i := 0; i < cfg.Conf.Projects.MaxPack+1; i++ {
        cus[i] = &CustomURL{
            Active:    true,
            Project:   conf.DefaultProject,
            Original:  longURL,
            User:      conf.AnonName,
            TTL:       nil,
            NotDirect: false,
            Spam:      0,
            Created:   now,
            Modified:  now,
        }
    }
    if err := GetShort(cfg.Conf, cus[:cfg.Conf.Projects.MaxPack]...); err != nil {
        t.Errorf("invalid: %v", err)
        return
    }
    cus[cfg.Conf.Projects.MaxPack] = cu
    if err := GetShort(cfg.Conf, cus...); err == nil {
        t.Errorf("invalid: %v", err)
        return
    }
    fcu, err := FindShort(cu.Short, cfg.Conf)
    if err != nil {
        t.Errorf("invalid: %v", err)
    }
    if fcu.Original != longURL {
        t.Errorf("incorrect behavior")
    }
    // get from cache
    fcu, err = FindShort(cu.Short, cfg.Conf)
    if err != nil {
        t.Errorf("invalid: %v", err)
    }
    if fcu.Original != longURL {
        t.Errorf("incorrect behavior")
    }
}

func BenchmarkEncode(b *testing.B) {
    // max 9223372036854775807 == AzL8n0Y58m7
    x := "AzL8n0Y58m7"
    for i := 0; i < b.N; i++ {
        num, err := Decode(x)
        if err != nil {
            b.Fatal(err)
        }
        if s := Encode(num); s != x {
            b.Fatalf("bad result: %v %v", s, x)
        }
    }
}

// 2127 ns/op  176 B/op
func BenchmarkInc1(b *testing.B) {
    x, y := "Ayzzzzzzzzz", "Az000000000"
    for i := 0; i < b.N; i++ {
        num, err := Decode(x)
        if err != nil {
            b.Fatal(err)
        }
        num = num + 1
        if s := Encode(num); s != y {
            b.Fatalf("bad result: %v %v", s, x)
        }
    }
}

func BenchmarkInc2(b *testing.B) {
    x, y := "Ayzzzzzzzzz", "Az000000000"
    for i := 0; i < b.N; i++ {
        if s := Inc(x); s != y {
            b.Fatalf("bad result: %v %v", s, x)
        }
    }
}
