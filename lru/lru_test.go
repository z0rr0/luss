// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package lru implements LRU cache methods to speed up search already saved short URLs.
package lru

import (
    "fmt"
    "testing"
)

func TestNew(t *testing.T) {
    // disabled
    c := New(0)
    c.Add("key", "value")
    if c.Len() != 0 {
        t.Errorf("incorrect behavior")
    }
    if _, ok := c.Get("key"); ok {
        t.Errorf("incorrect behavior")
    }
    c.Remove("key")
    // active
    c = New(2)
    c.Add("k1", "v1")
    if c.Len() == 0 {
        t.Errorf("incorrect behavior")
    }
    if _, ok := c.Get("k1"); !ok {
        t.Errorf("incorrect behavior")
    }
    c.Add("k2", "v2")
    c.Add("k3", "v3")
    c.Add("k2", "v2")
    if c.Len() != 2 {
        t.Errorf("incorrect behavior")
    }
    if _, ok := c.Get("k1"); ok {
        t.Errorf("incorrect behavior")
    }
    if _, ok := c.Get("k2"); !ok {
        t.Errorf("incorrect behavior")
    }
    c.Remove("k4")
    c.Remove("k2")
    if _, ok := c.Get("k2"); ok {
        t.Errorf("incorrect behavior")
    }
}

func BenchmarkLRU(b *testing.B) {
    var (
        size       = 4066
        k          = "short_URL_string"
        v    Value = "long URL sring"
    )
    c := New(size)
    for i := 0; i < b.N; i++ {
        key := Key(fmt.Sprintf("%s-%v", k, i))
        c.Add(key, v)
        c.Get(key)
    }
}
