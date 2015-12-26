// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package db implements MongoDB database access methods.
package db

import (
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
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	sessionKey key = 0
)

var (
	// Logger is a logger for error messages
	Logger = log.New(os.Stderr, "LOGGER [db]: ", log.Ldate|log.Ltime|log.Lshortfile)
	// Colls is a map of db collections names.
	// Keys can be used as aliases, values are real collection names.
	Colls = map[string]string{
		"test":     "test",
		"users":    "users",
		"urls":     "urls",
		"locks":    "locks",
		"ustats":   "ustats",
		"projects": "projects",
	}
)

// key is internal type to get session value from context.
type key int

// Item is any DB item, it contains only identifier.
type Item struct {
	ID bson.ObjectId `bson:"_id"`
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

// connect sets new MongoDB connection
// and saves the session to Conn.S fields.
func connect(c *conf.Conn) error {
	c.M.Lock()
	defer c.M.Unlock()
	if c.S == nil {
		Logger.Println("new session creation")
		s, err := mongoDBConnection(c.Cfg)
		if err != nil {
			return err
		}
		c.S = s
	} else if c.S.Ping() != nil {
		Logger.Println("session recreation")
		s, err := mongoDBConnection(c.Cfg)
		if err != nil {
			return err
		}
		// old session is not valid
		// close it and use new one
		oldS := c.S
		c.S = s
		oldS.Close()
	}
	return nil
}

// NewSession returns new MongoDB session based on Conn data.
func NewSession(c *conf.Conn, primary bool) (*mgo.Session, error) {
	if (c.S == nil) || (c.S.Ping() != nil) {
		err := connect(c)
		if err != nil {
			Logger.Printf("invalid connect: %v", err)
			return nil, err
		}
	}
	s := c.S.Copy()
	if !primary {
		s.SetMode(mgo.SecondaryPreferred, true)
	}
	return s, nil
}

// C return a collection pointer by its name from default database.
func C(ctx context.Context, name string) (*mgo.Collection, error) {
	s, err := CtxSession(ctx)
	if err != nil {
		return nil, err
	}
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
	mode := mgo.PrimaryPreferred
	if !cfg.PrimaryRead {
		mode = mgo.SecondaryPreferred
	}
	session.SetMode(mode, true)
	if cfg.PoolLimit < 2 {
		cfg.PoolLimit = 2
	}
	session.SetPoolLimit(cfg.PoolLimit)
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

// // CleanCollection removes all data from db collection.
// func CleanCollection(c *conf.Config, names ...string) error {
// 	conn, err := GetConn(c)
// 	defer ReleaseConn(conn)
// 	if err != nil {
// 		return err
// 	}
// 	for _, name := range names {
// 		if err == nil {
// 			coll := conn.C(name)
// 			_, err = coll.RemoveAll(nil)
// 		}
// 	}
// 	return err
// }

// // LockColls adds a lock recored to name-collection
// func LockColls(name string, conn *Conn) error {
// 	delay := time.Duration(time.Millisecond)
// 	coll := conn.C(Colls["locks"])
// 	for i := 0; i < maxLockAttempts; i++ {
// 		_, err := coll.Upsert(bson.M{"_id": name, "locked": false}, bson.M{"_id": name, "locked": true})
// 		if err == nil {
// 			return nil
// 		}
// 		time.Sleep(delay)
// 		delay *= 2
// 	}
// 	return fmt.Errorf("can't lock/update collection \"%v\" during %v attempts", Colls["locks"], maxLockAttempts)
// }

// // UnlockColls removes a lock recored from name-collection.
// func UnlockColls(name string, conn *Conn) error {
// 	coll := conn.C(Colls["locks"])
// 	return coll.Update(bson.M{"_id": name}, bson.M{"$set": bson.M{"locked": false}})
// }
