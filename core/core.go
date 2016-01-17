// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package core contains main internal methods.
package core

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/z0rr0/luss/auth"
	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/stats"
	"github.com/z0rr0/luss/trim"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	// trackerKey is a key to set/get tracker channel from context.
	trackerKey key = 1
	// trackerBuffer is a size of tracker channel.
	trackerBuffer = 32
)

var (
	// logger is a logger for error messages
	logger = log.New(os.Stderr, "LOGGER [core]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// key is a context key type.
type key int

// ErrHandler is a struct that contains handler error
// and returned HTTP status code.
type ErrHandler struct {
	Err    error
	Status int
}

// CuInfo is trim.CustomURL info with context.
type CuInfo struct {
	ctx  context.Context
	cu   *trim.CustomURL
	addr string
}

// String return main string info about error handler.
func (eh ErrHandler) String() string {
	return fmt.Sprintf("%d: %v", eh.Status, eh.Err)
}

// tracker saves info customer short URL request.
func tracker(ch <-chan *CuInfo) {
	var wg sync.WaitGroup
	for cui := range ch {
		c, err := conf.FromContext(cui.ctx)
		if err != nil {
			logger.Printf("tracker wasn't called, error: %v", err)
			continue
		}
		if !c.Settings.TrackOn {
			c.L.Debug.Println("request tracking is disabled")
			continue
		}
		wg.Add(2)
		// tracker handler
		go func() {
			defer wg.Done()
			if err := stats.Tracker(cui.ctx, cui.cu, cui.addr); err != nil {
				c.L.Error.Println(err)
			}
		}()
		// callback handler
		go func() {
			defer wg.Done()
			// anonymous callbacks will not be handled
			if cui.cu.User != auth.Anonymous {
				if err := stats.Callback(cui.ctx, cui.cu); err != nil {
					c.L.Error.Println(err)
				}
			}
		}()
		wg.Wait()
	}
}

// RunWorkers runs tracker workers.
func RunWorkers(ctx context.Context) (context.Context, error) {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return ctx, err
	}
	ch := make(chan *CuInfo, trackerBuffer)
	for i := 0; i < c.Settings.Trackers; i++ {
		go func() {
			tracker(ch)
		}()
	}
	c.L.Info.Printf("run %v trackers", c.Settings.Trackers)
	return context.WithValue(ctx, trackerKey, ch), nil
}

// TrackerChan extracts tracker channel.
func TrackerChan(ctx context.Context) (chan *CuInfo, error) {
	p, ok := ctx.Value(trackerKey).(chan *CuInfo)
	if !ok {
		return nil, errors.New("not found context tracker channel")
	}
	return p, nil
}

// clean disables expired short URLs.
func clean(c *conf.Config) error {
	var change int
	s, err := db.NewSession(c.Conn, false)
	if err != nil {
		return err
	}
	defer s.Close()
	coll, err := db.Coll(s, "urls")
	if err != nil {
		return err
	}
	condition := bson.D{
		{"off", false},
		{"ttl", bson.M{"$lt": time.Now().UTC()}},
	}
	update := bson.M{"$set": bson.M{"off": true}}
	cache, cacheOn := c.Cache.Strorage["URL"]
	if cacheOn {
		cu := &trim.CustomURL{}
		iter := coll.Find(condition).Iter()
		for iter.Next(cu) {
			if err := coll.UpdateId(cu.ID, update); err == nil {
				cache.Remove(cu.String())
				change++
			}
		}
		err = iter.Close()
		if err != nil {
			return err
		}
	} else {
		// cache is disable, update only URLs
		info, err := coll.UpdateAll(condition, update)
		if err != nil {
			return err
		}
		change = info.Updated
	}
	c.L.Debug.Printf("cleaned %v item(s)", change)
	return nil
}

// CleanWorker deactivates expired short URLs periodically every 5 minutes.
func CleanWorker(ctx context.Context) {
	var err error
	c, _ := conf.FromContext(ctx)
	tick := time.Tick(time.Duration(1 * time.Minute))
	for range tick {
		err = clean(c)
		if err != nil {
			c.L.Error.Printf("clean error: %v", err)
		}
	}
}

// validateParams checks HTTP parameters.
func validateParams(r *http.Request) (*trim.ReqParams, error) {
	var (
		nd  bool
		ttl *time.Time
	)
	if v := r.PostFormValue("ttl"); v != "" {
		t, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		expire := time.Now().Add(time.Duration(t) * time.Hour).UTC()
		ttl = &expire
	}
	if v := r.PostFormValue("nd"); v != "" {
		nd = true
	}
	params := &trim.ReqParams{
		Original:  r.PostFormValue("url"),
		Tag:       r.PostFormValue("tag"),
		NotDirect: nd,
		TTL:       ttl,
	}
	err := params.Valid()
	if err != nil {
		return nil, err
	}
	return params, nil
}

// TrimAddress returns URL path.
func TrimAddress(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	return u.EscapedPath(), nil
}

// HandlerTest handles test GET request.
func HandlerTest(ctx context.Context, w http.ResponseWriter, r *http.Request) ErrHandler {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	coll, err := db.C(ctx, "tests")
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	command := r.FormValue("write")
	switch {
	case c.Debug && command == "add":
		err = coll.Insert(bson.M{"ts": time.Now()})
	case c.Debug && command == "del":
		err = coll.Remove(nil)
	}
	if err != nil && err != mgo.ErrNotFound {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	n, err := coll.Count()
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	u, err := auth.ExtractUser(ctx)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	c.L.Debug.Printf("user=%v", u)
	fmt.Fprintf(w, "found %v items", n)
	return ErrHandler{nil, http.StatusOK}
}

// HandlerRedirect searches saved original URL by a short one.
func HandlerRedirect(ctx context.Context, short string, r *http.Request) (string, error) {
	cu, err := trim.Lengthen(ctx, short)
	if err != nil {
		return "", err
	}
	ch, err := TrackerChan(ctx)
	if err != nil {
		logger.Println(err)
	} else {
		cui := &CuInfo{ctx, cu, r.RemoteAddr}
		ch <- cui
	}
	// TODO: check direct redirect
	return cu.Original, nil
}

// HandlerIndex returns index web page.
func HandlerIndex(ctx context.Context, w http.ResponseWriter, r *http.Request) ErrHandler {
	data := map[string]string{}
	c, err := conf.FromContext(ctx)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	tpl, err := c.CacheTpl("base", "base.html", "index.html")
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	if r.Method == "POST" {
		p, err := validateParams(r)
		if err != nil {
			c.L.Error.Println(err)
			data["Error"] = "Invalid data."
			err = tpl.ExecuteTemplate(w, "base", data)
			if err != nil {
				return ErrHandler{err, http.StatusInternalServerError}
			}
			return ErrHandler{nil, http.StatusOK}
		}
		params := []*trim.ReqParams{p}
		cus, err := trim.Shorten(ctx, params)
		if err != nil {
			return ErrHandler{err, http.StatusInternalServerError}
		}
		data["Result"] = c.Address(cus[0].String())
	}
	err = tpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	return ErrHandler{nil, http.StatusOK}
}

// HandlerNotFound returns "not found" web page.
func HandlerNotFound(ctx context.Context, w http.ResponseWriter, r *http.Request) ErrHandler {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	tpl, err := c.CacheTpl("error", "base.html", "error.html")
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	data := map[string]string{
		"Message": "The page is not found.",
		"Error":   "",
	}
	err = tpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	return ErrHandler{nil, http.StatusOK}
}

// HandlerError returns "error" web page.
func HandlerError(ctx context.Context, w http.ResponseWriter, r *http.Request) ErrHandler {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	tpl, err := c.CacheTpl("error", "base.html", "error.html")
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	data := map[string]string{
		"Message": "Error",
		"Error":   "The error occurred, probably due to internal server problems.",
	}
	err = tpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		return ErrHandler{err, http.StatusInternalServerError}
	}
	return ErrHandler{nil, http.StatusOK}
}
