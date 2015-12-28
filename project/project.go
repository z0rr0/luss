// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package project implements methods to handler projects/users activities.
package project

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2/bson"
)

const (
	// Anonymous is a name of anonymous users and projects.
	Anonymous      = "anonymous"
	userKey    key = 1
	projectKey key = 2
	tokenKey   key = 3
)

var (
	// AnonUser is anonymous user.
	AnonUser = &User{Name: Anonymous}
	// AnonProject is anonymous project.
	AnonProject = &Project{Name: Anonymous, Users: []User{*AnonUser}}
	// ErrAnonymous is error of anonymous authentication
	ErrAnonymous = errors.New("anonymous request")
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

// CheckToken verifies the token, checks length and hash value.
// If the token is valid, then its 2nd part (hash) will be added to the returned context.
// It also marks empty token as ErrAnonymous error.
func CheckToken(ctx context.Context, token string) (context.Context, error) {
	l := len(token)
	if l == 0 {
		return setTokenContext(ctx, ""), ErrAnonymous
	}
	hexToken, err := hex.DecodeString(token)
	if err != nil {
		return ctx, err
	}
	c, err := conf.FromContext(ctx)
	if err != nil {
		return ctx, err
	}
	n := len(hexToken)
	// calculate token hash
	h := make([]byte, c.Listener.Security.TokenLen)
	d := sha3.NewShake256()
	d.Write([]byte(c.Listener.Security.Salt))
	d.Write(hexToken[:n/2])
	d.Read(h)
	// don't use bytes.Equal here, because
	// timing attack can be applicable for this method.
	if !EqualBytes(h, hexToken[n/2:]) {
		return ctx, errors.New("invalid token")
	}
	// hex.EncodeToString(h) == token[l/2:]
	return setTokenContext(ctx, token[l/2:]), nil
}

// setUserContext saves User struct to the Context.
func setUserContext(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// setUserContext saves Project struct to the Context.
func setProjectContext(ctx context.Context, p *Project) context.Context {
	return context.WithValue(ctx, projectKey, p)
}

// setUserContext saves Project struct to the Context.
func setTokenContext(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
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

// ExtractTokenKey extracts user's  from from context.
func ExtractTokenKey(ctx context.Context) (string, error) {
	t, ok := ctx.Value(tokenKey).(string)
	if !ok {
		return "", errors.New("not found context token")
	}
	return t, nil
}

// Authenticate checks user's authentication.
// It doesn't validate user's token value, only identifies anonymous
// and authenticated requests and writes Project and User to new context.
func Authenticate(ctx context.Context) (context.Context, error) {
	var u *User
	t, err := ExtractTokenKey(ctx)
	if err != nil {
		return ctx, err
	}
	if t == "" {
		// it is anonymous request
		ctx = setProjectContext(ctx, AnonProject)
		ctx = setUserContext(ctx, AnonUser)
		return ctx, ErrAnonymous
	}
	c, err := conf.FromContext(ctx)
	if err != nil {
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
