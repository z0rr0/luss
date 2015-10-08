// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package prj implements methods to handler projects/users activities.
package prj

import (
    "crypto/rand"
    "encoding/hex"
    "errors"
    "log"
    "os"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "golang.org/x/crypto/sha3"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

var (
    // Logger is a logger of important messages.
    Logger = log.New(os.Stderr, "LOGGER [luss/prj]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// getRndBytes generates random bytes.
func getRndBytes(n int) ([]byte, error) {
    b := make([]byte, n)
    _, err := rand.Read(b)
    return b, err
}

// genToken generates new user's token.
// It looks as trapdoor function: token=R+Hash(R+S), where S is a secret salt.
// This method is not very secure, but it's quick.
func genToken(c *conf.Config) (string, string, error) {
    r, err := getRndBytes(c.Listener.Security.TokenLen)
    if err != nil {
        return "", "", err
    }
    h := make([]byte, c.Listener.Security.TokenLen)
    d := sha3.NewShake256()
    d.Write([]byte(c.Listener.Security.Salt))
    d.Write(r)
    d.Read(h)
    // token=rnd[32]+hash(rnd+salt)[32]
    return hex.EncodeToString(r), hex.EncodeToString(h), nil
}

// EqualBytes compares two byte slices. It is crypto-safe, because
// successful and unsuccessful attempts have around a same duration time.
func EqualBytes(x, y []byte) bool {
    result := true
    mi, ma := x, y
    if len(x) > len(y) {
        mi, ma = y, x
        result = false
    }
    for i, v := range mi {
        if ma[i] != v {
            result = false
        }
    }
    return result
}

// CheckToken verifies incoming token, checks length and hash.
// It returns the 2nd (stored in DB) token part and error value.
func CheckToken(token string, c *conf.Config) (string, error) {
    l := len(token)
    if l == 0 {
        return "", errors.New("empty token value")
    }
    fullToken, err := hex.DecodeString(token)
    if err != nil {
        return "", err
    }
    n := len(fullToken)
    h := make([]byte, n/2)
    d := sha3.NewShake256()
    d.Write([]byte(c.Listener.Security.Salt))
    d.Write(fullToken[:n/2])
    d.Read(h)
    // don't use bytes.Equal here, because
    // timing attack can be applicable for this method.
    if !EqualBytes(h, fullToken[n/2:n]) {
        return "", errors.New("invalid token")
    }
    return token[l/2 : l], nil
}

// CheckUser verifies a token and returns the appropriate User.
func CheckUser(token string, c *conf.Config) (*conf.Project, *conf.User, error) {
    var u *conf.User
    if (token == "") && c.Projects.Anonymous {
        return &conf.AnonProject, &conf.AnonUser, nil
    }
    t, err := CheckToken(token, c)
    if err != nil {
        return nil, nil, err
    }
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return nil, nil, err
    }
    coll := conn.C(db.Colls["projects"])
    p := &conf.Project{}
    err = coll.Find(bson.M{"users.key": t}).One(p)
    if err != nil {
        return nil, nil, err
    }
    for i := range p.Users {
        if p.Users[i].Key == t {
            u = &p.Users[i]
            break
        }
    }
    if u == nil {
        err = errors.New("user is not found")
    }
    return p, u, err
}

// CreateProject creates new project.
func CreateProject(p *conf.Project, c *conf.Config) error {
    // TODO: check project name length
    var (
        p1, p2 string
        keyErr error
    )
    if len(p.Name) > c.Projects.MaxName {
        return errors.New("too long project name")
    }
    // p.Users has empty key-fields
    now := time.Now().UTC()
    secrets := make([]string, len(p.Users))
    for i := range p.Users {
        p1, p2, keyErr = genToken(c)
        if keyErr != nil {
            return keyErr
        }
        p.Users[i].Key = p2
        p.Users[i].Created = now
        p.Users[i].Secret = ""
        secrets[i] = p1 + p2
    }
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    coll := conn.C(db.Colls["projects"])
    err = coll.Insert(p)
    if mgo.IsDup(err) {
        return db.ErrDbDuplicate
    }
    for i := range p.Users {
        p.Users[i].Secret = secrets[i]
    }
    return err
}

// DeleteProject removes project and its links if it's needed.
func DeleteProject(name string, c *conf.Config, force bool) error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    coll := conn.C(db.Colls["projects"])
    err = coll.Remove(bson.M{"name": name})
    if err != nil {
        return err
    }
    if force {
        // deactivate all project's links
        c.Clean <- name
    }
    return nil
}

// UpdateProject updates project's users info, resets all their credentials.
func UpdateProject(name string, users []conf.User, c *conf.Config) error {
    var (
        p1, p2 string
        keyErr error
    )
    now := time.Now().UTC()
    secrets := make([]string, len(users))
    for i := range users {
        p1, p2, keyErr = genToken(c)
        if keyErr != nil {
            return keyErr
        }
        users[i].Key = p2
        users[i].Created = now
        users[i].Secret = ""
        secrets[i] = p1 + p2
    }
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    coll := conn.C(db.Colls["projects"])
    err = coll.Update(bson.M{"name": name}, bson.M{"$set": bson.M{"users": users}})
    if err != nil {
        return err
    }
    for i := range users {
        users[i].Secret = secrets[i]
    }
    return nil
}

// CreateAdmin creates new global admin user.
func CreateAdmin(name string, c *conf.Config) (*conf.User, error) {
    p1, p2, err := genToken(c)
    if err != nil {
        return nil, err
    }
    if err != nil {
        return nil, err
    }
    u := &conf.User{Name: name, Key: p2, Role: "admin", Created: time.Now().UTC()}
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return nil, err
    }
    coll := conn.C(db.Colls["projects"])
    err = coll.Update(bson.M{"name": conf.DefaultProject, "users.name": bson.M{"$ne": name}},
        bson.M{"$push": bson.M{"users": u}, "$set": bson.M{"modified": time.Now().UTC()}})
    if err != nil {
        return nil, err
    }
    u.Secret = p1 + p2
    return u, nil
}
