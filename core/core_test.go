// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/z0rr0/luss/auth"
	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/test"
)

func TestHandlerTest(t *testing.T) {
	cfg, err := conf.Parse(test.TcConfigName())
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("invalid behavior")
	}
	ctx, cancel := context.WithCancel(conf.NewContext(cfg))
	defer cancel()

	r := &http.Request{}
	ctx, _ = auth.CheckToken(ctx, r, false)

	s, err := db.NewSession(cfg.Conn, true)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx = db.NewContext(ctx, s)

	ctx, err = auth.Authenticate(ctx)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	if err := HandlerTest(ctx, w, r); err.Err != nil {
		t.Error(err.Err)
	}
	if w.Code != http.StatusOK {
		t.Error("invalid behavior")
	}
}
