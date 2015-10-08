// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package trim implements methods and structures to convert/de-convert
// users' URLs. Also it controls their consistency in the database.
package trim

import (
    "errors"
    "fmt"
    "log"
    "math"
    "os"
    "sort"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
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
    // ErrPartlyDone is a error when the package-task was completed only partly.
    ErrPartlyDone = errors.New("task is not completed")
)

// CustomURL stores info about user's URL.
type CustomURL struct {
    Short     string     `bson:"_id"`
    Active    bool       `bson:"active"`
    Project   string     `bson:"prj"`
    Original  string     `bson:"orig"`
    User      string     `bson:"u"`
    TTL       *time.Time `bson:"ttl"`
    NotDirect bool       `bson:"ndr"`
    Spam      float64    `bson:"spam"`
    Created   time.Time  `bson:"ts"`
    Modified  time.Time  `bson:"mod"`
}

// String returns a representative form of CustomURL.
func (c *CustomURL) String() string {
    return fmt.Sprintf("%s => %s", c.Short, c.Original)
}

// LockColls adds a lock recored to name-collection
func LockColls(name string, conn *db.Conn) error {
    const maxAttempts = 3
    delay := time.Duration(10 * time.Millisecond)
    coll := conn.C(db.Colls["locks"])
    for i := 0; i < maxAttempts; i++ {
        _, err := coll.Upsert(bson.M{"_id": name, "locked": false}, bson.M{"_id": name, "locked": true})
        if err == nil {
            return nil
        }
        time.Sleep(time.Duration(i+1) * delay)
    }
    return fmt.Errorf("can't lock/update collection \"%v\" during %v attempts", db.Colls["locks"], maxAttempts)
}

// UnlockColls removes a lock recored from name-collection.
func UnlockColls(name string, conn *db.Conn) error {
    coll := conn.C(db.Colls["locks"])
    return coll.Update(bson.M{"_id": name, "locked": true}, bson.M{"$set": bson.M{"locked": false}})
}

// FindShort checks that url exists and returns it.
func FindShort(url string, c *conf.Config) (*CustomURL, error) {
    // look in the cache
    // of found, return simplified CustomURL (only links)
    if val, ok := c.Cache.LRU.Get(url); ok {
        // Logger.Printf("found in the cache: %v", url)
        return &CustomURL{Short: url, Original: string(val)}, nil
    }
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return nil, err
    }
    coll := conn.C(db.Colls["urls"])
    cu := &CustomURL{}
    err = coll.Find(bson.M{"_id": url, "active": true}).One(cu)
    if err != nil {
        return nil, err
    }
    // add to the cache
    c.Cache.LRU.Add(url, cu.Original)
    return cu, nil
}

// GetShort returns a new short URL.
func GetShort(c *conf.Config, cu ...*CustomURL) error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    coll := conn.C(db.Colls["urls"])
    // lock
    err = LockColls("urls", conn)
    if err != nil {
        return err
    }
    defer UnlockColls("urls", conn)
    short, err := getMax(coll)
    if err != nil {
        return err
    }
    for i := range cu {
        cu[i].Short = short
        err = coll.Insert(cu[i])
        if err != nil {
            Logger.Printf("link insert error [%v]: %v", i, err)
            if i > 0 {
                return ErrPartlyDone
            }
            return err
        }
        short = Inc(short)
    }
    return nil
}

// getMax returns a max short URLs, so it should be called
// in locked mode to get actual data.
func getMax(coll *mgo.Collection) (string, error) {
    maxURL := &CustomURL{}
    err := coll.Find(nil).Sort("-_id").Limit(1).One(maxURL)
    if err != nil {
        if err == mgo.ErrNotFound {
            return string(Alphabet[1]), nil
        }
        return "", err
    }
    return Inc(maxURL.Short), nil
}

// Inc increments a number from Alphabet-base numeral system.
func Inc(s string) string {
    n := len(s)
    if n == 0 {
        return string(Alphabet[0])
    }
    if s[n-1] == Alphabet[basis-1] {
        if n == 1 {
            return string(Alphabet[1]) + string(Alphabet[0])
        }
        s = Inc(s[:n-1]) + string(Alphabet[0])
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
