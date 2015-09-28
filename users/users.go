// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package users implements users handling methods.
package users

import (
    "bytes"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "log"
    "os"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "golang.org/x/crypto/sha3"
    // "gopkg.in/mgo.v2"
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
    ID      bson.ObjectId `bson:"_id"`
    Name    string        `bson:"name"`
    Token   string        `bson:"token"`
    Role    string        `bson:"role"`
    Created time.Time     `bson:"created"`
    Secret  string
}

// Refresh reads user info from database using a filter by the name.
func (u *User) Refresh(c *conf.Config) error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    collection := conn.C("users")
    return collection.Find(bson.M{"name": u.Name}).One(u)
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
    collection := conn.C("users")
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
    collection := conn.C("users")
    if role == "" {
        role = "user"
    }
    p1, p2, tErr := genToken(c)
    if tErr != nil {
        return nil, tErr
    }
    _, err = collection.Upsert(bson.M{"name": name},
        bson.M{
            "name":    name,
            "role":    role,
            "token":   p2,
            "created": time.Now().UTC(),
        },
    )
    if err != nil {
        return nil, err
    }
    u := &User{Name: name}
    err = u.Refresh(c)
    u.Secret = p1 + p2
    return u, err
}

// genToken generates new user's token.
// It looks as [username+password], and it is not very secrete, but quickly.
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
func CheckToken(token string, c *conf.Config) error {
    if len(token) == 0 {
        return errors.New("empty token value")
    }
    fullToken, err := hex.DecodeString(token)
    if err != nil {
        return err
    }
    n := len(fullToken)
    h := make([]byte, n/2)
    d := sha3.NewShake256()
    d.Write([]byte(c.Listener.Security.Salt))
    d.Write(fullToken[:n/2])
    d.Read(h)
    if !bytes.Equal(h, fullToken[n/2:n]) {
        return errors.New("invalid token")
    }
    return nil
}
