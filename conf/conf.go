// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package conf implements configuration read methods.
package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/context"

	"github.com/hashicorp/golang-lru"
	"gopkg.in/mgo.v2"
)

const (
	// saltLent in minimal recommended salt length.
	saltLent = 16
	// AnonName is name of anonymous user.
	// AnonName = "anonymous"
	// DefaultProject is name of default project.
	// DefaultProject = "system"
)

var (
	// AnonUser is anonymous user
	// AnonUser = User{Name: AnonName, Role: "user", Created: time.Now().UTC()}
	// AnonProject is system project for administrative and anonymous users.
	// AnonProject = Project{Name: DefaultProject, Users: []User{AnonUser}}
	configKey key = 0
)

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

// key is internal type to get Config value from context.
type key int

// Conn is database connection structure.
type Conn struct {
	S   *mgo.Session
	M   sync.Mutex
	Cfg *MongoCfg
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
	PoolLimit   int      `json:"poollimit"`
	Debug       bool     `json:"debug"`
	MongoCred   *mgo.DialInfo
	Logger      *log.Logger
}

// cache is database connections pool settings
type cache struct {
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
	Debug    bool     `json:"debug"`
	Conn     *Conn
	L        Logger
}

// Address returns a full URL address.
func (c *Config) Address(url string) string {
	if c.Domain.Secure {
		return fmt.Sprintf("https://%s/%s", c.Domain.Name, url)
	}
	return fmt.Sprintf("http://%s/%s", c.Domain.Name, url)
}

// Validate validates configuration settings.
func (c *Config) Validate() error {
	var err error
	errFunc := func(msg, field string) error {
		return fmt.Errorf("invalid configuration \"%v\": %v", field, msg)
	}
	// listener and project settings
	switch {
	case c.Domain.Name == "":
		err = errFunc("short url domain can not be empty", "domain.name")
	case c.Listener.Port == 0:
		err = errFunc("not initialized server port value", "listener.port")
	case c.Listener.Timeout == 0:
		err = errFunc("not initialized server timeout value", "listener.timeout")
	case c.Listener.Security.TokenLen < 1:
		err = errFunc("incorrect or empty value", "listener.security.tokenlen")
	case len(c.Listener.Security.Salt) < saltLent:
		err = errFunc(fmt.Sprintf("insecure salt value, min length is %v symbols", saltLent), "listener.security.salt")
	case c.Projects.MaxSpam < 1:
		err = errFunc("incorrect or empty value", "projects.maxspam")
	case c.Projects.CleanMin < 1:
		err = errFunc("incorrect or empty value", "projects.cleanup")
	case c.Projects.CbNum < 1:
		err = errFunc("incorrect or empty value", "projects.cbnum")
	case c.Projects.CbBuf < 1:
		err = errFunc("incorrect or empty value", "projects.cbbuf")
	case c.Projects.CbLength < 1:
		err = errFunc("incorrect or empty value", "projects.cblength")
	case c.Projects.MaxName < 1:
		err = errFunc("incorrect or empty value", "projects.maxname")
	case c.Projects.MaxPack < 1:
		err = errFunc("incorrect or empty value", "projects.maxpack")
	}
	if err != nil {
		return err
	}
	// db connection check is skipped here
	c.Conn = &Conn{Cfg: &c.Db}
	// caching enabling
	if c.Cache.URLs > 0 {
		ca, err := lru.New(c.Cache.URLs)
		if err != nil {
			return err
		}
		c.Cache.URLsLRU = ca
	}
	if c.Cache.Projects > 0 {
		ca, err := lru.New(c.Cache.Projects)
		if err != nil {
			return err
		}
		c.Cache.ProjectsLRU = ca
	}
	// create logger
	logger := Logger{
		Debug: log.New(ioutil.Discard, "DEBUG [luss]: ", log.Ldate|log.Ltime|log.Lshortfile),
		Info:  log.New(os.Stdout, "INFO [luss]: ", log.Ldate|log.Ltime|log.Lshortfile),
		Error: log.New(os.Stderr, "ERROR [luss]: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
	if c.Debug {
		logger.Debug = log.New(os.Stdout, "DEBUG [luss]: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	c.L = logger
	return nil
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

// NewContext returns a new Context carrying Config.
func NewContext(c *Config) context.Context {
	return context.WithValue(context.Background(), configKey, c)
}

// FromContext extracts the Config from Context.
func FromContext(ctx context.Context) (*Config, error) {
	c, ok := ctx.Value(configKey).(*Config)
	if !ok {
		return nil, errors.New("not found context config")
	}
	return c, nil
}
