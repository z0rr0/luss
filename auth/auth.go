// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package auth implements methods to handler users activities.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	// Anonymous is a name of anonymous user.
	Anonymous = "anonymous"
	// Administrator is a name of administrator user.
	Administrator     = "admin"
	userKey       key = 1
	tokenKey      key = 2
)

var (
	// AnonUser is anonymous user.
	AnonUser = &User{Name: Anonymous}
	// ErrAnonymous is error of anonymous authentication
	ErrAnonymous = errors.New("anonymous request")
)

type key int

// User is structure of user's info.
type User struct {
	Name     string    `bson:"_id"`
	Disabled bool      `bson:"off"`
	Token    string    `bson:"token"`
	Roles    []string  `bson:"roles"`
	Modified time.Time `bson:"mt"`
	Created  time.Time `bson:"ct"`
}

// UserResult is result of user creation.
type UserResult struct {
	U    *User
	Name string
	T    string
	Err  string
}

// String returns user's name
func (u *User) String() string {
	return u.Name
}

// HasRole check user has a requested role.
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAnonymous returns true if user is anonymous.
func (u *User) IsAnonymous() bool {
	return u.Name == Anonymous
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
	h := tokenHash(b, c)
	return hex.EncodeToString(b), hex.EncodeToString(h), nil
}

// tokenHash calculates a SHA3 token hash.
func tokenHash(b []byte, c *conf.Config) []byte {
	h := make([]byte, c.Listener.Security.TokenLen)
	d := sha3.NewShake256()
	d.Write([]byte(c.Listener.Security.Salt))
	d.Write(b)
	d.Read(h)
	return h
}

// InitUsers initializes admin and anonymous users.
func InitUsers(c *conf.Config) error {
	s, err := db.NewSession(c.Conn, false)
	if err != nil {
		return err
	}
	defer s.Close()
	coll, err := db.Coll(s, "users")
	if err != nil {
		return err
	}
	b, err := hex.DecodeString(c.Listener.Security.Admin)
	if err != nil {
		return err
	}
	h := tokenHash(b, c)
	now := time.Now().UTC()
	users := []*User{
		&User{
			Name:     "admin",
			Disabled: false,
			Token:    hex.EncodeToString(h),
			Roles:    []string{"admin"},
			Modified: now,
			Created:  now,
		},
		&User{
			Name:     Anonymous,
			Disabled: false,
			Token:    "",
			Roles:    []string{},
			Modified: now,
			Created:  now,
		},
	}
	for _, u := range users {
		err := coll.Insert(u)
		if err != nil && !mgo.IsDup(err) {
			return err
		}
	}
	return nil
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
func CheckToken(ctx context.Context, r *http.Request, api bool) (context.Context, error) {
	var token string
	if api {
		if bearer := r.Header.Get("Authorization"); bearer != "" && !strings.HasPrefix(bearer, "Bearer") {
			return ctx, errors.New("invalid authorization data format")
		} else if bearer != "" {
			// split "Bearer"
			bearer = strings.Trim(bearer, ";")
			token = bearer[6:]
		}
	} else {
		token = r.PostFormValue("token")
	}
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
	if !EqualBytes(tokenHash(hexToken[:n/2], c), hexToken[n/2:]) {
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

// ExtractTokenKey extracts user's  from from context.
func ExtractTokenKey(ctx context.Context) (string, error) {
	t, ok := ctx.Value(tokenKey).(string)
	if !ok {
		return "", errors.New("not found context token")
	}
	return t, nil
}

// Authenticate checks user's authentication.
// It doesn't validate user's token value and doesn't detect anonymous
// request as error, so it should be identified before.
// It writes User to new context.
func Authenticate(ctx context.Context) (context.Context, error) {
	t, err := ExtractTokenKey(ctx)
	if err != nil {
		return ctx, err
	}
	if t == "" {
		// it is anonymous request
		return setUserContext(ctx, AnonUser), nil
	}
	// use already opened session from context
	coll, err := db.C(ctx, "users")
	if err != nil {
		return ctx, err
	}
	u := &User{}
	err = coll.Find(bson.M{"token": t, "off": false}).One(u)
	if err != nil {
		return ctx, err
	}
	return setUserContext(ctx, u), nil
}

// ChangeUsers updates user's tokens
func ChangeUsers(ctx context.Context, names []string) ([]UserResult, error) {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	user, err := ExtractUser(ctx)
	if err != nil {
		return nil, err
	}
	s, err := db.CtxSession(ctx)
	if err != nil {
		return nil, err
	}
	coll, err := db.Coll(s, "users")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	isAdmin := user.HasRole("admin")
	result := make([]UserResult, len(names))
	for i, name := range names {
		if !isAdmin && user.Name != name {
			result[i] = UserResult{Name: name, U: nil, Err: "permissions error"}
			continue
		}
		token, hash, err := genToken(c)
		if err != nil {
			result[i] = UserResult{Name: name, U: nil, Err: "internal error"}
			continue
		}
		err = coll.UpdateId(name, bson.M{"$set": bson.M{"token": hash, "mt": now}})
		if err != nil {
			errMsg := "internal error"
			if err == mgo.ErrNotFound {
				errMsg = "not found"
			}
			result[i] = UserResult{Name: name, U: nil, Err: errMsg}
			continue
		}
		result[i] = UserResult{Name: name, U: nil, T: token + hash, Err: ""}
	}
	return result, nil
}

// CreateUsers creates new users.
func CreateUsers(ctx context.Context, names []string) ([]UserResult, error) {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	s, err := db.CtxSession(ctx)
	if err != nil {
		return nil, err
	}
	coll, err := db.Coll(s, "users")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	result := make([]UserResult, len(names))
	for i, name := range names {
		token, hash, err := genToken(c)
		if err != nil {
			result[i] = UserResult{Name: name, U: nil, Err: "internal error"}
			continue
		}
		user := &User{
			Name:     name,
			Token:    hash,
			Roles:    []string{"user"},
			Modified: now,
			Created:  now,
		}
		if err := coll.Insert(user); err != nil {
			if mgo.IsDup(err) {
				result[i] = UserResult{Name: name, U: nil, Err: "duplicate item"}
			} else {
				result[i] = UserResult{Name: name, U: nil, Err: "internal error"}
			}
			continue
		}
		result[i] = UserResult{Name: name, U: user, T: token + hash, Err: ""}
	}
	return result, nil
}
