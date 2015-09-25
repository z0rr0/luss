// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package users implements users handling methods.
package users

import (
    "encoding/hex"
    // "fmt"
    // "math/rand"
    // "time"

    "github.com/z0rr0/luss/conf"
    // "golang.org/x/crypto/pbkdf2"
    "golang.org/x/crypto/sha3"
)

const (
    // // Size512 is the size, in bytes, of a SHA3-512 checksum.
    // Size512 = 64
    // Size256 = 32

    // tokenLen is size of token in bytes.
    tokenLen = 32
    pwIters  = 4096
)

// TokenGen is 24+tokenLen
func TokenGen(username string, c *conf.Config) string {
    h := make([]byte, tokenLen)
    d := sha3.NewShake256()
    d.Write([]byte(c.Listener.Salts[0]))
    d.Write([]byte(username))
    d.Read(h)
    return hex.EncodeToString(h)
}

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
