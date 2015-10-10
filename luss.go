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
    encode := flag.Int64("encode", 0, "encode numeric value to string")
    decode := flag.String("decode", "", "decode string value to numeric one")
    flag.Parse()
    if *version {
        fmt.Printf("%v: %v\n\trevision: %v\n\tbuild date: %v\n", Name, Version, Revision, BuildDate)
        return
    }
    // max int64 9223372036854775807 => AzL8n0Y58m7
    // real, max decode/encode 839299365868340223 <=> zzzzzzzzzz
    isDecoded := regexp.MustCompile(fmt.Sprintf("^[%s]{1,10}$", trim.Alphabet))
    isShortUrl := regexp.MustCompile(fmt.Sprintf("^/[%s]{1,10}$", trim.Alphabet))
    // CLI commands
    switch {
    case *admin != "":
        cf, err := utils.InitFileConfig(*config, *debug)
        if err != nil {
            utils.LoggerError.Panicf("init config error [%v]", err)
        }
        if u, err := prj.CreateAdmin(*admin, cf); err != nil {
            utils.LoggerError.Panicf("create admin error [%v]\n\tProbably this user already exists.", err)
        } else {
            fmt.Printf("Administrator is created:\n\tname: %v\n\ttoken: %v\n", u.Name, u.Secret)
        }
        return
    case *encode > 0:
        // max 9223372036854775807 => AzL8n0Y58m7
        fmt.Printf("encoding:\n\t%v => %v\n", *encode, trim.Encode(*encode))
        return
    case *decode != "":
        if !isDecoded.MatchString(*decode) {
            fmt.Printf("ERROR: incorrect 'decode' value: %v\n", *decode)
        }
        if dv, derr := trim.Decode(*decode); derr != nil {
            fmt.Printf("ERROR: incorrect 'decode' value: %v\n", *decode)
        } else {
            fmt.Printf("decoding:\n\t%v => %v\n", *decode, dv)
        }
        return
    }
    // configuration initialization
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
        // try to find a service handler
        f, ok := handlers[url]
        if ok {
            code, msg := f(w, r)
            if code != http.StatusOK {
                http.Error(w, msg, code)
            }
            return
        }
        if isShortUrl.MatchString(url) {
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
