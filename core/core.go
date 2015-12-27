// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package core contains main internal methods.
package core

import (
	"fmt"
	"net/http"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/project"
	"golang.org/x/net/context"
)

// HandlerTest handles test GET request.
func HandlerTest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return err
	}
	coll, err := db.C(ctx, "test")
	if err != nil {
		c.L.Error.Println(err)
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
		c.L.Error.Println(err)
		return err
	}
	n, err := coll.Count()
	if err != nil {
		c.L.Error.Println(err)
		return err
	}
	u, err := project.ExtractUser(ctx)
	if err != nil {
		c.L.Error.Println(err)
		return err
	}
	p, err := project.ExtractProject(ctx)
	if err != nil {
		c.L.Error.Println(err)
		return err
	}
	c.L.Debug.Printf("user=%v, project=%v", u, p)
	fmt.Fprintf(w, "found %v items", n)
	return nil
}