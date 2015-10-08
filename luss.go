// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package main implements main methods of LUSS service.
package main

import (
    "flag"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "regexp"
    "strings"
    "syscall"
    "time"

    "github.com/z0rr0/luss/httph"
    "github.com/z0rr0/luss/prj"
    "github.com/z0rr0/luss/trim"
    "github.com/z0rr0/luss/utils"
)

const (
    // Name is a programm name
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

func interrupt() error {
    c := make(chan os.Signal)
    signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
    return fmt.Errorf("%v", <-c)
}

// HandlerTest is a response for test request.
func HandlerTest(w http.ResponseWriter, r *http.Request) (int, string) {
    fmt.Fprintf(w, "%v: %v %v\n", Name, Version, Revision)
    q := r.URL.Query()
    if (q.Get("write") == "yes") && (utils.Mode == utils.DebugMode) {
        if err := httph.TestWrite(utils.Cfg.Conf); err != nil {
            fmt.Fprintln(w, "data is not written")
        } else {
            fmt.Fprintln(w, "data is written")
        }
    }
    return http.StatusOK, ""
}

func main() {
    var err error
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("abnormal termination [%v]: %v\n", Version, r)
        }
    }()
    debug := flag.Bool("debug", false, "debug mode")
    version := flag.Bool("version", false, "show version")
    config := flag.String("config", Config, "configuration file")
    admin := flag.String("admin", "", "add new admin user")
    flag.Parse()
    if *version {
        fmt.Printf("%v: %v\n\trevision: %v\n\tbuild date: %v\n", Name, Version, Revision, BuildDate)
        return
    }
    // configuration initialization
    if *admin != "" {
        cf, err := utils.InitFileConfig(*config, *debug)
        if err != nil {
            utils.LoggerError.Panicf("init config error [%v]", err)
        }
        if u, err := prj.CreateAdmin(*admin, cf); err != nil {
            utils.LoggerError.Panicf("create admin error [%v]\n\tProbably this user already exists.", err)
        } else {
            utils.LoggerInfo.Printf("Administrator is created:\n\tname: %v\n\ttoken: %v\n", u.Name, u.Secret)
        }
        return
    }
    err = utils.InitConfig(*config, *debug)
    if err != nil {
        utils.LoggerError.Panicf("init config error [%v]", err)
    }
    errc := make(chan error)
    go func() {
        errc <- interrupt()
    }()
    listener := fmt.Sprintf("%v:%v", utils.Cfg.Conf.Listener.Host, utils.Cfg.Conf.Listener.Port)
    utils.LoggerInfo.Printf("%v running: version=%v [%v %v]\n Listen: %v", Name, Version, Revision, BuildDate, listener)
    server := &http.Server{
        Addr:           listener,
        Handler:        http.DefaultServeMux,
        ReadTimeout:    time.Duration(utils.Cfg.Conf.Listener.Timeout) * time.Second,
        WriteTimeout:   time.Duration(utils.Cfg.Conf.Listener.Timeout) * time.Second,
        MaxHeaderBytes: 1 << 20,
        ErrorLog:       utils.LoggerError,
    }

    isUrl := regexp.MustCompile(fmt.Sprintf("^/[%s]+$", trim.Alphabet))
    // TODO: do all handlers
    handlers := map[string]func(w http.ResponseWriter, r *http.Request) (int, string){
        "/test/t":   HandlerTest,
        "/add/link": httph.HandlerAddLink,
        "/add/json": httph.HandlerAddJSON,
        // "/p/add" - POST name+email -> confrim+admin
        // "/p/edit" - PUT users+roles
        // "/p/del" - DELETE del+remove links?
        // "/p/stat" - GET stats: day1-day2
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        url := strings.TrimRight(r.URL.Path, "/")
        start, code := time.Now().UTC(), http.StatusOK
        defer func() {
            utils.LoggerInfo.Printf("%v  %v\t%v", code, time.Now().Sub(start), url)
        }()
        // try to find service handler
        f, ok := handlers[url]
        if ok {
            code, msg := f(w, r)
            if code != http.StatusOK {
                http.Error(w, msg, code)
            }
            // code = s
            return
        }
        if isUrl.MatchString(url) {
            // fmt.Fprintln(w, "call url handler")
            link, err := httph.HandlerRedirect(strings.TrimLeft(url, "/"), r)
            if err == nil {
                code = http.StatusFound
                http.Redirect(w, r, link, code)
                return
            }
        }
        code = http.StatusNotFound
        http.NotFound(w, r)
    })

    // run server
    go func() {
        errc <- server.ListenAndServe()
    }()
    utils.LoggerInfo.Printf("%v termination, reason[%v]: %v[%v]\n", Name, <-errc, Version, Revision)
}
