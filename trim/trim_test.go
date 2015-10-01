// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package trim implements methods and structures to convert/de-convert
// users' URLs. Also it controls their consistency in the database.
package trim

import (
    "testing"
)

// 0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz

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

// 876 ns/op  160 B/op
func BenchmarkInc2(b *testing.B) {
    x, y := "Ayzzzzzzzzz", "Az000000000"
    for i := 0; i < b.N; i++ {
        if s := Inc(x); s != y {
            b.Fatalf("bad result: %v %v", s, x)
        }
    }
}
