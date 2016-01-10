// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package trim implements methods and structures to convert users' URLs.
// Also it controls their consistency in the database.
package trim

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/z0rr0/luss/auth"
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
	// ErrEmptyCallback is error about empty empty callback usage.
	ErrEmptyCallback = errors.New("empty callback request")
	// isShortURL is regexp pattern to check short URL,
	// max int64 9223372036854775807 => AzL8n0Y58m7
	// real, max decode/encode 839299365868340223 <=> zzzzzzzzzz
	isShortURL = regexp.MustCompile(fmt.Sprintf("^[%s]{1,10}$", Alphabet))
	// logger is a logger for error messages
	logger = log.New(os.Stderr, "LOGGER [trim]: ", log.Ldate|log.Ltime|log.Lshortfile)
	// basis is a numeral system basis
	basis = int64(len(Alphabet))
)

// CallBack is callback info.
type CallBack struct {
	URL    string `bson:"u"`
	Method string `bson:"m"`
	Name   string `bson:"name"`
	Value  string `bson:"value"`
}

// CustomURL stores info about user's URL.
type CustomURL struct {
	ID        int64      `bson:"_id"`
	Disabled  bool       `bson:"off"`
	Group     string     `bson:"group"`
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

// Filter is a data filter to export URLs info.
type Filter struct {
	Group  string
	Tag    string
	Period [2]*time.Time
	Active bool
}

// ReqParams is request parameters required for new
// short URL creation.
type ReqParams struct {
	Original  string
	Tag       string
	Group     string
	NotDirect bool
	TTL       *time.Time
	Cb        CallBack
}

// ChangeResult is result of CustomURL pack change.
type ChangeResult struct {
	Cu  *CustomURL
	Err string
}

// String returns short string URL without domain prefix.
func (cu *CustomURL) String() string {
	return Encode(cu.ID)
}

// String returns request info.
func (rp *ReqParams) String() string {
	return rp.Original
}

// Valid checks ReqParams values.
func (rp *ReqParams) Valid() error {
	const lenLimit = 255
	if rp.Original == "" {
		return errors.New("empty request parameters")
	}
	if len(rp.Tag) > lenLimit {
		return errors.New("too long tag value")
	}
	if len(rp.Group) > lenLimit {
		return errors.New("too long group name")
	}
	u, err := url.ParseRequestURI(rp.Original)
	if err != nil {
		return err
	}
	rp.Original = u.String()
	if rp.Cb.URL != "" {
		u, err = url.ParseRequestURI(rp.Cb.URL)
		if err != nil {
			return err
		}
		rp.Cb.URL = u.String()
		if rp.Cb.Method != "GET" && rp.Cb.Method != "POST" {
			return errors.New("unknown or not allowed request method")
		}
	}
	return nil
}

// Callback returns a prepared body request Reader as bytes.Buffer pointer.
func (cu *CustomURL) Callback() (*http.Request, error) {
	if cu.Cb.URL == "" {
		return nil, ErrEmptyCallback
	}
	if cu.Cb.Method != "GET" && cu.Cb.Method != "POST" {
		return nil, errors.New("unknown callback method")
	}
	params := url.Values{}
	if cu.Cb.Name != "" {
		params.Add(cu.Cb.Name, cu.Cb.Value)
	}
	params.Add("id", cu.String())
	params.Add("tag", cu.Tag)
	body := bytes.NewBufferString(params.Encode())
	return http.NewRequest(cu.Cb.Method, cu.Cb.URL, body)
}

// String returns main callback info.
func (cb *CallBack) String() string {
	return fmt.Sprintf("%v:%v", cb.Method, cb.URL)
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

// MultiLengthen returns short URLs info for slice of links.
func MultiLengthen(ctx context.Context, links []string) ([]ChangeResult, error) {
	var result []ChangeResult
	c, err := conf.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	n := len(links)
	if n > c.Settings.MaxPack {
		return nil, fmt.Errorf("too big pack size [%v]", n)
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
	for _, link := range links {
		id, err := Decode(link)
		if err != nil {
			logger.Printf("decode error [%v]: %v", link, err)
			result = append(result, ChangeResult{Cu: &CustomURL{ID: id}, Err: "invalid value"})
			continue
		}
		cu := &CustomURL{}
		err = coll.FindId(id).One(cu)
		if err != nil {
			msg := "internal error"
			if err == mgo.ErrNotFound {
				msg = "not found"
			}
			result = append(result, ChangeResult{Cu: &CustomURL{ID: id}, Err: msg})
			continue
		}
		result = append(result, ChangeResult{Cu: cu})
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
	cache, cacheOn := c.Cache.Strorage["URL"]
	if cacheOn {
		if cu, ok := cache.Get(short); ok {
			// c.L.Debug.Println("read from LRU cache", short)
			return cu.(*CustomURL), nil
		}
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
	if cacheOn {
		cache.Add(short, cu)
	}
	return cu, nil
}

// Shorten returns new short links.
func Shorten(ctx context.Context, params []*ReqParams) ([]*CustomURL, error) {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	u, err := auth.ExtractUser(ctx)
	if err != nil {
		return nil, err
	}
	// check URLs pack size
	n := len(params)
	if n > c.Settings.MaxPack {
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
			Group:     param.Group,
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

// IsShort checks link can be short URL.
func IsShort(link string) (string, bool) {
	pattern := strings.Trim(link, "/")
	return pattern, isShortURL.MatchString(pattern)
}

// Import imports short URLs.
func Import(ctx context.Context, links map[string]*ReqParams) ([]ChangeResult, error) {
	var result []ChangeResult
	c, err := conf.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	u, err := auth.ExtractUser(ctx)
	if err != nil {
		return nil, err
	}
	n := len(links)
	if n > c.Settings.MaxPack {
		return nil, fmt.Errorf("too big pack size [%v]", n)
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
	now := time.Now().UTC()
	for short, param := range links {
		num, err := Decode(short)
		if err != nil {
			result = append(result, ChangeResult{Err: "invalid short URL value"})
			continue
		}
		cu := &CustomURL{
			ID:        num,
			Group:     param.Group,
			Tag:       param.Tag,
			Original:  param.Original,
			User:      u.Name,
			TTL:       param.TTL,
			NotDirect: param.NotDirect,
			Created:   now,
			Modified:  now,
			Cb:        param.Cb,
		}
		err = coll.Insert(cu)
		if err != nil {
			msg := "internal error"
			if mgo.IsDup(err) {
				msg = "duplicate item"
			}
			result = append(result, ChangeResult{Err: msg})
			continue
		}
		result = append(result, ChangeResult{Cu: cu})
	}
	return result, nil
}

// Export exports URLs data.
func Export(ctx context.Context, filter Filter) ([]*CustomURL, error) {
	var result []*CustomURL
	s, err := db.CtxSession(ctx)
	if err != nil {
		return nil, err
	}
	coll, err := db.Coll(s, "urls")
	if err != nil {
		return nil, err
	}
	conditions := bson.M{"group": filter.Group, "tag": filter.Tag}
	if filter.Active {
		conditions["off"] = false
	}
	switch {
	case filter.Period[0] != nil && filter.Period[1] != nil:
		conditions["$and"] = []bson.M{
			bson.M{"ts": bson.M{"$gte": *filter.Period[0]}},
			bson.M{"ts": bson.M{"$lte": *filter.Period[1]}},
		}
	case filter.Period[0] != nil:
		conditions["ts"] = bson.M{"$gte": *filter.Period[0]}
	case filter.Period[1] != nil:
		conditions["ts"] = bson.M{"$lte": *filter.Period[1]}
	}
	err = coll.Find(conditions).All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
