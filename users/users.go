// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package users implements users handling methods.
package users

import (
    "encoding/hex"
    "fmt"

    "github.com/z0rr0/luss/conf"
    "golang.org/x/crypto/pbkdf2"
    "golang.org/x/crypto/sha3"
)

const (
    // SHA3Size is the size, in bytes, of a SHA3-512 checksum.
    SHA3Size = 64
    pwIters  = 4096
)

// PwdHash generates a password hash.
func PwdHash(username, password string, c *conf.Config) string {
    value := fmt.Sprintf("%v%v%v", username, password, c.Listener.Salts[0])
    pHash := pbkdf2.Key([]byte(value), []byte(c.Listener.Salts[1]), pwIters, SHA3Size, sha3.New512)
    return hex.EncodeToString(pHash)
}
