// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package utils contains LUSS additional methods.
package utils

import (
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "time"

    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
)

const (
    // ReleaseMode turns off debug mode
    ReleaseMode = 0
    // DebugMode turns on debug mode
    DebugMode = 1
)

var (
    // Mode is a current debug/release mode.
    Mode = ReleaseMode
    // LoggerError is a logger for error messages
    LoggerError = log.New(os.Stderr, "ERROR [LUSS]: ", log.Ldate|log.Ltime|log.Lshortfile)
    // LoggerInfo is a logger for info messages
    LoggerInfo = log.New(os.Stdout, "INFO [LUSS]: ", log.Ldate|log.Ltime|log.Lshortfile)
    // LoggerDebug is a logger for debug messages
    LoggerDebug = log.New(ioutil.Discard, "DEBUG [LUSS]: ", log.Ldate|log.Lmicroseconds|log.Llongfile)
    // Cfg is a main configuration object.
    Cfg *Configuration
    // Conns is a channel to get database connection.
    Conns chan db.Conn
)

// Configuration is main configuration storage.
// It is used as a singleton.
type Configuration struct {
    C      *conf.Config
    Pool   *db.ConnPool
    Conn   chan *db.Conn
    Logger *log.Logger
}

// Debug activates debug mode.
func Debug(debug bool) {
    debugHandler := ioutil.Discard
    if debug {
        debugHandler = os.Stdout
        Mode = DebugMode
    } else {
        Mode = ReleaseMode
    }
    LoggerDebug = log.New(debugHandler, "DEBUG [LUSS]: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// errorGen simplifies error generation during configuration validation
func errorGen(msg, field string) error {
    return fmt.Errorf("invalid configuration \"%v\": %v", field, msg)
}

func checkDbConnection(cfg *conf.MongoCfg, ch chan *db.Conn) error {
    conn, err := db.GetConn(cfg, ch)
    if err != nil {
        return err
    }
    defer conn.Release()
    LoggerInfo.Println("DB connection checked")
    return err
}

// InitConfig initializes configuration from a file.
func InitConfig(filename string, debug bool) error {
    cf, err := conf.ParseConfig(filename)
    if err != nil {
        return err
    }
    // check configuration values
    switch {
    case cf.Listener.Port == 0:
        err = errorGen("not initialized server port value", "listener.port")
    case cf.Db.Reconnects < 1:
        err = errorGen("database reconnects attempts should be greater than zero", "database.reconnects")
    case cf.Cache.DbPoolSize < 1:
        err = errorGen("connection pool size should be greater than zero", "cache.dbpoolsize")
    }
    if err != nil {
        return err
    }
    cf.Db.RcnDelay = time.Duration(cf.Db.RcnTime) * time.Millisecond
    // create connection pool
    pool := db.NewConnPool(cf.Cache.DbPoolSize)
    ch, errch := make(chan *db.Conn, pool.Cap()), make(chan error)
    go pool.Produce(ch, errch)
    err = <-errch
    if err != nil {
        return err
    }
    // common configuration
    Cfg = &Configuration{C: cf, Pool: pool, Conn: ch, Logger: LoggerError}
    err = checkDbConnection(&Cfg.C.Db, Cfg.Conn)
    if err != nil {
        return err
    }
    go Cfg.Pool.Monitor(time.Duration(Cfg.C.Cache.DbPoolTTL) * time.Second)
    Debug(debug)
    return nil
}
