// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package core contains main internal methods.
package core

import (
	"fmt"
	"net/http"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"golang.org/x/net/context"
)

// HandlerTest handles test GET request.
func HandlerTest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return err
	}
	s, err := db.NewSession(c.Conn, true)
	if err != nil {
		c.L.Error.Println(err)
		return err
	}
	defer s.Close()
	ctx = db.NewContext(ctx, s)
	coll, err := db.C(ctx, "test")
	if err != nil {
		c.L.Error.Println(err)
		return err
	}
	query := r.URL.Query()
	if c.Debug && query.Get("write") == "yes" {
		err = coll.Insert(bson.M{"ts": time.Now()})
	}
	if err != nil {
		c.L.Error.Println(err)
		return err
	}
	n, err := coll.Count()
	if err != nil {
		c.L.Error.Println(err)
		return err
	}
	fmt.Fprintf(w, "found %v items", n)
	return nil
}
