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
    "syscall"
    "time"

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

func main() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("abnormal termination [%v]: %v\n", Version, r)
        }
    }()
    debug := flag.Bool("debug", false, "debug mode")
    version := flag.Bool("version", false, "show version")
    config := flag.String("config", Config, "configuration file")
    flag.Parse()
    if *version {
        fmt.Printf("%v: %v\n\trevision: %v\n\tbuild date: %v\n", Name, Version, Revision, BuildDate)
        return
    }
    err := utils.InitConfig(*config, *debug)
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
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "%v running: version=%v [%v]\n Listen: %v", Name, Version, Revision, listener)
    })
    // run server
    go func() {
        errc <- server.ListenAndServe()
    }()
    utils.LoggerInfo.Printf("%v termination, reason[%v]: %v[%v]\n", Name, <-errc, Version, Revision)
}
