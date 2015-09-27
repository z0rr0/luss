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

    "github.com/z0rr0/hashq"
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
    Conf   *conf.Config
    Pool   *hashq.HashQ
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

func checkDbConnection(cfg *conf.Config) error {
    conn, err := db.GetConn(cfg)
    if err != nil {
        return err
    }
    defer db.ReleaseConn(conn)
    LoggerInfo.Println("DB connection checked")
    return err
}

func InitFileConfig(filename string, debug bool) (*conf.Config, error) {
    Debug(debug)
    cf, err := conf.ParseConfig(filename)
    if err != nil {
        return nil, err
    }
    // check configuration values
    switch {
    case cf.Listener.Port == 0:
        err = errorGen("not initialized server port value", "listener.port")
    case cf.Db.Reconnects < 1:
        err = errorGen("database reconnects attempts should be greater than zero", "database.reconnects")
    case cf.Cache.DbPoolSize < 1:
        err = errorGen("connection pool size should be greater than zero", "cache.dbpoolsize")
    case !cf.GoodSalts():
        err = errorGen(fmt.Sprintf("insecure salt value, min length is %v symbols", conf.SaltsLen), "listener.security.salt")
    case cf.Listener.Security.DbKeys < 1:
        err = errorGen("incorrect or empty value", "listener.security.dbkeys")
    case cf.Listener.Security.TokenLen < 1:
        err = errorGen("incorrect or empty value", "listener.security.tokenlen")
    }

    if err != nil {
        return nil, err
    }
    cf.Db.RcnDelay = time.Duration(cf.Db.RcnTime) * time.Millisecond
    return cf, nil
}

// InitConfig initializes configuration from a file.
func InitConfig(filename string, debug bool) error {
    cf, err := InitFileConfig(filename, debug)
    if err != nil {
        return err
    }
    hashq.Debug(cf.Cache.Debug)
    // create connection pool
    pool, perr := db.NewConnPool(cf)
    if perr != nil {
        return perr
    }
    // common configuration
    Cfg = &Configuration{Conf: cf, Pool: pool, Logger: LoggerError}
    err = checkDbConnection(Cfg.Conf)
    if err != nil {
        return err
    }
    return nil
}
