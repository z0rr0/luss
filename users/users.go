// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package users implements users handling methods.
package users

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

const (
    // dbKeyLen is the size, in bytes, of a SHA3 Shake256 checksum.
    dbKeyLen = 64
    // maxUserNameLen is maximum length of user name
    maxUserNameLen = 256
)

var (
    // Logger is a logger of important messages.
    Logger = log.New(os.Stderr, "LOGGER [luss/users]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// DbKey is a key info about validate app secrete keys.
type DbKey struct {
    ID      bson.ObjectId `bson:"_id"`
    Value   string        `bson:"value"`
    Created time.Time     `bson:"created"`
}

// User is user's information.
type User struct {
    ID        bson.ObjectId `bson:"_id"`
    Name      string        `bson:"name"`
    Token     string        `bson:"token"`
    Role      string        `bson:"role"`
    Created   time.Time     `bson:"created"`
    Anonymous bool
    Secret    string
}

// Refresh reads user info from database using a filter by the name.
func (u *User) Refresh(c *conf.Config) error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    collection := conn.C(db.Colls["users"])
    return collection.Find(bson.M{"name": u.Name}).One(u)
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

// GenRndBytes generates random bytes.
func GenRndBytes(n int) ([]byte, error) {
    b := make([]byte, n)
    _, err := rand.Read(b)
    return b, err
}

// DeleteUser deletes user by his name.
func DeleteUser(name string, c *conf.Config) error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    collection := conn.C(db.Colls["users"])
    return collection.Remove(bson.M{"name": name})
}

// CreateUser creates new User.
func CreateUser(name, role string, c *conf.Config) (*User, error) {
    if len(name) > maxUserNameLen {
        return nil, errors.New("too long user's name")
    }
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return nil, err
    }
    collection := conn.C(db.Colls["users"])
    if role == "" {
        role = "user"
    }
    p1, p2, tErr := genToken(c)
    if tErr != nil {
        return nil, tErr
    }
    err = collection.Insert(bson.M{
        "name":    name,
        "role":    role,
        "token":   p2,
        "created": time.Now().UTC(),
    })
    if err != nil {
        return nil, err
    }
    u := &User{Name: name}
    err = u.Refresh(c)
    u.Secret = p1 + p2
    return u, err
}

// CheckUser verifies a token and returns the appropriate User.
func CheckUser(token string, c *conf.Config) (*User, error) {
    t, err := CheckToken(token, c)
    if err != nil {
        return nil, err
    }
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return nil, err
    }
    collection := conn.C(db.Colls["users"])
    u := &User{}
    err = collection.Find(bson.M{"token": t}).One(u)
    if err == mgo.ErrNotFound {
        u.Anonymous = true
        err = nil
    }
    return u, err
}

// genToken generates new user's token.
// It looks as trapdoor function: token=R+Hash(R+S), where S is a secret salt.
// This method is not very secure, but it's quick.
func genToken(c *conf.Config) (string, string, error) {
    r, err := GenRndBytes(c.Listener.Security.TokenLen)
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

// CheckToken verifies incoming token, checks length and hash.
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
