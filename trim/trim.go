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
	"github.com/z0rr0/luss/project"
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
	ID        int64      `bson:"_id"`
	Disabled  bool       `bson:"off"`
	Project   string     `bson:"prj"`
	Tag       string     `bson:"tag"`
	Original  string     `bson:"orig"`
	User      string     `bson:"u"`
	TTL       *time.Time `bson:"ttl"`
	NotDirect bool       `bson:"ndr"`
	Spam      float64    `bson:"spam"`
	Created   time.Time  `bson:"ts"`
	Modified  time.Time  `bson:"mod"`
	Cb        CallBack   `bson:"cb"`
}

// ReqParams is request parameters required for new
// short URL creation.
type ReqParams struct {
	Original  string
	Tag       string
	NotDirect bool
	TTL       *time.Time
	Cb        CallBack
}

// String return short string URL without domain prefix.
func (c *CustomURL) String() string {
	return Encode(c.ID)
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
// It uses own database session if it's needed
// or it gets data from the cache.
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
	err = coll.Find(bson.M{"_id": num, "off": false}).One(cu)
	if err != nil {
		return nil, err
	}
	cache.Add(short, cu)
	return cu, nil
}

// Shorten returns new short link.
func Shorten(ctx context.Context, params []ReqParams) ([]*CustomURL, error) {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	u, err := project.ExtractUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := project.ExtractProject(ctx)
	if err != nil {
		return nil, err
	}
	// check URLs pack size
	n := len(params)
	if n > c.Projects.MaxPack {
		return nil, fmt.Errorf("too big ReqParams pack size [%v]", n)
	}
	s, err := db.CtxSession(ctx)
	if err != nil {
		return nil, err
	}
	err = db.LockURL(s)
	if err != nil {
		return nil, err
	}
	defer db.UnlockURL(s)
	// prepare
	coll, err := db.Coll(s, "urls")
	if err != nil {
		return nil, err
	}
	num, err := getMax(coll)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	documents := make([]interface{}, n)
	cus := make([]*CustomURL, n)
	for i, param := range params {
		num++
		cus[i] = &CustomURL{
			ID:        num,
			Project:   p.Name,
			Tag:       param.Tag,
			Original:  param.Original,
			User:      u.Name,
			TTL:       param.TTL,
			NotDirect: param.NotDirect,
			Created:   now,
			Modified:  now,
			Cb:        param.Cb,
		}
		documents[i] = cus[i]
	}
	err = coll.Insert(documents...)
	if err != nil {
		c.L.Error.Println(err)
		return nil, err
	}
	return cus, nil
}
