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
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/golang-lru"
	"github.com/z0rr0/luss/conf"
	"gopkg.in/mgo.v2"
)

const (
	// SaltsLen in minimal recommended salt length.
	SaltsLen = 16
	// AnonName is name of anonymous user.
	// AnonName = "anonymous"
	// DefaultProject is name of default project.
	// DefaultProject = "system"
)

// var (
// 	// AnonUser is anonymous user
// 	AnonUser = User{Name: AnonName, Role: "user", Created: time.Now().UTC()}
// 	// AnonProject is system project for administrative and anonymous users.
// 	AnonProject = Project{Name: DefaultProject, Users: []User{AnonUser}}
// )

// User is structure of user's info.
// type User struct {
// 	Name    string    `bson:"name"`
// 	Key     string    `bson:"key"`
// 	Role    string    `bson:"role"`
// 	Created time.Time `bson:"ts"`
// 	Secret  string    `bson:",omitempty"`
// }

// // Project is structure of project's info.
// type Project struct {
// 	ID       bson.ObjectId `bson:"_id"`
// 	Name     string        `bson:"name"`
// 	Users    []User        `bson:"users"`
// 	Modified time.Time     `bson:"modified"`
// }

// Conn is database connection structure.
type Conn struct {
	S   *mgo.Session
	m   sync.Mutex
	Cfg *conf.MongoCfg
}

// domain is settings if main service domain.
type domain struct {
	Name    string `json:"name"`
	Secure  bool   `json:"secure"`
	Address string
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

// projects is projects' settings.
type projects struct {
	MaxSpam   int   `json:"maxspam"`
	CleanMin  int64 `json:"cleanup"`
	CbAllow   bool  `json:"cballow"`
	CbNum     int   `json:"cbnum"`
	CbBuf     int   `json:"cbbuf"`
	CbLength  int   `json:"cblength"`
	MaxName   int   `json:"maxname"`
	Anonymous bool  `json:"anonymous"`
	MaxPack   int   `json:"maxpack"`
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
	MongoCred   *mgo.DialInfo
	Logger      *log.Logger
	Conn        *Conn
}

// Cache is database connections pool settings
type Cache struct {
	URLs        int `json:"urls"`
	Projects    int `json:"projects"`
	URLsLRU     *lru.Cache
	ProjectsLRU *lru.Cache
}

// Logger is common logger structure.
type Logger struct {
	Debug *log.Logger
	Info  *log.Logger
	Error *log.Logger
}

// Config is main configuration storage.
type Config struct {
	Domain   domain   `json:"domain"`
	Listener listener `json:"listener"`
	Projects projects `json:"projects"`
	Db       MongoCfg `json:"database"`
	Cache    cache    `json:"cache"`
	L        Logger
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
	port := fmt.Sprint(cfg.Port)
	for i, host := range cfg.Hosts {
		hosts[i] = net.JoinHostPort(host, port)
	}
	return hosts
}

// Parse reads a configuration file and
// returns a pointer to prepared Config structure.
func Parse(name string) (*Config, error) {
	cfg := &Config{}
	fullpath, err := filepath.Abs(strings.Trim(name, " "))
	if err != nil {
		return cfg, err
	}
	data, err := ioutil.ReadFile(fullpath)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, err
}
