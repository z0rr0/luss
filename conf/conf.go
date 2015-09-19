// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package conf implements MongoDB database access method.
package conf

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    // "os"
    "path/filepath"
    "strings"
    "time"

    "gopkg.in/mgo.v2"
)

// listener is HTTP server configuration
type listener struct {
    Host    string `json:"host"`
    Port    uint   `json:"port"`
    Timeout int64  `json:"timeout"`
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
    Reconnects  uint     `json:"reconnects"`
    RcnTime     int64    `json:"rcntime"`
    Debug       bool     `json:"debug"`
    RcnDelay    time.Duration
    MongoCred   *mgo.DialInfo
    Logger      *log.Logger
}

// cacheCfg is database connections pool settings
type cacheCfg struct {
    DbPoolSize int   `json:"dbpoolsize"`
    DbPoolTTL  int64 `json:"dbpoolttl"`
}

// Config is main configuration storage.
type Config struct {
    Listener listener `json:"listener"`
    Db       MongoCfg `json:"database"`
    Cache    cacheCfg `json:"cache"`
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
