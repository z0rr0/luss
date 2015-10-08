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
    "strconv"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "github.com/z0rr0/luss/prj"
    "github.com/z0rr0/luss/trim"
    "github.com/z0rr0/luss/utils"
    "gopkg.in/mgo.v2/bson"
)

// ReqJSON is structure of user's JSON request.
type ReqJSON struct {
    Original string `json:"url"`
    Project  string `json:"project"`
    TTL      int    `json:"ttl"`
}

// UserRawRequest is structure of raw user's request data.
type UserRawRequest struct {
    Token string
    Param string
    URL   *url.URL
    TTL   *time.Time
}

// UserRequest is structure of verified user's request data.
type UserRequest struct {
    User    *prj.User
    Project *prj.Project
    URL     *url.URL
    Param   string
    TTL     *time.Time
}

func VerifyUserRawRequests(reqs []UserRawRequest, c *conf.Config) ([]UserRequest, error) {
    if len(reqs) == 0 {
        return nil, errors.New("empty user request")
    }
    // TODO: check anonymous here
    // check token
    p, u, err := prj.CheckUser(reqs[0].Token, c)
    if err != nil {
        return nil, err
    }
    result := make([]UserRequest, len(reqs))
    for i := range reqs {
        result[i] = UserRequest{
            User:    u,
            Project: p,
            URL:     reqs[i].URL,
            TTL:     reqs[i].TTL,
            Param:   reqs[i].Param,
        }
    }
    return result, nil
}

// TODO: ParseJSONRequest...

// ParseLinkRequest parses HTTP request and returns RequestForm pointer.
func ParseLinkRequest(r *http.Request, c *conf.Config) ([]UserRequest, error) {
    var ttl *time.Time
    rawurl := r.PostFormValue("url")
    if rawurl == "" {
        return nil, errors.New("url parameter not found")
    }
    url, err := utils.ParseURL(rawurl)
    if err != nil {
        return nil, err
    }
    if t := r.PostFormValue("ttl"); t != "" {
        ti, err := strconv.Atoi(t)
        if err != nil {
            return nil, err
        }
        expired := time.Now().Add(time.Duration(ti) * time.Hour)
        ttl = &expired
    }
    uReqs := []UserRawRequest{UserRawRequest{URL: url, Token: r.PostFormValue("token"), TTL: ttl, Param: r.PostFormValue("cbp")}}
    return VerifyUserRawRequests(uReqs, c)
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
    uReqs, err := ParseLinkRequest(r, utils.Cfg.Conf)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusBadRequest, "bad request"
    }
    uReq, now := uReqs[0], time.Now().UTC()
    cu := &trim.CustomURL{
        // Short:   "",
        Active:    true,
        Project:   uReq.Project.Name,
        Original:  uReq.URL.String(),
        User:      uReq.User.Name,
        TTL:       uReq.TTL,
        NotDirect: false,
        Spam:      0,
        Created:   now,
        Modified:  now,
    }
    err = trim.GetShort(utils.Cfg.Conf, cu)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusInternalServerError, "internal error"
    }
    // log
    utils.LoggerDebug.Printf("passed [%v] => [%v]", cu.Original, cu.Short)
    fmt.Fprintf(w, cu.Short)
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
    // write to buffered channel: stats + callback
    // handler method should be limited by goroutines numbers
    utils.Cfg.Conf.Workers.ChStats <- cu.Short
    return cu.Original, nil
}
