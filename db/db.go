// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package db implements MongoDB database access method.
package db

import (
    "crypto/tls"
    "crypto/x509"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "os"
    "sync"
    "sync/atomic"
    "time"

    "github.com/z0rr0/luss/conf"
    "gopkg.in/mgo.v2"
)

var (
    // LoggerError is a logger for error messages
    LoggerError = log.New(os.Stderr, "ERROR [LUSS-db]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Conn is database connection structure.
type Conn struct {
    Session *mgo.Session
    mutex   sync.RWMutex
    one     sync.Once
}

// ConnPool is a pool of database connections.
// It needs: slow insert, fast update and read.
type ConnPool struct {
    Pool  []*Conn
    conf  *MongoConf
    mutex sync.RWMutex
    ptr   int // current positions
    size  int // pool size
}

// Push pushes a pointer of new mogno session to the connection pool.
func (c *ConnPool) Push(s *mgo.Session) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    c.Pool = append(c.Pool, &Conn{Session: s})
    c.size = len(c.Pool)
}

// Clean iterates by connections pool and closes all database sessions.
func (c *ConnPool) Clean() {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    for _, conn := range c.Pool {
        // check session only to don't lock recently used connections
        if conn.Session != nil {
            conn.mutex.Lock()
            conn.close()
            conn.mutex.Unlock()
        }
    }
}

// Monitor closes unused connections with period d.
// It can be run in separated goroutine.
func (c *ConnPool) Monitor(d time.Duration) {
    for {
        select {
        case <-time.After(d):
            c.clean()
        }
    }
}

// Produce constantly gets new connections from the pool,
// opens it if needed and sends to ch channel.
// If an error will be during connection opening,
// then a consumer will get a connection with empty session.
func (c *ConnPool) Produce(ch chan<- *Conn) error {
    if c.size == 0 {
        return errors.New("empty connections pool")
    }
    // pool is not blocked because its size can't be decreased
    for {
        if c.ptr == c.size {
            c.ptr = 0
        }
        conn := c.Pool[c.ptr%c.size]
        conn.mutex.RLock()
        // consumer should call conn.Release() later
        once := func() {
            conn.open(c.conf)
        }
        conn.one.Do(once)
        ch <- conn
    }
    return nil
}

// open opens new database connection but only if it wasn't ready before.
func (c *Conn) open(cfg *conf.MongoCfg) error {
    session, err := MongoDBConnection(c.conf)
    if err != nil {
        LoggerError.Printf("can't open database connection: %v", err)
        return err
    }
    c.Session = session
    return nil
}

// close closes open connection.
// Connection pool can't be decreased, so it is not blocked in this method.
func (c *Conn) close() {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if c.Session != nil {
        c.Session.Close()
        c.Session, c.one = nil, sync.Once{}
    }
}

// Release releases connection resource, but doesn't close.
func (c *Conn) Release() {
    c.mutex.RUnlock()
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

// GetConnection returns a database connection from the pool.
// Caller should run conn.Release() after connection use in any way.
func GetConnection(ch <-chan *Conn) (*Conn, error) {
    conn := <-ch
    if conn.Session == nil {
        return nil, errors.New("empty db session")
    }
    return conn, nil
}
