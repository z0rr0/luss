// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package trim implements methods and structures to convert/de-convert
// users' URLs. Also it controls their consistency in the database.
package trim

import (
    "fmt"
    "math"
    "sort"
    "time"

    // "gopkg.in/mgo.v2/bson"
)

const (
    // Alphabet is a sorted set of basis numeral system chars.
    Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

var (
    // basis is a numeral system basis
    basis = int64(len(Alphabet))
)

// CustomURL stores info about user's URL.
type CustomURL struct {
    Short     string    `bson:"_id"`
    Project   string    `bson:"project"`
    OriginURL string    `bson:"origin"`
    TTL       time.Time `bson:"ttl"`
    Spam      float64   `bson:"spam"`
    Num       int64
}

// Encode converts a decimal number to Alphabet-base numeral system.
func Encode(x int64) string {
    var result, sign string
    if x == 0 {
        return "0"
    }
    if x < 0 {
        sign, x = "-", -x
    }
    for x > 0 {
        i := int(x % basis)
        result = string(Alphabet[i]) + result
        x = x / basis
    }
    return sign + result
}

// Decode converts a Alphabet-base number to decimal one.
func Decode(x string) (int64, error) {
    var (
        result, j int64
        sign      bool
    )
    l := len(x)
    if l == 0 {
        return 0, nil
    }
    // the first character is '-'
    if x[0] == 0x2D {
        sign, x = true, x[1:l]
        l--
    }
    for i := l - 1; i >= 0; i-- {
        c := x[i]
        k := sort.Search(int(basis), func(t int) bool { return Alphabet[t] >= c })
        p := int64(k)
        if !((p < basis) && (Alphabet[k] == c)) {
            return 0, fmt.Errorf("can't convert %q", c)
        }
        result = result + p*pow(basis, j)
        j++
    }
    if sign {
        result = -result
    }
    return result, nil
}

// pow returns x**y, only uses int64 types instead float64.
func pow(x, y int64) int64 {
    return int64(math.Pow(float64(x), float64(y)))
}
