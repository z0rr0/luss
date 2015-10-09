// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package httph contains main HTTP handlers.
package httph

import (
    "encoding/json"
    "errors"
    "fmt"
    "io"
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

type urlReqJSON struct {
    Original string `json:"url"`
    TTL      int    `json:"ttl"`
    Param    string `json:"p"`
}

type urlRespJSON struct {
    Original string `json:"o"`
    Short    string `json:"s"`
}

// ReqJSON is structure of user's JSON request.
type ReqJSON struct {
    Token string       `json:"token"`
    URLs  []urlReqJSON `json:"urls"`
}

// ReqJSON is structure of JSON response.
type RespJSON struct {
    User string        `json:"user"`
    URLs []urlRespJSON `json:"urls"`
    N    int           `json:"n"`
}

// UserRawRequest is structure of raw user's request data.
type UserRawRequest struct {
    Param string
    URL   *url.URL
    TTL   *time.Time
}

// Marshall encodes JSON response.
func (r *RespJSON) Marshall(w http.ResponseWriter) error {
    w.Header().Set("Content-Type", "application/json")
    b, err := json.Marshal(r)
    if err != nil {
        return err
    }
    fmt.Fprintf(w, "%s", b)
    return nil
}

// UserRequest is structure of verified user's request data.
type UserRequest struct {
    User    *conf.User
    Project *conf.Project
    URL     *url.URL
    Param   string
    TTL     *time.Time
}

// VerifyUserRawRequests validates user's data.
func VerifyUserRawRequests(token string, reqs []UserRawRequest, c *conf.Config) ([]UserRequest, error) {
    if len(reqs) == 0 {
        return nil, errors.New("empty user request")
    }
    // check user credentials (it can be anonymous)
    p, u, err := prj.CheckUser(token, c)
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

// ParseJSONRequest parses HTTP JSON pack request.
func ParseJSONRequest(r *http.Request, c *conf.Config) ([]UserRequest, error) {
    var ttl *time.Time
    decoder := json.NewDecoder(r.Body)
    req := &ReqJSON{}
    err := decoder.Decode(req)
    if (err != nil) && (err != io.EOF) {
        return nil, err
    }
    uReqs := make([]UserRawRequest, len(req.URLs))
    for i := range req.URLs {
        url, err := utils.ParseURL(req.URLs[i].Original)
        if err != nil {
            return nil, err
        }
        if req.URLs[i].TTL > 0 {
            expired := time.Now().Add(time.Duration(req.URLs[i].TTL) * time.Hour)
            ttl = &expired
        } else {
            ttl = nil
        }
        uReqs[i] = UserRawRequest{
            URL:   url,
            Param: req.URLs[i].Param,
            TTL:   ttl,
        }
    }
    return VerifyUserRawRequests(req.Token, uReqs, c)
}

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
    uReqs := []UserRawRequest{
        UserRawRequest{
            URL:   url,
            TTL:   ttl,
            Param: r.PostFormValue("p"),
        },
    }
    return VerifyUserRawRequests(r.PostFormValue("token"), uReqs, c)
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
    uReqs, err := ParseJSONRequest(r, utils.Cfg.Conf)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusBadRequest, "bad request"
    }
    n := len(uReqs)
    resp := &RespJSON{
        User: uReqs[0].User.Name,
        URLs: make([]urlRespJSON, n),
        N:    n,
    }
    cus := make([]*trim.CustomURL, n)
    for i := range uReqs {
        now := time.Now().UTC()
        cu := &trim.CustomURL{
            // Short:   "",
            Active:    true,
            Project:   uReqs[i].Project.Name,
            Original:  uReqs[i].URL.String(),
            User:      uReqs[i].User.Name,
            TTL:       uReqs[i].TTL,
            NotDirect: false,
            Spam:      0,
            Created:   now,
            Modified:  now,
        }
        cus[i] = cu
    }
    err = trim.GetShort(utils.Cfg.Conf, cus...)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusInternalServerError, "internal error"
    }
    for i := range uReqs {
        resp.URLs[i].Original = cus[i].Original
        resp.URLs[i].Short = cus[i].Short
    }
    err = resp.Marshall(w)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusInternalServerError, "internal error"
    }
    utils.LoggerDebug.Printf("passed [%v] JSON items", n)
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
    now := time.Now().UTC()
    cu := &trim.CustomURL{
        // Short:   "",
        Active:    true,
        Project:   uReqs[0].Project.Name,
        Original:  uReqs[0].URL.String(),
        User:      uReqs[0].User.Name,
        TTL:       uReqs[0].TTL,
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
