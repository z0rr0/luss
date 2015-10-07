// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package prj implements methods to handler projects/users activities.
package prj

import (
    "testing"
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
