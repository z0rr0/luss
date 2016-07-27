// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package test contains additional methods for testing.
package test

import (
	"log"
	"os"
	"path/filepath"
)

const (
	// Config - test configuration name
	Config = "luss.json"
)

var (
	// LoggerError is a test logger.
	LoggerError = log.New(os.Stderr, "TEST [LUSS]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// TcBuildDir is used for the tests and returns a directory for the build.
func TcBuildDir() string {
	repoDir := []string{os.Getenv("GOPATH"), "src", "github.com", "z0rr0", "luss"}
	return filepath.Join(repoDir...)
}

// TcConfigName returns test configuration file name
func TcConfigName() string {
	return filepath.Join(os.Getenv("GOPATH"), Config)
}
