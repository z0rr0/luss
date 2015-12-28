// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package trim implements methods and structures to convert users' URLs.
// Also it controls their consistency in the database.
package trim

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	// Alphabet is a sorted set of basis numeral system chars.
	Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

var (
	// basis is a numeral system basis
	basis = int64(len(Alphabet))
)

// CallBack is callback info.
type CallBack struct {
	URL       string `bson:"u"`
	Method    string `bson:"m"`
	Parameter string `bson:"p"`
}

// CustomURL stores info about user's URL.
type CustomURL struct {
	ID        int64     `bson:"_id"`
	Active    bool      `bson:"active"`
	Project   string    `bson:"prj"`
	Tag       string    `bson:"tag"`
	Original  string    `bson:"orig"`
	User      string    `bson:"u"`
	TTL       time.Time `bson:"ttl"`
	NotDirect bool      `bson:"ndr"`
	Spam      float64   `bson:"spam"`
	Created   time.Time `bson:"ts"`
	Modified  time.Time `bson:"mod"`
	Cb        CallBack  `bson:"cb"`
	Short     string    `bson:",omitempty"`
}

// getMax returns a max short URLs, so it should be called
// in locked primary mode to get actual data.
func getMax(coll *mgo.Collection) (int64, error) {
	maxURL := &db.ItemURL{}
	err := coll.Find(nil).Sort("-_id").Limit(1).One(maxURL)
	if err != nil {
		if err == mgo.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	return maxURL.ID, nil
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
	if x[0] == '-' {
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

// Lengthen converts a short link to original one.
func Lengthen(ctx context.Context, short string) (*CustomURL, error) {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	cache := c.Cache.URLsLRU
	if cu, ok := cache.Get(short); ok {
		return cu.(*CustomURL), nil
	}
	num, err := Decode(short)
	if err != nil {
		return nil, err
	}
	s, err := db.NewSession(c.Conn, false)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	coll, err := db.Coll(s, "urls")
	if err != nil {
		return nil, err
	}
	cu := &CustomURL{}
	err = coll.Find(bson.M{"_id": num, "active": true}).One(cu)
	if err != nil {
		return nil, err
	}
	cache.Add(short, cu)
	return cu, nil
}

// Shorten returns new short link.
// func Shorten(ctx context.Context, original string) (*CustomURL, error) {

// }
