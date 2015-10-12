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

// urlReqJSON contains parameter of user's request.
type urlReqJSON struct {
    Original string `json:"url"`
    TTL      int    `json:"ttl"`
    Param    string `json:"p"`
}

// urlRespJSON is a response for an user's URL.
type urlRespJSON struct {
    Original string `json:"o"`
    Short    string `json:"s"`
}

// ReqJSON is structure of user's JSON request.
type ReqJSON struct {
    Token string       `json:"token"`
    Tag   string       `json:"tag"`
    URLs  []urlReqJSON `json:"urls"`
}

// RespJSON is structure of JSON response.
type RespJSON struct {
    User string        `json:"user"`
    URLs []urlRespJSON `json:"urls"`
    N    int           `json:"n"`
}

// UserRequestMeta is additional info for user's request.
type UserRequestMeta struct {
    User      *conf.User
    Project   *conf.Project
    Tag       string
    Anonymous bool
}

// UserRequest is structure of verified user's request data.
type UserRequest struct {
    URL   *url.URL
    Param string
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

// VerifyUserRequest validates user's data.
func VerifyUserRequest(token, tag string, c *conf.Config) (*UserRequestMeta, error) {
    var isAnonymous bool
    // check user credentials (it can be anonymous)
    p, u, err := prj.CheckUser(token, c)
    if err != nil {
        return nil, err
    }
    // is it anonymous request?
    if p == &conf.AnonProject {
        isAnonymous, tag = true, ""
    }
    meta := &UserRequestMeta{
        User:      u,
        Project:   p,
        Tag:       tag,
        Anonymous: isAnonymous,
    }
    return meta, nil
}

// ParseJSONRequest parses HTTP JSON pack request.
func ParseJSONRequest(r *http.Request, c *conf.Config) (*UserRequestMeta, []UserRequest, error) {
    var (
        ttl   *time.Time
        param string
    )
    decoder := json.NewDecoder(r.Body)
    req := &ReqJSON{}
    err := decoder.Decode(req)
    if (err != nil) && (err != io.EOF) {
        return nil, nil, err
    }
    n := len(req.URLs)
    if n == 0 {
        return nil, nil, errors.New("empty user request")
    }
    meta, err := VerifyUserRequest(req.Token, req.Tag, c)
    if err != nil {
        return nil, nil, err
    }
    uReqs := make([]UserRequest, n)
    for i := range req.URLs {
        url, err := utils.ParseURL(req.URLs[i].Original)
        if err != nil {
            return nil, nil, err
        }
        if req.URLs[i].TTL > 0 {
            expired := time.Now().Add(time.Duration(req.URLs[i].TTL) * time.Hour)
            ttl = &expired
        } else {
            ttl = nil
        }
        if meta.Anonymous {
            param = ""
        } else {
            param = req.URLs[i].Param
        }
        uReqs[i] = UserRequest{
            URL:   url,
            TTL:   ttl,
            Param: param,
        }
    }
    return meta, uReqs, nil
}

// ParseLinkRequest parses HTTP request and returns RequestForm pointer.
func ParseLinkRequest(r *http.Request, c *conf.Config) (*UserRequestMeta, []UserRequest, error) {
    var (
        ttl   *time.Time
        param string
    )
    rawurl := r.PostFormValue("url")
    if rawurl == "" {
        return nil, nil, errors.New("url parameter not found")
    }
    meta, err := VerifyUserRequest(r.PostFormValue("token"), r.PostFormValue("tag"), c)
    if err != nil {
        return nil, nil, err
    }
    url, err := utils.ParseURL(rawurl)
    if err != nil {
        return nil, nil, err
    }
    if t := r.PostFormValue("ttl"); t != "" {
        ti, err := strconv.Atoi(t)
        if err != nil {
            return nil, nil, err
        }
        expired := time.Now().Add(time.Duration(ti) * time.Hour)
        ttl = &expired
    }
    if meta.Anonymous {
        param = ""
    } else {
        param = r.PostFormValue("p")
    }
    uReqs := []UserRequest{
        UserRequest{
            URL:   url,
            TTL:   ttl,
            Param: param,
        },
    }
    return meta, uReqs, nil
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
    meta, uReqs, err := ParseJSONRequest(r, utils.Cfg.Conf)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusBadRequest, "bad request"
    }
    n := len(uReqs)
    resp := &RespJSON{
        User: meta.User.Name,
        URLs: make([]urlRespJSON, n),
        N:    n,
    }
    cus := make([]*trim.CustomURL, n)
    for i := range uReqs {
        now := time.Now().UTC()
        cu := &trim.CustomURL{
            // Short:   "",
            Project:   meta.Project.Name,
            User:      meta.User.Name,
            Tag:       meta.Tag,
            Original:  uReqs[i].URL.String(),
            TTL:       uReqs[i].TTL,
            NotDirect: false,
            Active:    true,
            Spam:      0,
            Created:   now,
            Modified:  now,
        }
        cus[i] = cu
    }
    err = trim.GetShort(utils.Cfg.Conf, cus...)
    // err = trim.GetShort(utils.Cfg.Conf, cus...)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusInternalServerError, "internal error"
    }
    for i := range uReqs {
        resp.URLs[i].Original = cus[i].Original
        resp.URLs[i].Short = utils.Cfg.Conf.Domain.Address + cus[i].Short
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
    meta, uReqs, err := ParseLinkRequest(r, utils.Cfg.Conf)
    if err != nil {
        utils.LoggerError.Println(err)
        return http.StatusBadRequest, "bad request"
    }
    now := time.Now().UTC()
    cu := &trim.CustomURL{
        Project:   meta.Project.Name,
        User:      meta.User.Name,
        Tag:       meta.Tag,
        Original:  uReqs[0].URL.String(),
        TTL:       uReqs[0].TTL,
        Active:    true,
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
    fmt.Fprintf(w, utils.Cfg.Conf.Domain.Address+cu.Short)
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
