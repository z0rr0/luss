// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package trim implements methods and structures to convert/de-convert
// users' URLs. Also it controls their consistency in the database.
package trim

import (
    "fmt"
    "log"
    "math"
    "os"
    "sort"
    "time"

    "golang.org/x/net/idna"
)

const (
    // Alphabet is a sorted set of basis numeral system chars.
    Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

var (
    // basis is a numeral system basis
    basis = len(Alphabet)
    // Logger is a logger for error messages
    Logger = log.New(os.Stderr, "LOGGER [luss/trim]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// CustomURL stores info about user's URL.
type CustomURL struct {
    Short     string     `bson:"_id"`
    Project   string     `bson:"project"`
    OriginURL string     `bson:"origin"`
    TTL       *time.Time `bson:"ttl"`
    Spam      float64    `bson:"spam"`
}

func ToShort(url string) (*CustomURL, error) {
    s, err := idna.ToASCII(url)
    if err != nil {
        return nil, err
    }
    c := &CustomURL{Short: s, Project: "default", OriginURL: url, TTL: nil, Spam: 0}
    return c, nil
}

// Inc increments a number from Alphabet-base numeral system.
func Inc(s string) string {
    n := len(s)
    if n == 0 {
        return "0"
    }
    if s[n-1] == Alphabet[basis-1] {
        if n == 1 {
            return "10"
        }
        s = Inc(s[:n-1]) + "0"
    } else {
        i := sort.Search(basis, func(j int) bool { return Alphabet[j] >= s[n-1] })
        if (i < basis) && (Alphabet[i] == s[n-1]) {
            s = s[:n-1] + string(Alphabet[i+1])
        } else {
            Logger.Panicf("unexpected behavior: %q is not found in \"%v\"", s[n-1], Alphabet)
        }
    }
    return s
}

// pow returns x**y, only uses int64 types instead float64.
func pow(x, y int64) int64 {
    return int64(math.Pow(float64(x), float64(y)))
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
    b := int64(basis)
    for x > 0 {
        i := int(x % b)
        result = string(Alphabet[i]) + result
        x = x / b
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
    if x[0] == '-' {
        sign, x = true, x[1:l]
        l--
    }
    b := int64(basis)
    for i := l - 1; i >= 0; i-- {
        c := x[i]
        k := sort.Search(basis, func(t int) bool { return Alphabet[t] >= c })
        p := int64(k)
        if !((p < b) && (Alphabet[k] == c)) {
            return 0, fmt.Errorf("can't convert %q", c)
        }
        result = result + p*pow(b, j)
        j++
    }
    if sign {
        result = -result
    }
    return result, nil
}
