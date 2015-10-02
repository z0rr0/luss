// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package httph contains main HTTP handlers.
package httph

import (
    "fmt"
    "net/http"
    "net/url"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "github.com/z0rr0/luss/utils"
    "golang.org/x/net/idna"
    "gopkg.in/mgo.v2/bson"
)

func checkAddLinkForm(r *http.Request) (string, error) {
    err := r.ParseForm() // 10 MB
    if err != nil {
        return "", err
    }
    r.ParseMultipartForm(32 << 10) // 32 MB
    return r.PostFormValue("url"), nil
}

// TestWrite writes temporary data to the database.
func TestWrite(c *conf.Config) error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    coll := conn.C(db.Colls["test"])
    return coll.Insert(bson.M{"ts": time.Now()})
}

func HandlerAddLink(w http.ResponseWriter, r *http.Request) (int, string) {
    if r.Method != "POST" {
        return http.StatusMethodNotAllowed, "method not allowed"
    }
    raw, err := checkAddLinkForm(r)
    if (err != nil) || (raw == "") {
        utils.LoggerDebug.Println(err)
        return http.StatusBadRequest, "bad request"
    }
    url, err := url.ParseRequestURI(raw)
    if err != nil {
        utils.LoggerDebug.Println(err)
        return http.StatusBadRequest, "bad request - invalid URL"
    }
    // ascii raw URL
    host, err := idna.ToASCII(url.Host)
    if err != nil {
        utils.LoggerDebug.Println(err)
        return http.StatusBadRequest, "bad request - bad domain"
    }
    url.Host = host
    utils.LoggerDebug.Printf("passed %v", url)
    fmt.Fprintln(w, url.String())
    return http.StatusOK, ""
}
