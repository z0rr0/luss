// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package api contains API methods.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/core"
	"github.com/z0rr0/luss/trim"
	"golang.org/x/net/context"
)

// shortResponse is common HTTP response.
type shortResponse struct {
	Err    int    `json:"errcode"`
	Msg    string `json:"msg"`
	Result []bool `json:"result"`
}

// addRequest is JSON API add request callback data.
type addCbRequest struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

// addRequest is JSON API add request data.
type addRequest struct {
	URL       string       `json:"url"`
	Tag       string       `json:"tag"`
	TTL       uint64       `json:"ttl"`
	NotDirect bool         `json:"nd"`
	Group     string       `json:"group"`
	Cb        addCbRequest `json:"cb"`
}

// addResponseItem is a item of response for add request.
type addResponseItem struct {
	ID       string `json:"id"`
	Original string `json:"url"`
	Short    string `json:"short"`
}

// addResponse is a response for add request.
type addResponse struct {
	Err    int               `json:"errcode"`
	Msg    string            `json:"msg"`
	Result []addResponseItem `json:"result"`
}

// HandlerError returns JSON API response about the error.
func HandlerError(w http.ResponseWriter, code int) error {
	w.Header().Set("Content-Type", "application/json")
	resp := shortResponse{Err: code, Msg: http.StatusText(code), Result: []bool{}}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}

// validateParams checks HTTP parameters for add-request.
func validateAddParams(r *http.Request) ([]*trim.ReqParams, error) {
	var (
		ars []addRequest
		ttl *time.Time
	)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ars)
	if (err != nil) && (err != io.EOF) {
		return nil, err
	}
	n := len(ars)
	if n == 0 {
		return nil, errors.New("empty request")
	}
	now := time.Now()
	result := make([]*trim.ReqParams, n)
	for i, ar := range ars {
		if ar.TTL > 0 {
			expire := now.Add(time.Duration(ar.TTL) * time.Hour).UTC()
			ttl = &expire
		} else {
			ttl = nil
		}
		params := &trim.ReqParams{
			Original:  ar.URL,
			Tag:       ar.Tag,
			NotDirect: ar.NotDirect,
			TTL:       ttl,
			Group:     ar.Group,
			Cb: trim.CallBack{
				URL:    ar.Cb.URL,
				Method: ar.Cb.Method,
				Name:   ar.Cb.Name,
				Value:  ar.Cb.Value,
			},
		}
		err = params.Valid()
		if err != nil {
			return nil, err
		}
		result[i] = params
	}
	return result, nil
}

// HandlerAdd creates new short URL.
func HandlerAdd(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return core.ErrHandler{err, http.StatusInternalServerError}
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		return core.ErrHandler{fmt.Errorf("unsupported content-type: %v", ct), http.StatusBadRequest}
	}
	params, err := validateAddParams(r)
	if err != nil {
		return core.ErrHandler{err, http.StatusBadRequest}
	}
	cus, err := trim.Shorten(ctx, params)
	if err != nil {
		return core.ErrHandler{err, http.StatusInternalServerError}
	}
	items := make([]addResponseItem, len(cus))
	for i, cu := range cus {
		id := cu.String()
		items[i] = addResponseItem{
			ID:       id,
			Short:    c.Address(id),
			Original: cu.Original,
		}
	}
	result := &addResponse{
		Err:    0,
		Msg:    "ok",
		Result: items,
	}
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(result)
	if err != nil {
		return core.ErrHandler{err, http.StatusInternalServerError}
	}
	fmt.Fprintf(w, "%s", b)
	return core.ErrHandler{nil, http.StatusOK}
}
