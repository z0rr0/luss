// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package db implements MongoDB database access methods.
package db

import (
    "crypto/tls"
    "crypto/x509"
    "errors"
    "io/ioutil"
    "log"
    "net"
    "os"
    // "runtime/debug"
    "sync"
    "time"

    "github.com/z0rr0/hashq"
    "github.com/z0rr0/luss/conf"
    "gopkg.in/mgo.v2"
)

var (
    // Logger is a logger for error messages
    Logger = log.New(os.Stderr, "LOGGER [luss/db]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Conn is database connection structure.
// mgo is thread-safe and pings a connection after its opening.
type Conn struct {
    Session *mgo.Session
    mutex   sync.RWMutex
    one     sync.Once
    rlu     bool // recently used
}

// New creates new Conn structure.
func (c *Conn) New() hashq.Shared {
    return &Conn{}
}

// Close closes an opened connection. d is a timeout after closing.
// Connection pool can't be decreased, so it is not blocked in this method.
func (c *Conn) Close(d time.Duration) bool {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    if c.Session == nil {
        c.one = sync.Once{}
        return false
    }
    // don't close a connection immediately,
    // only if it was not used during all d period.
    if c.rlu {
        c.rlu = false
        return false
    }
    c.Session.Close()
    c.Session, c.one = nil, sync.Once{}
    time.Sleep(d)
    return true
}

// Open opens new database connection but only if it wasn't ready before.
// The boolean returned value will be true, it a connection was opened firstly.
func (c *Conn) Open(cfg *conf.MongoCfg) (bool, error) {
    var (
        err   error
        isNew bool
    )
    c.mutex.RLock()
    openOnce := func() {
        s, serr := MongoDBConnection(cfg)
        if serr != nil {
            err = serr
            return
        }
        c.Session, isNew = s, true
        c.Database = s.DB(cfg.Db.Database)
    }
    c.one.Do(openOnce)
    return isNew, err
}

// release releases connection resource, but doesn't close.
func (c *Conn) release() {
    c.mutex.RUnlock()
}

// C returns a pointer to the database collection cname.
// It is only a wrapper method, and should be called only if connection c is opened.
func (c *Conn) C(cname string) *mgo.Collection {
    // default database from mgo.DialInfo will be used.
    return c.Session.DB("").C(cname)
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

// MongoDBConnection is an initialization of MongoDb connection.
func MongoDBConnection(cfg *conf.MongoCfg) (*mgo.Session, error) {
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
    if !cfg.PrimaryRead {
        session.SetMode(mgo.Eventual, true)
    }
    // session.EnsureSafe(&mgo.Safe{W: 1})
    if cfg.Debug {
        mgo.SetLogger(cfg.Logger)
        mgo.SetDebug(true)
    }
    return session, nil
}

// ReleaseConn unlocks a connection if it exists.
func ReleaseConn(c *Conn) {
    if c != nil {
        c.release()
    }
}

// GetConn returns a database connection from the pool.
// Caller should run conn.Release() after connection usage.
func GetConn(cfg *conf.Config) (*Conn, error) {
    isNew, err := true, errors.New("connection initial error")
    shared := <-cfg.Db.ConChan
    conn := shared.(*Conn)
    for i := 0; i < cfg.Db.Reconnects; i++ {
        isNew, err = conn.Open(&cfg.Db)
        if !isNew && (err == nil) {
            err = conn.Session.Ping()
        }
        if err == nil {
            // Close() checks and updates this field also
            conn.rlu = true
            // conn is RLocked by Open()
            return conn, nil
        }
        conn.Release()
        conn.Close(cfg.Db.RcnDelay)
    }
    // connection is already closed
    // don't return it to use in ReleaseConn
    return nil, err
}

// NewConnPool initializes new connections pool.
func NewConnPool(cfg *conf.Config) (*hashq.HashQ, error) {
    if cfg == nil {
        return nil, errors.New("invalid parameter")
    }
    pool := hashq.New(cfg.Cache.DbPoolSize, &Conn{}, 0)
    ch, errch := make(chan hashq.Shared, cfg.ConnCap()), make(chan error)
    go pool.Produce(ch, errch)
    err := <-errch
    if err != nil {
        return nil, err
    }
    go pool.Monitor(time.Duration(cfg.Cache.DbPoolTTL) * time.Second)
    cfg.Db.ConChan = ch
    return pool, nil
}
