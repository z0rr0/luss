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

// HandlerAddJSON creates and saves new short links from JSON format.
func HandlerAddJSON(w http.ResponseWriter, r *http.Request) (int, string) {
    if r.Method != "POST" {
        return http.StatusMethodNotAllowed, "method not allowed"
    }
    t := r.Header["Content-Type"]
    fmt.Println(len(t), t)
    fmt.Fprintf(w, "data=%v", "ok")
    return http.StatusOK, ""
}

// HandlerAddLink creates and saves new short link.
func HandlerAddLink(w http.ResponseWriter, r *http.Request) (int, string) {
    if r.Method != "POST" {
        return http.StatusMethodNotAllowed, "method not allowed"
    }
    reqf, err := trim.CheckReqForm(r, utils.Cfg.Conf)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusBadRequest, "bad request"
    }
    url, err := url.ParseRequestURI(reqf.Original)
    if err != nil {
        return http.StatusBadRequest, "bad request - invalid URL"
    }
    // ascii raw URL
    host, err := idna.ToASCII(url.Host)
    if err != nil {
        return http.StatusBadRequest, "bad request - bad domain"
    }
    url.Host = host
    reqf.Original = url.String()
    // incomming URL is Ok, try to trim and save
    short, err := trim.GetShort(reqf, utils.Cfg.Conf)
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
