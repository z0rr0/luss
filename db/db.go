// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package db implements MongoDB database access methods.
package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/z0rr0/luss/conf"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	maxLockAttempts     = 10
	lockKey             = 1
	sessionKey      key = 0
)

var (
	// Logger is a logger for error messages
	Logger = log.New(os.Stderr, "LOGGER [db]: ", log.Ldate|log.Ltime|log.Lshortfile)
	// Colls is a map of db collections names.
	// Keys can be used as aliases, values are real collection names.
	Colls = map[string]string{
		"urls":   "urls",
		"tracks": "tracks",
		"locks":  "locks",
		"users":  "users",
		"tests":  "tests",
	}
)

// key is internal type to get session value from context.
type key int

// Item is any DB item, it contains only identifier.
type Item struct {
	ID bson.ObjectId `bson:"_id"`
}

// ItemURL is any DB item, it contains only short URL identifier.
type ItemURL struct {
	ID int64 `bson:"_id"`
}

// NewContext returns a new Context carrying MongoDB session.
func NewContext(ctx context.Context, s *mgo.Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// CtxSession finds and returns MongoDB session from the Context.
func CtxSession(ctx context.Context) (*mgo.Session, error) {
	s, ok := ctx.Value(sessionKey).(*mgo.Session)
	if !ok {
		return nil, errors.New("not found context db session")
	}
	return s, nil
}

// NewSession returns new MongoDB session based on Conn data.
func NewSession(c *conf.Conn, primary bool) (*mgo.Session, error) {
	c.M.Lock()
	defer c.M.Unlock()
	switch {
	case (c.S == nil) || (c.S.Ping() != nil):
		s, err := mongoDBConnection(c.Cfg)
		if err != nil {
			return nil, err
		}
		if !primary {
			s.SetMode(mgo.SecondaryPreferred, true)
		}
		Logger.Printf("new session creation: %p", s)
		c.S = s
	case c.S.Ping() != nil:
		s, err := mongoDBConnection(c.Cfg)
		if err != nil {
			return nil, err
		}
		if !primary {
			s.SetMode(mgo.SecondaryPreferred, true)
		}
		Logger.Printf("new session recreation: %p", s)
		// old session is not valid
		// close it and use new one
		old := c.S
		c.S = s
		old.Close()
		old = nil
	}
	return c.S.Clone(), nil
}

// NewCtxSession creates new database session and saves it to the context.
func NewCtxSession(ctx context.Context, c *conf.Config, primary bool) (context.Context, *mgo.Session, error) {
	if c.Conn == nil {
		return ctx, nil, errors.New("empty main session")
	}
	s, err := NewSession(c.Conn, primary)
	if err != nil {
		return ctx, nil, err
	}
	ctx = NewContext(ctx, s)
	return ctx, s, nil
}

// C return a collection pointer by its name from default database.
func C(ctx context.Context, name string) (*mgo.Collection, error) {
	s, err := CtxSession(ctx)
	if err != nil {
		return nil, err
	}
	return Coll(s, name)
}

// Coll return database collection pointer.
func Coll(s *mgo.Session, name string) (*mgo.Collection, error) {
	cname, ok := Colls[name]
	if !ok {
		return nil, errors.New("unknown collection name")
	}
	return s.DB("").C(cname), nil
}

// MongoCredential initializes MongoDB credentials.
func MongoCredential(cfg *conf.MongoCfg) error {
	if cfg.Ssl {
		pool := x509.NewCertPool()
		pemData, err := ioutil.ReadFile(cfg.SslKeyFile)
		if err != nil {
			return err
		}
		ok := pool.AppendCertsFromPEM(pemData)
		if !ok {
			return errors.New("invalid certificate")
		}
		cert, err := tls.X509KeyPair(pemData, pemData)
		if err != nil {
			return err
		}
		tlsConfig := &tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{cert},
		}
		dial := func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			if err != nil {
				cfg.Logger.Printf("tls.Dial(%s) failed with %v", addr, err)
				return nil, err
			}
			cfg.Logger.Printf("SSL connection: %v", addr.String())
			return conn, nil
		}
		cfg.MongoCred = &mgo.DialInfo{
			Addrs:          cfg.Addrs(),
			Timeout:        time.Duration(cfg.Timeout) * time.Second,
			Database:       cfg.Database,
			Source:         cfg.AuthDB,
			Username:       cfg.Username,
			Password:       cfg.Password,
			ReplicaSetName: cfg.ReplicaSet,
			DialServer:     dial,
		}
	} else {
		cfg.MongoCred = &mgo.DialInfo{
			Addrs:          cfg.Addrs(),
			Timeout:        time.Duration(cfg.Timeout) * time.Second,
			Database:       cfg.Database,
			Source:         cfg.AuthDB,
			Username:       cfg.Username,
			Password:       cfg.Password,
			ReplicaSetName: cfg.ReplicaSet,
		}
	}
	return nil
}

// mongoDBConnection is an initialization of MongoDb connection.
func mongoDBConnection(cfg *conf.MongoCfg) (*mgo.Session, error) {
	if cfg.MongoCred == nil {
		err := MongoCredential(cfg)
		if err != nil {
			return nil, err
		}
	}
	session, err := mgo.DialWithInfo(cfg.MongoCred)
	if err != nil {
		return nil, err
	}
	if cfg.PoolLimit > 1 {
		session.SetPoolLimit(cfg.PoolLimit)
	}
	// session.EnsureSafe(&mgo.Safe{W: 1})
	if cfg.Debug {
		mgo.SetLogger(cfg.Logger)
		mgo.SetDebug(true)
	}
	return session, nil
}

// CheckID converts string s to ObjectId if it is possible,
// otherwise it returns error.
func CheckID(s string) (bson.ObjectId, error) {
	d, err := hex.DecodeString(s)
	if err != nil || len(d) != 12 {
		return "", errors.New("invalid database ID")
	}
	return bson.ObjectId(d), nil
}

// LockURL locks short URL creation actions.
// It is useful for distributed usage,
// database is used for consistency short URLs values.
func LockURL(s *mgo.Session) error {
	delay := time.Duration(time.Millisecond)
	coll := s.DB("").C(Colls["locks"])
	for i := 0; i < maxLockAttempts; i++ {
		err := coll.Insert(bson.M{"_id": lockKey})
		if err == nil {
			return nil
		}
		time.Sleep(delay)
		delay *= 2
	}
	return errors.New("can not lock URLs")
}

// UnlockURL unlocks short URLs creation actions.
func UnlockURL(s *mgo.Session) error {
	coll := s.DB("").C(Colls["locks"])
	err := coll.RemoveId(lockKey)
	if err != nil {
		Logger.Println(err)
		return err
	}
	return nil
}
