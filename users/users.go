// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package users implements users handling methods.
package users

import (
    "encoding/hex"
    "fmt"
    "log"
    "math/rand"
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
}

// Refresh reads user info from database using a filter by the name.
func (u *User) Refresh() error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    collection := conn.C("users")
    return collection.Find(bson.M{"name": u.Name}).One(u)
}

// createDbKey creates a new db key.
func createDbKey(col *mgo.Collection) error {
    const rlim int64 = 10000000000000
    ts := time.Now().UnixNano()
    rand.Seed(ts)
    h := make([]byte, dbKeyLen)
    sha3.ShakeSum256(h, []byte(fmt.Sprintf("%x%x", rand.Int63n(rlim), ts)))
    return col.Insert(bson.M{"value": hex.EncodeToString(h), "created": time.Now().UTC()})
}

// removeDbKeys removes all db keys. This should be used only for the tests.
func removeDbKeys(conn *db.Conn) error {
    collection := conn.C("keys")
    _, err := collection.RemoveAll(bson.M{})
    return err
}

// validateDbKeys checks that db keys exist.
func validateDbKeys(col *mgo.Collection, limit int) error {
    n, err := col.Count()
    if err != nil {
        return err
    }
    Logger.Printf("%v db keys were created", (limit - n))
    for i := 0; i < (limit - n); i++ {
        insErr := createDbKey(col)
        if insErr != nil {
            return insErr
        }
    }
    return nil
}

// GetDbKeys returns db keys map [key:value] and/or create new one if it is needed.
func GetDbKeys(c *conf.Config) error {
    conn, err := db.GetConn(c)
    if err != nil {
        return err
    }
    defer db.ReleaseConn(conn)
    collection := conn.C("keys")
    err = validateDbKeys(collection, c.Listener.Security.DbKeys)
    if err != nil {
        return err
    }
    dbkeys := []*DbKey{}
    iter := collection.Find(bson.M{}).Iter()
    err = iter.All(&dbkeys)
    if err != nil {
        return err
    }
    n := len(dbkeys)
    c.Db.DbAppKeys, c.Db.DbAppValues = make([]string, n), make(map[string]string, n)
    for i := range dbkeys {
        c.Db.DbAppKeys[i] = dbkeys[i].ID.Hex()
        c.Db.DbAppValues[c.Db.DbAppKeys[i]] = dbkeys[i].Value
    }
    return nil
}

// CreateUser creates new User's token.
func CreateUser(name, role string, c *conf.Config) (*User, error) {
    conn, err := db.GetConn(c)
    if err != nil {
        return nil, err
    }
    defer db.ReleaseConn(conn)
    collection := conn.C("users")
    if role == "" {
        role = "user"
    }
    _, err = collection.Upsert(bson.M{"name": name},
        bson.M{
            "name":    name,
            "role":    role,
            "token":   genToken(),
            "created": time.Now().UTC(),
        },
    )
    if err != nil {
        return nil, err
    }
    u := &User{}
    err = u.Refresh()
    return u, err
}

// genToken generates new user's token.
// key(24)+rnd(16)+hash()
// ToDo
func genToken(c *conf.Config) string {
    const rlim int64 = 2147483646
    ts := time.Now().UnixNano()
    rand.Seed(ts)
    idx := rand.Intn(len(c.Db.DbAppKeys))
    r := fmt.Sprintf("%x%x", ts, rand.Int63n(rlim))
    h := make([]byte, c.Listener.Security.TokenLen)

    d := sha3.NewShake256()
    d.Write([]byte(c.Db.DbAppValues[c.Db.DbAppKeys[idx]]))
    d.Write([]byte(c.Listener.Security.Salt))
    d.Write([]byte(r))
    d.Read(h)
    // key(24)+hash(c.Listener.Security.TokenLen)+rnd
    token := fmt.Sprintf("%v%v%v", c.Db.DbAppKeys[idx], hex.EncodeToString(h), r)
    return token
}

// // TokenGen is 24+tokenLen
// func TokenGen(value string, c *conf.Config) string {
//     h := make([]byte, c.Listener.Security.TokenLen)
//     d := sha3.NewShake256()
//     d.Write([]byte(c.Listener.Security.Salt))
//     d.Write([]byte(value))
//     d.Read(h)
//     return hex.EncodeToString(h)
// }

// func PwdGen(username string, c *conf.Config) string {
//     value := fmt.Sprintf("%v%v", username, c.Listener.Salts[0])
//     pHash := pbkdf2.Key([]byte(value), []byte(c.Listener.Salts[1]), pwIters, Size256, sha3.New256)
//     return hex.EncodeToString(pHash)
// }

// // PwdHash generates a password hash.
// func PwdHash(username, password string, c *conf.Config) string {
//     value := fmt.Sprintf("%v%v%v", username, password, c.Listener.Salts[0])
//     pHash := pbkdf2.Key([]byte(value), []byte(c.Listener.Salts[1]), pwIters, Size512, sha3.New512)
//     return hex.EncodeToString(pHash)
// }

// func NewToken(c *conf.Config) string {
//     rand.Seed(time.Now().UnixNano())
//     h := make([]byte, 32)
//     d := NewShake256()

// }
