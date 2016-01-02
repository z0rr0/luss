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
	"text/template"

	"github.com/hashicorp/golang-lru"
	"github.com/oschwald/geoip2-golang"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
)

const (
	// saltLent in minimal recommended salt length.
	saltLent = 16
	// configKey is internal context key
	configKey key = 0
)

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
	Templates string   `json:"templates"`
	Host      string   `json:"host"`
	Port      uint     `json:"port"`
	Timeout   int64    `json:"timeout"`
	Security  security `json:"security"`
}

// settings is a struct for different settings.
type settings struct {
	MaxSpam   int    `json:"maxspam"`
	CleanMin  int64  `json:"cleanup"`
	CbAllow   bool   `json:"cballow"`
	CbNum     int    `json:"cbnum"`
	CbBuf     int    `json:"cbbuf"`
	CbLength  int    `json:"cblength"`
	MaxName   int    `json:"maxname"`
	Anonymous bool   `json:"anonymous"`
	MaxPack   int    `json:"maxpack"`
	Trackers  int    `json:"trackers"`
	GeoIPDB   string `json:"geoipdb"`
}

// MongoCfg is database configuration settings
type MongoCfg struct {
	Hosts      []string `json:"hosts"`
	Port       uint     `json:"port"`
	Timeout    uint     `json:"timeout"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Database   string   `json:"database"`
	AuthDB     string   `json:"authdb"`
	ReplicaSet string   `json:"replica"`
	Ssl        bool     `json:"ssl"`
	SslKeyFile string   `json:"sslkeyfile"`
	Reconnects int      `json:"reconnects"`
	RcnTime    int64    `json:"rcntime"`
	PoolLimit  int      `json:"poollimit"`
	Debug      bool     `json:"debug"`
	MongoCred  *mgo.DialInfo
	Logger     *log.Logger
}

// cache is database connections pool settings
type cache struct {
	URLs      int `json:"urls"`
	Templates int `json:"templates"`
	Strorage  map[string]*lru.Cache
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
	Settings settings `json:"settings"`
	Db       MongoCfg `json:"database"`
	Cache    cache    `json:"cache"`
	Debug    bool     `json:"debug"`
	Conn     *Conn
	GeoDB    *geoip2.Reader
	L        Logger
}

// Close closes main database connection.
func (c *Conn) Close() {
	c.M.Lock()
	defer c.M.Unlock()
	if c.S != nil {
		c.S.Close()
	}
}

// Close releases configuration resources.
func (c *Config) Close() {
	c.Conn.Close()
	c.GeoDB.Close()
}

// Address returns a full URL address.
func (c *Config) Address(url string) string {
	domain := c.Domain.Name
	if c.Debug {
		domain += fmt.Sprintf(":%v", c.Listener.Port)
	}
	if c.Domain.Secure {
		return fmt.Sprintf("https://%s/%s", domain, url)
	}
	return fmt.Sprintf("http://%s/%s", domain, url)
}

// checkTemplates verifies template path value and updates it if needed.
func (c *Config) checkTemplates() error {
	fullpath, err := filepath.Abs(strings.Trim(c.Listener.Templates, " "))
	if err != nil {
		return err
	}
	fm, err := os.Stat(fullpath)
	if err != nil {
		return err
	}
	if !fm.Mode().IsDir() {
		return fmt.Errorf("templates folder is not a directory")
	}
	c.Listener.Templates = fullpath
	return nil
}

// checkGeoIPDB validates Geo IP database file path.
func (c *Config) checkGeoIPDB() error {
	fullpath, err := filepath.Abs(c.Settings.GeoIPDB)
	if err != nil {
		return err
	}
	db, err := geoip2.Open(fullpath)
	if err != nil {
		return err
	}
	c.GeoDB = db
	return nil
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
	case c.Settings.MaxSpam < 1:
		err = errFunc("incorrect or empty value", "projects.maxspam")
	case c.Settings.CleanMin < 1:
		err = errFunc("incorrect or empty value", "projects.cleanup")
	case c.Settings.CbNum < 1:
		err = errFunc("incorrect or empty value", "projects.cbnum")
	case c.Settings.CbBuf < 1:
		err = errFunc("incorrect or empty value", "projects.cbbuf")
	case c.Settings.CbLength < 1:
		err = errFunc("incorrect or empty value", "projects.cblength")
	case c.Settings.MaxName < 1:
		err = errFunc("incorrect or empty value", "projects.maxname")
	case c.Settings.MaxPack < 1:
		err = errFunc("incorrect or empty value", "projects.maxpack")
	case c.Settings.Trackers < 1:
		err = errFunc("incorrect or empty value", "projects.trackers")
	case c.checkTemplates() != nil:
		err = errFunc("invalid template name", "listener.templates")
	case c.Cache.URLs < 0:
		err = errFunc("incorrect value", "cache.urls")
	case c.Cache.Templates < 0:
		err = errFunc("incorrect value", "cache.templates")
	}
	if err != nil {
		return err
	}
	// db connection check is skipped here
	c.Conn = &Conn{Cfg: &c.Db}
	// caching enabling
	err = c.allocateLRU()
	if err != nil {
		return err
	}
	// geo ip
	err = c.checkGeoIPDB()
	if err != nil {
		return err
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

// NewContext returns a new Context carrying Config.
func NewContext(c *Config) context.Context {
	return context.WithValue(context.Background(), configKey, c)
}

// tpl returns absolute templates files paths
func (c *Config) tpl(tpls ...string) []string {
	paths := make([]string, len(tpls))
	for i := range tpls {
		paths[i] = filepath.Join(c.Listener.Templates, tpls[i])
	}
	return paths
}

// CacheTpl return HTML template from file, but first tries to find it in the LRU cache.
func (c *Config) CacheTpl(key string, tpls ...string) (*template.Template, error) {
	cache, cacheOn := c.Cache.Strorage["Tpl"]
	if cacheOn {
		if t, ok := cache.Get(key); ok {
			return t.(*template.Template), nil
		}
	}
	templates := c.tpl(tpls...)
	te, err := template.ParseFiles(templates...)
	if err != nil {
		return nil, err
	}
	if cacheOn {
		cache.Add(key, te)
	}
	return te, nil
}

// StaticDir returns a path of static files
func (c *Config) StaticDir() (string, error) {
	fullpath := filepath.Join(c.Listener.Templates, "static")
	fm, err := os.Stat(fullpath)
	if err != nil {
		return "", err
	}
	if !fm.Mode().IsDir() {
		return "", fmt.Errorf("not found directory of static files: %v", fullpath)
	}
	return fullpath, nil
}

// allocateLRU allocated LRU cache if it was activated.
func (c *Config) allocateLRU() error {
	c.Cache.Strorage = make(map[string]*lru.Cache)
	if size := c.Cache.Templates; size > 0 {
		storage, err := lru.New(size)
		if err != nil {
			return err
		}
		c.Cache.Strorage["Tpl"] = storage
	}
	if size := c.Cache.URLs; size > 0 {
		storage, err := lru.New(size)
		if err != nil {
			return err
		}
		c.Cache.Strorage["URL"] = storage
	}
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

// FromContext extracts the Config from Context.
func FromContext(ctx context.Context) (*Config, error) {
	c, ok := ctx.Value(configKey).(*Config)
	if !ok {
		return nil, errors.New("not found context config")
	}
	return c, nil
}
