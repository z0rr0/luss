// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package httph contains main HTTP handlers.
package httph

import (
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "github.com/z0rr0/luss/trim"
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

// HandlerAddLink adds returns a new save short link.
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
    // TODO: authentication
    project, user := "default", "anonymous"
    // ascii raw URL
    host, err := idna.ToASCII(url.Host)
    if err != nil {
        utils.LoggerDebug.Println(err)
        return http.StatusBadRequest, "bad request - bad domain"
    }
    url.Host = host
    // incomming URL is Ok, try to trim and save
    short, err := trim.GetShort(url.String(), user, project, nil, utils.Cfg.Conf)
    if err != nil {
        return http.StatusInternalServerError, "internal error"
    }
    // log
    utils.LoggerDebug.Println("passed:", short.String())
    fmt.Fprintln(w, short.Short)
    return http.StatusOK, ""
}

// HandlerRedirect searches already save short link and returns it.
func HandlerRedirect(short string, r *http.Request) (string, error) {
    if r.Method != "GET" {
        return "", errors.New("method not allowed")
    }
    cu, err := trim.FindShort(short, utils.Cfg.Conf)
    if err != nil {
        utils.LoggerDebug.Printf("invalid short link: %v", short)
        return "", err
    }
    go cu.Stat(utils.Cfg.Conf)
    return cu.Original, nil
}
