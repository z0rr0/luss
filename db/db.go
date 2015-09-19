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

    "github.com/z0rr0/luss/conf"
    "gopkg.in/mgo.v2"
)

var (
    // Logger is a logger for error messages
    Logger = log.New(os.Stderr, "LOGGER [luss/db]: ", log.Ldate|log.Ltime|log.Lshortfile)
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
    mutex sync.Mutex
}

// Cap returns a recommended capacity for the connections channel.
func (c *ConnPool) Cap() int {
    var result int
    switch s := c.Size(); {
    case s > 128:
        result = 32
    case s > 32:
        result = 16
    default:
        result = s
    }
    return result
}

// NewConnPool creates new connections pool.
func NewConnPool(size int) *ConnPool {
    c := &ConnPool{}
    for i := 0; i < size; i++ {
        c.Push(nil)
    }
    return c
}

// Push pushes a pointer of new MongoDB session to the connection pool.
func (c *ConnPool) Push(s *mgo.Session) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    c.Pool = append(c.Pool, &Conn{Session: s})
}

// Clean iterates by connections pool and closes all database sessions.
// Only one Clean() function works in one time.
func (c *ConnPool) Clean() {
    Logger.Printf("run connection clean, num=%v", c.Size())

    c.mutex.Lock()
    defer c.mutex.Unlock()
    i := 0
    for _, conn := range c.Pool {
        // check session only to don't close recently/now used connections
        if conn.Session != nil {
            conn.close(0)
            i++
        }
    }
    Logger.Printf("end connection clean, closed=%v", i)
}

// Size returns a pool size.
func (c *ConnPool) Size() int {
    return len(c.Pool)
}

// Monitor closes unused connections with period d.
// It can be run in a separated goroutine.
func (c *ConnPool) Monitor(d time.Duration) {
    for {
        select {
        case <-time.After(d):
            c.Clean()
        }
    }
}

// Produce constantly gets new connections from the pool, and sends to ch channel.
// It doesn't open them, because a consumer will do this.
// The channel ch should be buffered to exclude a bottle neck here.
func (c *ConnPool) Produce(ch chan<- *Conn, err chan error) {
    if c.Size() == 0 {
        err <- errors.New("empty connections pool")
        return
    }
    err <- nil
    // pool is not blocked because its size can't be decreased.
    for {
        for _, conn := range c.Pool {
            ch <- conn
        }
    }
}

// Open opens new database connection but only if it wasn't ready before.
func (c *Conn) Open(cfg *conf.MongoCfg) error {
    var err error
    c.mutex.RLock()
    openOnce := func() {
        s, serr := MongoDBConnection(cfg)
        if serr != nil {
            err = serr
            return
        }
        c.Session = s
    }
    c.one.Do(openOnce)
    return err
}

// close closes open connection.
// Connection pool can't be decreased, so it is not blocked in this method.
func (c *Conn) close(d time.Duration) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if c.Session != nil {
        c.Session.Close()
        c.Session, c.one = nil, sync.Once{}
    }
    time.Sleep(d)
}

// Release releases connection resource, but doesn't close.
func (c *Conn) Release() {
    c.mutex.RUnlock()
}

// C returns a pointer to the database collection cname.
// It is only a wrapper method, and should be called only if connection c is opened.
func (c *Conn) C(cfg *conf.MongoCfg, cname string) *mgo.Collection {
    return c.Session.DB(cfg.Database).C(cname)
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

// GetConn returns a database connection from the pool.
// Caller should run conn.Release() after connection usage if err is nil.
func GetConn(cfg *conf.MongoCfg, ch <-chan *Conn) (*Conn, error) {
    var (
        i   uint
        err error
    )
    conn := <-ch
    for i = 0; i < cfg.Reconnects; i++ {
        err = conn.Open(cfg)
        if err != nil {
            conn.Release()
            conn.close(cfg.RcnDelay)
            continue
        }
        // send a ping before connection usage,
        // it adds 100-150 µs during local tests.
        err = conn.Session.Ping()
        if err != nil {
            conn.Release()
            conn.close(cfg.RcnDelay)
            continue
        }
        // all is ok
        break
    }
    return conn, err
}
