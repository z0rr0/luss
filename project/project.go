// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package project implements methods to handler projects/users activities.
package project

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2/bson"
)

const (
	userKey    key = 1
	projectKey key = 2
)

var (
	// AnonUser is anonymous user.
	AnonUser = &User{Name: "anonymous"}
	// AnonProject is anonymous project.
	AnonProject = &Project{Name: "anonymous", Users: []User{*AnonUser}}
)

type key int

// User is structure of user's info.
type User struct {
	Name   string    `bson:"name"`
	Key    string    `bson:"key"`
	Role   string    `bson:"role"`
	Ts     time.Time `bson:"ts"`
	Secret string    `bson:",omitempty"`
}

// Project is structure of project's info.
type Project struct {
	ID    bson.ObjectId `bson:"_id"`
	Name  string        `bson:"name"`
	Users []User        `bson:"users"`
	Ts    time.Time     `bson:"ts"`
}

// String returns user's name
func (u *User) String() string {
	return u.Name
}

// String returns project's name
func (p *Project) String() string {
	return p.Name
}

// genToken generates new user's token and
// returns random hex number and its hash.
// It looks as trapdoor function: token=R+Hash(R+S), where S is a secret salt.
// This method is not very secure, but it works quite quickly.
func genToken(c *conf.Config) (string, string, error) {
	b := make([]byte, c.Listener.Security.TokenLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", "", err
	}
	h := make([]byte, c.Listener.Security.TokenLen)
	d := sha3.NewShake256()
	d.Write([]byte(c.Listener.Security.Salt))
	d.Write(b)
	d.Read(h)
	// token=rnd[TokenLen]+hash(salt+rnd)[TokenLen]
	return hex.EncodeToString(b), hex.EncodeToString(h), nil
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

// checkToken verifies the token, checks length and hash.
// It returns the 2nd (stored in DB) token part and error value.
func checkToken(token string, c *conf.Config) (string, error) {
	l := len(token)
	if l == 0 {
		return "", errors.New("empty token value")
	}
	hexToken, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}
	n := len(hexToken)
	h := make([]byte, n/2)
	d := sha3.NewShake256()
	d.Write([]byte(c.Listener.Security.Salt))
	d.Write(hexToken[:n/2])
	d.Read(h)
	// don't use bytes.Equal here, because
	// timing attack can be applicable for this method.
	if !EqualBytes(h, hexToken[n/2:]) {
		return "", errors.New("invalid token")
	}
	return token[l/2:], nil
}

// setUserContext saves User struct to the Context.
func setUserContext(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// setUserContext saves Project struct to the Context.
func setProjectContext(ctx context.Context, p *Project) context.Context {
	return context.WithValue(ctx, projectKey, p)
}

// ExtractUser extracts user from from context.
func ExtractUser(ctx context.Context) (*User, error) {
	u, ok := ctx.Value(userKey).(*User)
	if !ok {
		return nil, errors.New("not found context user")
	}
	return u, nil
}

// ExtractProject extracts project from from context.
func ExtractProject(ctx context.Context) (*Project, error) {
	p, ok := ctx.Value(projectKey).(*Project)
	if !ok {
		return nil, errors.New("not found context project")
	}
	return p, nil
}

// Authenticate checks user's authentication.
func Authenticate(ctx context.Context, r *http.Request) (context.Context, error) {
	var u *User
	token := r.PostFormValue("token")
	if token == "" {
		// it is anonymous request
		ctx = setProjectContext(ctx, AnonProject)
		ctx = setUserContext(ctx, AnonUser)
		return ctx, nil
	}
	c, err := conf.FromContext(ctx)
	if err != nil {
		return ctx, err
	}
	t, err := checkToken(token, c)
	if err != nil {
		c.L.Error.Println(err)
		return ctx, err
	}
	// use already opened session from context
	coll, err := db.C(ctx, "projects")
	if err != nil {
		c.L.Error.Println(err)
		return ctx, err
	}
	p := &Project{}
	err = coll.Find(bson.M{"users.key": t}).One(p)
	if err != nil {
		c.L.Error.Println(err)
		return ctx, err
	}
	err = errors.New("user not found")
	for i, user := range p.Users {
		if user.Key == t {
			u, err = &p.Users[i], nil
			break
		}
	}
	if err != nil {
		c.L.Error.Println(err)
		return ctx, err
	}
	ctx = setProjectContext(ctx, p)
	ctx = setUserContext(ctx, u)
	return ctx, nil
}
