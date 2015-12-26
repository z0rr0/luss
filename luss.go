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
	"strings"
	"syscall"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/core"
	"github.com/z0rr0/luss/db"
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
	Method string
}

func interrupt() error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	return fmt.Errorf("%v", <-c)
}

// HandlerTest is a response for test request.
// func HandlerTest(w http.ResponseWriter, r *http.Request) (int, string) {
//     fmt.Fprintf(w, "%v: %v %v\n", Name, Version, Revision)
//     q := r.URL.Query()
//     if (q.Get("write") == "yes") && (utils.Mode == utils.DebugMode) {
//         if err := httph.TestWrite(utils.Cfg.Conf); err != nil {
//             fmt.Fprintln(w, "data is not written")
//         } else {
//             fmt.Fprintln(w, "data is written")
//         }
//     }
//     return http.StatusOK, ""
// }

func main() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("abnormal termination [%v]: %v\n", Version, r)
		}
	}()
	version := flag.Bool("version", false, "show version")
	config := flag.String("config", Config, "configuration file")

	// admin := flag.String("admin", "", "add new admin user")
	// encode := flag.Int64("encode", 0, "encode numeric value to string")
	// decode := flag.String("decode", "", "decode string value to numeric one")
	flag.Parse()
	if *version {
		fmt.Printf("%v: %v\n\trevision: %v\n\tbuild date: %v\n", Name, Version, Revision, BuildDate)
		return
	}
	// max int64 9223372036854775807 => AzL8n0Y58m7
	// real, max decode/encode 839299365868340223 <=> zzzzzzzzzz
	// isDecoded := regexp.MustCompile(fmt.Sprintf("^[%s]{1,10}$", trim.Alphabet))
	// isShortUrl := regexp.MustCompile(fmt.Sprintf("^/[%s]{1,10}$", trim.Alphabet))
	// CLI commands
	// switch {
	// case *admin != "":
	//     cf, err := utils.InitFileConfig(*config, *debug)
	//     if err != nil {
	//         utils.LoggerError.Panicf("init config error [%v]", err)
	//     }
	//     if u, err := prj.CreateAdmin(*admin, cf); err != nil {
	//         utils.LoggerError.Panicf("create admin error [%v]\n\tProbably this user already exists.", err)
	//     } else {
	//         fmt.Printf("Administrator is created:\n\tname: %v\n\ttoken: %v\n", u.Name, u.Secret)
	//     }
	//     return
	// case *encode > 0:
	//     // max 9223372036854775807 => AzL8n0Y58m7
	//     fmt.Printf("encoding:\n\t%v => %v\n", *encode, trim.Encode(*encode))
	//     return
	// case *decode != "":
	//     if !isDecoded.MatchString(*decode) {
	//         fmt.Printf("ERROR: incorrect 'decode' value: %v\n", *decode)
	//     }
	//     if dv, derr := trim.Decode(*decode); derr != nil {
	//         fmt.Printf("ERROR: incorrect 'decode' value: %v\n", *decode)
	//     } else {
	//         fmt.Printf("decoding:\n\t%v => %v\n", *decode, dv)
	//     }
	//     return
	// }
	// configuration initialization
	cfg, err := conf.Parse(*config)
	if err != nil {
		log.Panicf("init config error [%v]", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Panicf("config validate error [%v]", err)
	}
	errc := make(chan error)
	go func() {
		errc <- interrupt()
	}()
	// check db connection
	go func() {
		s, err := db.NewSession(cfg.Conn, true)
		if err != nil {
			errc <- err
		}
		s.Close()
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
	// TODO: do all handlers
	handlers := map[string]Handler{
		"/test/t": Handler{F: core.HandlerTest, Auth: false, Method: "GET"},
		// "/add/link": httph.HandlerAddLink,
		// "/add/json": httph.HandlerAddJSON,
		// "/p/add" - POST name+email -> confrim+admin
		// "/p/edit" - PUT users+roles
		// "/p/del" - DELETE del+remove links?
		// "/p/stat" - GET stats: day1-day2
	}
	// // TODO: export/import
	baseCtx := conf.NewContext(cfg)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var err error
		url := "/"
		if r.URL.Path != url {
			url = strings.TrimRight(r.URL.Path, "/")
		}
		start, code := time.Now(), http.StatusOK
		defer func() {
			cfg.L.Info.Printf("%v  %v\t%v", code, time.Since(start), url)
		}()
		ctx, cancel := context.WithCancel(baseCtx)
		defer cancel()
		s, err := db.NewSession(cfg.Conn, false)
		if err != nil {
			code = http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
			return
		}
		defer s.Close()
		ctx = db.NewContext(ctx, s)
		// try to find a service handler
		rh, ok := handlers[url]
		if ok {
			if (rh.Method != "ANY") && (rh.Method != r.Method) {
				code = http.StatusMethodNotAllowed
				http.Error(w, http.StatusText(code), code)
				return
			}
			err = rh.F(ctx, w, r)
			if err != nil {
				code = http.StatusInternalServerError
				http.Error(w, http.StatusText(code), code)
				return
			}
		}

		// if ok {
		// 	code, msg := f(w, r)
		// 	if code != http.StatusOK {
		// 		http.Error(w, msg, code)
		// 	}
		// 	return
		// }
		// if isShortUrl.MatchString(url) {
		// 	link, err := httph.HandlerRedirect(strings.TrimLeft(url, "/"), r)
		// 	if err == nil {
		// 		code = http.StatusFound
		// 		http.Redirect(w, r, link, code)
		// 		return
		// 	}
		// }
		// code = http.StatusNotFound
		// http.NotFound(w, r)
	})

	// run server
	go func() {
		errc <- server.ListenAndServe()
	}()
	cfg.L.Info.Printf("%v termination, reason[%v]: %v [%v]\n", Name, <-errc, Version, Revision)
}
