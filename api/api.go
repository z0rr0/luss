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
	"time"

	"github.com/z0rr0/luss/auth"
	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/core"
	"github.com/z0rr0/luss/trim"
	"golang.org/x/net/context"
)

const (
	// Version is API version.
	Version = "0.0.1"
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

// getRequest is JSON API get request data.
type getRequest struct {
	Short string `json:"short"`
}

// userAddRequest is request data of new user.
type userAddRequest struct {
	Name string `json:"name"`
}

// userAddResponseItem is info about user creation result.
type userAddResponseItem struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	Err   string `json:"error"`
}

// userAddResponse is a response for user add request.
type userAddResponse struct {
	Err    int                   `json:"errcode"`
	Msg    string                `json:"msg"`
	Result []userAddResponseItem `json:"result"`
}

// userDelResponseItem is info about user delete result.
type userDelResponseItem struct {
	Name string `json:"name"`
	Err  string `json:"error"`
}

// userDelResponse is a response for users delete request.
type userDelResponse struct {
	Err    int                   `json:"errcode"`
	Msg    string                `json:"msg"`
	Result []userDelResponseItem `json:"result"`
}

// infoResponseItem is a result item in info response.
type infoResponseItem struct {
	Version  string `json:"version"`
	AuthOk   bool   `json:"authok"`
	PackSize int    `json:"pack_size"`
}

// infoResponse is a response for info request.
type infoResponse struct {
	Err    int                `json:"errcode"`
	Msg    string             `json:"msg"`
	Result []infoResponseItem `json:"result"`
}

// HandlerError returns JSON API response about the error.
func HandlerError(w http.ResponseWriter, code int) error {
	resp := shortResponse{Err: code, Msg: http.StatusText(code), Result: []bool{}}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
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
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	defer r.Body.Close()
	params, err := validateAddParams(r)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
	}
	cus, err := trim.Shorten(ctx, params)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
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
	b, err := json.Marshal(result)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", b)
	return core.ErrHandler{Err: nil, Status: http.StatusOK}
}

// HandlerGet returns info about short URLs.
func HandlerGet(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	var grs []getRequest
	c, err := conf.FromContext(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&grs)
	if (err != nil) && (err != io.EOF) {
		return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
	}
	links := []string{}
	for i := range grs {
		link, err := core.TrimAddress(grs[i].Short)
		if err != nil {
			c.L.Error.Printf("invalid short url [%v]: %v", link, err)
			continue
		}
		if l, ok := trim.IsShort(link); ok {
			links = append(links, l)
		}
	}
	if len(links) == 0 {
		if err := HandlerError(w, http.StatusOK); err != nil {
			c.L.Error.Println(err)
		}
		c.L.Debug.Println("empty request")
		return core.ErrHandler{Err: nil, Status: http.StatusOK}
	}
	cus, err := trim.MultiLengthen(ctx, links)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
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
	b, err := json.Marshal(result)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", b)
	return core.ErrHandler{Err: nil, Status: http.StatusOK}
}

// HandlerUserAdd creates new user.
func HandlerUserAdd(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	var uar []userAddRequest
	c, err := conf.FromContext(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	user, err := auth.ExtractUser(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	if !user.HasRole("admin") {
		return core.ErrHandler{Err: errors.New("permissions error"), Status: http.StatusForbidden}
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&uar)
	if (err != nil) && (err != io.EOF) {
		return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
	}
	if len(uar) == 0 {
		if err := HandlerError(w, http.StatusOK); err != nil {
			c.L.Error.Println(err)
		}
		c.L.Debug.Println("empty request")
		return core.ErrHandler{Err: nil, Status: http.StatusOK}
	}
	names := make([]string, len(uar))
	for i, v := range uar {
		names[i] = v.Name
	}
	usersResult, err := auth.CreateUsers(ctx, names)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	items := make([]userAddResponseItem, len(usersResult))
	for i, ur := range usersResult {
		if ur.Err != "" {
			items[i] = userAddResponseItem{
				Name:  ur.Name,
				Token: "",
				Err:   ur.Err,
			}
		} else {
			items[i] = userAddResponseItem{
				Name:  ur.U.Name,
				Token: ur.T,
				Err:   "",
			}
		}
	}
	result := &userAddResponse{
		Err:    0,
		Msg:    "ok",
		Result: items,
	}
	b, err := json.Marshal(result)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", b)
	return core.ErrHandler{Err: nil, Status: http.StatusOK}
}

// HandlerPwd updates user's token.
func HandlerPwd(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	var uar []userAddRequest
	c, err := conf.FromContext(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	user, err := auth.ExtractUser(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	if !user.HasRole("admin") && !user.HasRole("user") {
		return core.ErrHandler{Err: errors.New("permissions error"), Status: http.StatusForbidden}
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&uar)
	if (err != nil) && (err != io.EOF) {
		return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
	}
	if len(uar) == 0 {
		if err := HandlerError(w, http.StatusOK); err != nil {
			c.L.Error.Println(err)
		}
		c.L.Debug.Println("empty request")
		return core.ErrHandler{Err: nil, Status: http.StatusOK}
	}
	names := make([]string, len(uar))
	for i, v := range uar {
		names[i] = v.Name
	}
	usersResult, err := auth.ChangeUsers(ctx, names)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	items := make([]userAddResponseItem, len(usersResult))
	for i, ur := range usersResult {
		items[i] = userAddResponseItem{
			Name:  ur.Name,
			Token: ur.T,
			Err:   "",
		}
	}
	result := &userAddResponse{
		Err:    0,
		Msg:    "ok",
		Result: items,
	}
	b, err := json.Marshal(result)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", b)
	return core.ErrHandler{Err: nil, Status: http.StatusOK}
}

// HandlerUserDel disables user.
func HandlerUserDel(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	var uar []userAddRequest
	c, err := conf.FromContext(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	user, err := auth.ExtractUser(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	if !user.HasRole("admin") {
		return core.ErrHandler{Err: errors.New("permissions error"), Status: http.StatusForbidden}
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&uar)
	if (err != nil) && (err != io.EOF) {
		return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
	}
	if len(uar) == 0 {
		if err := HandlerError(w, http.StatusOK); err != nil {
			c.L.Error.Println(err)
		}
		c.L.Debug.Println("empty request")
		return core.ErrHandler{Err: nil, Status: http.StatusOK}
	}
	names := make([]string, len(uar))
	for i, v := range uar {
		names[i] = v.Name
	}
	usersResult, err := auth.DisableUsers(ctx, names)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	items := make([]userDelResponseItem, len(usersResult))
	for i, ur := range usersResult {
		items[i] = userDelResponseItem{
			Name: ur.Name,
			Err:  ur.Err,
		}
	}
	result := &userDelResponse{
		Err:    0,
		Msg:    "ok",
		Result: items,
	}
	b, err := json.Marshal(result)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", b)
	return core.ErrHandler{Err: nil, Status: http.StatusOK}
}

// HandlerInfo returns main API info.
func HandlerInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	c, err := conf.FromContext(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	user, err := auth.ExtractUser(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	result := &infoResponse{
		Err: 0,
		Msg: "ok",
		Result: []infoResponseItem{
			infoResponseItem{
				Version:  Version,
				AuthOk:   !user.IsAnonymous(),
				PackSize: c.Settings.MaxPack,
			},
		},
	}
	b, err := json.Marshal(result)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", b)
	return core.ErrHandler{Err: nil, Status: http.StatusOK}
}
