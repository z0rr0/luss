// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package core contains main internal methods.
package core

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/project"
	"github.com/z0rr0/luss/trim"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// HandlerTest handles test GET request.
func HandlerTest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return err
	}
	coll, err := db.C(ctx, "test")
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
	u, err := project.ExtractUser(ctx)
	if err != nil {
		return err
	}
	p, err := project.ExtractProject(ctx)
	if err != nil {
		return err
	}
	c.L.Debug.Printf("user=%v, project=%v", u, p)
	fmt.Fprintf(w, "found %v items", n)
	return nil
}

// HandlerRedirect searches saved original URL by a short one.
func HandlerRedirect(ctx context.Context, short string) (string, error) {
	cu, err := trim.Lengthen(ctx, short)
	if err != nil {
		return "", err
	}
	// TODO: check direct redirect
	// TODO: add callback handler call
	// TODO: add tracker actions
	return cu.Original, nil
}

// HandlerAdd creates and returns new short URL.
// It expects POST request with parameters: {url(string),ttl(uint64),tag(string),nd(bool)}.
func HandlerAdd(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var (
		nd     bool
		tag    string
		err    error
		rawURL string
		u      *url.URL
		ttl    *time.Time
	)
	c, err := conf.FromContext(ctx)
	if err != nil {
		return err
	}
	rawURL = r.PostFormValue("url")
	if rawURL == "" {
		return errors.New("empty URL request parameter")
	}
	// it is only to validate url value and escaping
	u, err = url.ParseRequestURI(rawURL)
	if err != nil {
		return err
	}
	tag = r.PostFormValue("tag")
	if p := r.PostFormValue("ttl"); p != "" {
		t, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return err
		}
		expire := time.Now().Add(time.Duration(t) * time.Hour).UTC()
		ttl = &expire
	}
	if p := r.PostFormValue("nd"); p != "" {
		nd = true
	}
	params := []trim.ReqParams{
		trim.ReqParams{
			Original:  u.String(),
			Tag:       tag,
			NotDirect: nd,
			TTL:       ttl,
		},
	}
	cus, err := trim.Shorten(ctx, params)
	if err != nil {
		return err
	}
	if len(cus) < 1 {
		return errors.New("empty shorten result")
	}
	fmt.Fprintf(w, "%v", c.Address(cus[0].String()))
	return nil
}
