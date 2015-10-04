// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package conf implements configuration read methods.
package conf

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "path/filepath"
    "strings"
    "time"

    "github.com/z0rr0/hashq"
    "github.com/z0rr0/luss/lru"
    "gopkg.in/mgo.v2"
)

const (
    // SaltsLen in minimal recommended salt length.
    SaltsLen = 16
)

// security contains main security settings.
type security struct {
    Salt     string `json:"salt"`
    TokenLen int    `json:"tokenlen"`
}

// listener is HTTP server configuration
type listener struct {
    Host     string   `json:"host"`
    Port     uint     `json:"port"`
    Timeout  int64    `json:"timeout"`
    Security security `json:"security"`
}

// MongoCfg is database configuration settings
type MongoCfg struct {
    Hosts       []string `json:"hosts"`
    Port        uint     `json:"port"`
    Timeout     uint     `json:"timeout"`
    Username    string   `json:"username"`
    Password    string   `json:"password"`
    Database    string   `json:"database"`
    AuthDB      string   `json:"authdb"`
    ReplicaSet  string   `json:"replica"`
    Ssl         bool     `json:"ssl"`
    SslKeyFile  string   `json:"sslkeyfile"`
    PrimaryRead bool     `json:"primaryread"`
    Reconnects  int      `json:"reconnects"`
    RcnTime     int64    `json:"rcntime"`
    Debug       bool     `json:"debug"`
    RcnDelay    time.Duration
    ConChan     chan hashq.Shared
    MongoCred   *mgo.DialInfo
    Logger      *log.Logger
}

// cacheCfg is database connections pool settings
type cacheCfg struct {
    DbPoolSize int   `json:"dbpoolsize"`
    DbPoolTTL  int64 `json:"dbpoolttl"`
    LRUSize    int   `json:"lrusize"`
    Debug      bool  `json:"debug"`
    LRU        *lru.Cache
}

// Config is main configuration storage.
type Config struct {
    Listener listener `json:"listener"`
    Db       MongoCfg `json:"database"`
    Cache    cacheCfg `json:"cache"`
}

// ConnCap returns a recommended connections capacity.
func (c *Config) ConnCap() int {
    var result int
    switch {
    case c.Cache.DbPoolSize > 128:
        result = 32
    case c.Cache.DbPoolSize > 32:
        result = 16
    case c.Cache.DbPoolSize > 8:
        result = 8
    default:
        result = c.Cache.DbPoolSize
    }
    return result
}

// GoodSalts verifies salts values.
func (c *Config) GoodSalts() bool {
    return len(c.Listener.Security.Salt) >= SaltsLen
}

// Addrs return an array of available MongoDB connections addresses.
func (cfg *MongoCfg) Addrs() []string {
    hosts := make([]string, len(cfg.Hosts))
    for i, host := range cfg.Hosts {
        hosts[i] = fmt.Sprintf("%v:%v", host, cfg.Port)
    }
    return hosts
}

// ParseConfig reads a configuration file and
// returns a pointer to prepared Config structure.
func ParseConfig(name string) (*Config, error) {
    cfg := &Config{}
    fullpath, err := filepath.Abs(strings.Trim(name, " "))
    if err != nil {
        return cfg, err
    }
    jsondata, err := ioutil.ReadFile(fullpath)
    if err != nil {
        return cfg, err
    }
    err = json.Unmarshal(jsondata, cfg)
    return cfg, err
}
