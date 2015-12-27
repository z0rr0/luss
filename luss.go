// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
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
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/core"
	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/project"
	"github.com/z0rr0/luss/trim"
	"golang.org/x/net/context"
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
	F      func(ctx context.Context, w http.ResponseWriter, r *http.Request) error
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
	// max int64 9223372036854775807 => AzL8n0Y58m7
	// real, max decode/encode 839299365868340223 <=> zzzzzzzzzz
	isShortURL := regexp.MustCompile(fmt.Sprintf("^/[%s]{1,10}$", trim.Alphabet))
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
	handlers := map[string]Handler{
		"/test/t": Handler{F: core.HandlerTest, Auth: false, API: false, Method: "GET"},
		// "/add/link"
		// "/api/add/"
		// "/api/add/json"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := "/"
		if r.URL.Path != url {
			url = strings.TrimRight(r.URL.Path, "/")
		}
		start, code := time.Now(), http.StatusOK
		defer func() {
			cfg.L.Info.Printf("%v  %v\t%v", code, time.Since(start), url)
		}()
		ctx, cancel := context.WithCancel(conf.NewContext(cfg))
		defer cancel()
		rh, ok := handlers[url]
		if ok {
			if (rh.Method != "ANY") && (rh.Method != r.Method) {
				code = http.StatusMethodNotAllowed
				http.Error(w, http.StatusText(code), code)
				return
			}
			// open database session
			s, err := db.NewSession(cfg.Conn, true)
			if err != nil {
				cfg.L.Error.Println(err)
				code = http.StatusMethodNotAllowed
				http.Error(w, http.StatusText(code), code)
				return
			}
			defer s.Close()
			ctx = db.NewContext(ctx, s)
			// authentication
			ctx, err := project.Authenticate(ctx, r)
			if err != nil {
				cfg.L.Error.Println(err)
				code = http.StatusMethodNotAllowed
				http.Error(w, http.StatusText(code), code)
				return
			}
			// call a found handler
			if err := rh.F(ctx, w, r); err != nil {
				cfg.L.Error.Println(err)
				code = http.StatusInternalServerError
				http.Error(w, http.StatusText(code), code)
				return
			}
			return
		} else if isShortURL.MatchString(url) {
			cfg.L.Debug.Println("short url")
			return
		}
		code = http.StatusNotFound
		http.NotFound(w, r)
	})
	// run server
	go func() {
		errc <- server.ListenAndServe()
	}()
	cfg.L.Info.Printf("%v termination, reason[%v]: %v [%v]\n", Name, <-errc, Version, Revision)
}
