// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package main implements main methods of LUSS service.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/z0rr0/luss/api"
	"github.com/z0rr0/luss/auth"
	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/core"
	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/trim"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
)

const (
	// Name is a program name
	Name = "LUSS"
	// Config is default configuration file name
	Config = "config.json"
)

var (
	// Version is LUSS version
	Version = ""
	// Revision is revision number
	Revision = ""
	// BuildDate is build date
	BuildDate = ""
)

// Handler is a struct to check and handle incoming HTTP request.
type Handler struct {
	F      func(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler
	Auth   bool
	API    bool
	Method string
}

func interrupt() error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	return fmt.Errorf("%v", <-c)
}

func main() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("abnormal termination [%v]: %v\n", Version, r)
		}
	}()
	version := flag.Bool("version", false, "show version")
	config := flag.String("config", Config, "configuration file")
	flag.Parse()
	if *version {
		fmt.Printf("%v: %v\n\trevision: %v\n\tbuild date: %v\n", Name, Version, Revision, BuildDate)
		return
	}
	// configuration initialization
	cfg, err := conf.Parse(*config)
	if err != nil {
		log.Panicf("init config error [%v]", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Panicf("config validate error [%v]", err)
	}
	// check db connection
	s, err := db.NewSession(cfg.Conn, true)
	if err != nil {
		log.Panic(err)
	}
	s.Close()
	defer cfg.Close()
	// init users
	if err := auth.InitUsers(cfg); err != nil {
		log.Panic(err)
	}
	// set init context
	mainCtx := conf.NewContext(cfg)
	mainCtx, err = core.RunWorkers(mainCtx)
	if err != nil {
		log.Panic(err)
	}
	errc := make(chan error)
	go func() {
		errc <- interrupt()
	}()
	listener := net.JoinHostPort(cfg.Listener.Host, fmt.Sprint(cfg.Listener.Port))
	cfg.L.Info.Printf("%v running (debug=%v):\n\tlisten: %v\n\tversion=%v [%v %v]", Name, cfg.Debug, listener, Version, Revision, BuildDate)
	server := &http.Server{
		Addr:           listener,
		Handler:        http.DefaultServeMux,
		ReadTimeout:    time.Duration(cfg.Listener.Timeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Listener.Timeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       cfg.L.Error,
	}
	// static files
	staticDir, _ := cfg.StaticDir()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	// keys should not match to trim.IsShortURL pattern (short URLs set)
	handlers := map[string]Handler{
		"/":              Handler{F: core.HandlerIndex, Auth: false, API: false, Method: "ANY"},
		"/test/t":        Handler{F: core.HandlerTest, Auth: false, API: false, Method: "ANY"},
		"/error/notfoud": Handler{F: core.HandlerNotFound, Auth: false, API: false, Method: "GET"},
		"/error/common":  Handler{F: core.HandlerError, Auth: false, API: false, Method: "GET"},
		"/api/add":       Handler{F: api.HandlerAdd, Auth: false, API: true, Method: "POST"},
		"/api/get":       Handler{F: api.HandlerGet, Auth: false, API: true, Method: "POST"},
		// /api/user/add
		// /api/user/pwd
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := "/"
		if r.URL.Path != path {
			path = strings.TrimRight(r.URL.Path, "/")
		}
		start, code, isAPI := time.Now(), http.StatusOK, false
		ctx, cancel := context.WithCancel(mainCtx)
		defer func() {
			cancel()
			switch {
			case code == http.StatusNotFound && !isAPI:
				core.HandlerNotFound(ctx, w, r)
			case code != http.StatusOK && !isAPI:
				core.HandlerError(ctx, w, r)
			case code != http.StatusOK:
				if err := api.HandlerError(w, code); err != nil {
					cfg.L.Error.Println(err)
				}
			}
			cfg.L.Info.Printf("%-5v %v\t%-12v\t%v", r.Method, code, time.Since(start), path)
		}()
		rh, ok := handlers[path]
		if ok {
			isAPI = rh.API
			if (rh.Method != "ANY") && (rh.Method != r.Method) {
				code = http.StatusMethodNotAllowed
				return
			}
			// pre-authentication: quickly check a token value
			ctx, err := auth.CheckToken(ctx, r, isAPI)
			// anonymous request should be allow/deny here
			authRequired := rh.Auth || !cfg.Settings.Anonymous
			if err != nil && (authRequired || err != auth.ErrAnonymous) {
				cfg.L.Debug.Printf("auth=%v, err=%v", rh.Auth, err)
				code = http.StatusUnauthorized
				return
			}
			// open database session
			s, err := db.NewSession(cfg.Conn, true)
			if err != nil {
				cfg.L.Error.Println(err)
				code = http.StatusInternalServerError
				return
			}
			defer s.Close()
			ctx = db.NewContext(ctx, s)
			// authentication
			ctx, err = auth.Authenticate(ctx)
			if err != nil {
				cfg.L.Error.Println(err)
				code = http.StatusUnauthorized
				return
			}
			// call a found handler
			if err := rh.F(ctx, w, r); err.Err != nil {
				cfg.L.Error.Println(err)
				code = err.Status
				return
			}
			return
		} else if link, ok := trim.IsShort(path); ok {
			if r.Method != "GET" {
				code = http.StatusMethodNotAllowed
				return
			}
			origURL, err := core.HandlerRedirect(ctx, link, r)
			switch {
			case err == nil:
				code = http.StatusFound
				http.Redirect(w, r, origURL, code)
			case err == mgo.ErrNotFound:
				code = http.StatusNotFound
				// http.NotFound(w, r)
			default:
				cfg.L.Error.Println(err)
				code = http.StatusInternalServerError
			}
			return
		}
		code = http.StatusNotFound
		// http.NotFound(w, r)
	})
	// run server
	go func() {
		errc <- server.ListenAndServe()
	}()
	cfg.L.Info.Printf("%v termination, reason[%v]: %v [%v]\n", Name, <-errc, Version, Revision)
}
