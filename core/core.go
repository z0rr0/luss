// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
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

// cuInfo is trim.CustomURL info with context.
type cuInfo struct {
	ctx  context.Context
	cu   *trim.CustomURL
	addr string
}

// tracker saves info customer short URL request.
func tracker(ch <-chan *cuInfo) {
	var wg sync.WaitGroup
	for cui := range ch {
		c, err := conf.FromContext(cui.ctx)
		if err != nil {
			logger.Printf("tracker wasn't called, error: %v", err)
			continue
		}
		wg.Add(2)
		// tracker
		go func() {
			defer wg.Done()
			if err := stats.Tracker(cui.ctx, cui.cu, cui.addr); err != nil {
				c.L.Error.Println(err)
			}
		}()
		// callback handler
		go func() {
			defer wg.Done()
			if err := stats.Callback(cui.ctx, cui.cu); err != nil {
				c.L.Error.Println(err)
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
	ch := make(chan *cuInfo, trackerBuffer)
	for i := 0; i < c.Settings.Trackers; i++ {
		go func() {
			tracker(ch)
		}()
	}
	c.L.Info.Printf("run %v trackers", c.Settings.Trackers)
	return context.WithValue(ctx, trackerKey, ch), nil
}

// TrackerChan extracts tracker channel.
func TrackerChan(ctx context.Context) (chan *cuInfo, error) {
	p, ok := ctx.Value(trackerKey).(chan *cuInfo)
	if !ok {
		return nil, errors.New("not found context tracker channel")
	}
	return p, nil
}

// validateParams checks HTTP parameters.
func validateParams(r *http.Request) (trim.ReqParams, error) {
	var (
		nd     bool
		tag    string
		err    error
		rawURL string
		u      *url.URL
		ttl    *time.Time
		p      trim.ReqParams
	)
	rawURL = r.PostFormValue("url")
	if rawURL == "" {
		return p, errors.New("empty URL request parameter")
	}
	// it is only to validate url value and escaping
	u, err = url.ParseRequestURI(rawURL)
	if err != nil {
		return p, err
	}
	tag = r.PostFormValue("tag")
	if v := r.PostFormValue("ttl"); v != "" {
		t, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return p, err
		}
		expire := time.Now().Add(time.Duration(t) * time.Hour).UTC()
		ttl = &expire
	}
	if v := r.PostFormValue("nd"); v != "" {
		nd = true
	}
	params := trim.ReqParams{
		Original:  u.String(),
		Tag:       tag,
		NotDirect: nd,
		TTL:       ttl,
	}
	return params, nil
}

// HandlerTest handles test GET request.
func HandlerTest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return err
	}
	coll, err := db.C(ctx, "tests")
	if err != nil {
		return err
	}
	command := r.FormValue("write")
	switch {
	case c.Debug && command == "add":
		err = coll.Insert(bson.M{"ts": time.Now()})
	case c.Debug && command == "del":
		err = coll.Remove(nil)
	}
	if err != nil && err != mgo.ErrNotFound {
		return err
	}
	n, err := coll.Count()
	if err != nil {
		return err
	}
	u, err := auth.ExtractUser(ctx)
	if err != nil {
		return err
	}
	c.L.Debug.Printf("user=%v", u)
	fmt.Fprintf(w, "found %v items", n)
	return nil
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
		cui := &cuInfo{ctx, cu, r.RemoteAddr}
		ch <- cui
	}
	// TODO: check direct redirect
	return cu.Original, nil
}

// HandlerIndex return index web page.
func HandlerIndex(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	data := map[string]string{}
	c, err := conf.FromContext(ctx)
	if err != nil {
		return err
	}
	tpl, err := c.CacheTpl("base", "base.html", "index.html")
	if err != nil {
		return err
	}
	if r.Method == "POST" {
		p, err := validateParams(r)
		if err != nil {
			c.L.Error.Println(err)
			data["Error"] = "Invalid data."
			return tpl.ExecuteTemplate(w, "base", data)
		}
		params := []trim.ReqParams{p}
		cus, err := trim.Shorten(ctx, params)
		if err != nil {
			return err
		}
		data["Result"] = c.Address(cus[0].String())
	}
	return tpl.ExecuteTemplate(w, "base", data)
}
