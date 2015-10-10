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
    "gopkg.in/mgo.v2/bson"
)

const (
    // SaltsLen in minimal recommended salt length.
    SaltsLen = 16
    // AnonName is name of anonymous user.
    AnonName = "anonymous"
    // DefaultProject is name of default project.
    DefaultProject = "system"
)

var (
    // AnonUser is anonymous user
    AnonUser = User{Name: AnonName, Role: "user", Created: time.Now().UTC()}
    // AnonProject is system project for administrative and anonymous users.
    AnonProject = Project{Name: DefaultProject, Users: []User{AnonUser}}
)

// User is structure of user's info.
type User struct {
    Name    string    `bson:"name"`
    Key     string    `bson:"key"`
    Role    string    `bson:"role"`
    Created time.Time `bson:"ts"`
    Secret  string    `bson:",omitempty"`
}

// Project is structure of project's info.
type Project struct {
    ID       bson.ObjectId `bson:"_id"`
    Name     string        `bson:"name"`
    Users    []User        `bson:"users"`
    Modified time.Time     `bson:"modified"`
}

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

// workers is main workers settings
type workers struct {
    CleanMin int64 `json:"cleanup"`
    NumStats int   `json:"numstats"`
    BufStats int   `json:"bufstat"`
    NumCb    int   `json:"numcb"`
    BufCb    int   `json:"bufcb"`
    CleanD   time.Duration
    ChStats  chan string
    ChCb     chan string
}

// projects is projects' settings.
type projects struct {
    MaxSpam   int  `json:"maxspam"`
    CbLength  int  `json:"cblength"`
    CbAllow   bool `json:"cballow"`
    MaxName   int  `json:"maxname"`
    Anonymous bool `json:"anonymous"`
    MaxPack   int  `json:"maxpack"`
}

// domain is settings if main service domain.
type domain struct {
    Name    string `json:"name"`
    Secure  bool   `json:"secure"`
    Address string
}

// Config is main configuration storage.
type Config struct {
    Domain   domain   `json:"domain"`
    Listener listener `json:"listener"`
    Db       MongoCfg `json:"database"`
    Cache    cacheCfg `json:"cache"`
    Workers  workers  `json:"workers"`
    Projects projects `json:"projects"`
    Pool     *hashq.HashQ
    Clean    chan string
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

// Address returns a full short url address.
func (c *Config) Address(url string) string {
    if c.Domain.Secure {
        return fmt.Sprintf("https://%s/%s", c.Domain.Name, url)
    }
    return fmt.Sprintf("http://%s/%s", c.Domain.Name, url)
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
