// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package utils contains LUSS additional methods.
package utils

import (
    "fmt"
    "io/ioutil"
    "log"
    "net/url"
    "os"
    "time"

    "github.com/z0rr0/hashq"
    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
    "github.com/z0rr0/luss/lru"
    "golang.org/x/net/idna"
    "gopkg.in/mgo.v2/bson"
)

const (
    // ReleaseMode turns off debug mode
    ReleaseMode = 0
    // DebugMode turns on debug mode
    DebugMode = 1
    // cleanBuf is buffer size of projects' cleanup calls.
    cleanBuf = 5
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
    LoggerDebug.Println("DB connection checked")
    return err
}

// InitDefaultProject creates default system project.
func InitDefaultProject(c *conf.Config) error {
    conn, err := db.GetConn(c)
    defer db.ReleaseConn(conn)
    if err != nil {
        return err
    }
    coll := conn.C(db.Colls["projects"])
    data := bson.M{
        "$setOnInsert": bson.M{
            "name":     conf.AnonProject.Name,
            "users":    conf.AnonProject.Users,
            "modified": time.Now().UTC(),
        },
    }
    _, err = coll.Upsert(bson.M{"name": conf.DefaultProject}, data)
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
    case cf.Listener.Security.TokenLen < 1:
        err = errorGen("incorrect or empty value", "listener.security.tokenlen")
    case cf.Workers.CleanMin < 1:
        err = errorGen("incorrect or empty value", "workers.cleanup")
    case cf.Workers.NumStats < 1:
        err = errorGen("incorrect or empty value", "workers.numstats")
    case cf.Workers.NumCb < 1:
        err = errorGen("incorrect or empty value", "workers.numcb")
    case cf.Workers.BufStats < 1:
        err = errorGen("incorrect or empty value", "workers.bufstats")
    case cf.Workers.BufCb < 1:
        err = errorGen("incorrect or empty value", "workers.bufcb")
    case cf.Projects.MaxSpam < 1:
        err = errorGen("incorrect or empty value", "projects.maxspam")
    case cf.Projects.CbLength < 1:
        err = errorGen("incorrect or empty value", "projects.cblength")
    case cf.Projects.MaxName < 2:
        err = errorGen("incorrect or empty value", "projects.maxname")
    }
    if err != nil {
        return nil, err
    }
    if cf.Cache.LRUSize < 1 {
        LoggerInfo.Println("LRU cache is disabled")
    } else {
        LoggerDebug.Printf("LRU cache size is %v", cf.Cache.LRUSize)
    }
    if cf.Workers.BufStats > cf.Workers.NumStats {
        cf.Workers.BufStats = cf.Workers.NumStats
    }
    if cf.Workers.BufCb > cf.Workers.NumCb {
        cf.Workers.BufCb = cf.Workers.NumCb
    }
    cf.Cache.LRU = lru.New(cf.Cache.LRUSize)
    cf.Db.RcnDelay = time.Duration(cf.Db.RcnTime) * time.Millisecond
    cf.Workers.CleanD = time.Duration(cf.Workers.CleanMin) * time.Second
    // create connection pool
    hashq.Debug(cf.Cache.Debug)
    pool, perr := db.NewConnPool(cf)
    if perr != nil {
        return nil, perr
    }
    cf.Pool, cf.Clean = pool, make(chan string, cleanBuf)
    err = InitDefaultProject(cf)
    if err != nil {
        return nil, err
    }
    cf.Db.Logger = LoggerError
    return cf, nil
}

// InitConfig initializes configuration from a file.
func InitConfig(filename string, debug bool) error {
    cf, err := InitFileConfig(filename, debug)
    if err != nil {
        return err
    }
    // common configuration
    Cfg = &Configuration{Conf: cf, Logger: LoggerError}
    go Cfg.RunWorkers()
    go Cfg.URLCleaner()
    return checkDbConnection(Cfg.Conf)
}

// Stat updates statistics about CustomURL usage.
func Stat(url string, conf *conf.Config) error {
    conn, err := db.GetConn(conf)
    defer db.ReleaseConn(conn)
    if err != nil {
        LoggerError.Printf("can't update statistics: %v", err)
        return err
    }
    coll := conn.C(db.Colls["ustats"])
    y, m, d := time.Now().UTC().Date()
    day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
    _, err = coll.Upsert(bson.M{"url": url, "day": day}, bson.M{"$inc": bson.M{"c": 1}})
    if err != nil {
        LoggerError.Printf("can't update statistics: %v", err)
    }
    return err
}

// CallBAck send callback requests.
func CallBAck(pname string, conf *conf.Config) error {
    conn, err := db.GetConn(conf)
    defer db.ReleaseConn(conn)
    if err != nil {
        LoggerError.Printf("callback error: %v", err)
        return err
    }
    // coll := conn.C(db.Colls["projects"])
    // TODO: find callbacks and send requests
    return nil
}

// URLCleaner deletes expired short links or all ones of a requested project.
func (c *Configuration) URLCleaner() {
    LoggerDebug.Println("URLCleaner is started")
    urlsC := db.Colls["urls"]
    clean := func(cond bson.M) {
        conn, err := db.GetConn(c.Conf)
        defer db.ReleaseConn(conn)
        if err != nil {
            LoggerError.Printf("can't run cleanup: %v", err)
            return
        }
        coll := conn.C(urlsC)
        info, err := coll.UpdateAll(cond, bson.M{"$set": bson.M{"active": false, "mod": time.Now().UTC()}})
        if err != nil {
            LoggerError.Printf("can't finish cleanup: %v", err)
            return
        }
        LoggerInfo.Printf("URLCleaner disabled %v item(s)", info.Updated)
    }
    for {
        select {
        case <-time.After(c.Conf.Workers.CleanD):
            clean(bson.M{"ttl": bson.M{"$lt": time.Now().UTC()}, "active": true})
        case p := <-c.Conf.Clean:
            clean(bson.M{"prj": p, "active": true})
        }
    }
}

// RunWorkers runs workers goroutines to handled
// statistics saving and callbacks requests.
func (c *Configuration) RunWorkers() {
    c.Conf.Workers.ChStats = make(chan string, c.Conf.Workers.BufStats)
    c.Conf.Workers.ChCb = make(chan string, c.Conf.Workers.BufCb)
    for i := 0; i < c.Conf.Workers.NumStats; i++ {
        // start stat handlers
        go func() {
            for s := range c.Conf.Workers.ChStats {
                Stat(s, c.Conf)
            }
        }()
    }
    for i := 0; i < c.Conf.Workers.NumCb; i++ {
        // start callbacks handlers
        go func() {
            for s := range c.Conf.Workers.ChCb {
                CallBAck(s, c.Conf)
            }
        }()
    }
    LoggerDebug.Println("RunWorkers is started")
}

// ParseURL parses rawurl into a URL structure and checks/converts IDNA hostname.
func ParseURL(rawurl string) (*url.URL, error) {
    url, err := url.ParseRequestURI(rawurl)
    if err != nil {
        return nil, err
    }
    host, err := idna.ToASCII(url.Host)
    if err != nil {
        return nil, err
    }
    url.Host = host
    return url, nil
}
