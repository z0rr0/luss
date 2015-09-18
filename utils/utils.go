// package utils contains LUSS aditional methods.
package utils

import (
    "sync"

    "gopkg.in/mgo.v2"
)

// Conn is database connection structure.
type Conn struct {
    Session *mgo.Session
    mutex   sync.Mutex
}

// ConnPool is a pool of database connections.
// We need: slow insert, fast update and read.
type ConnPool struct {
    Pool    []*Conn
    current int
    mutex   sync.RWMutex
}

func (c *ConnPool) Push(conn *Conn) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    c.Pool = append(c.Pool, conn)
}

func (c *ConnPool) Get() *Conn {

}
