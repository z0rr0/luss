// Copyright 2016 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package api contains API methods.
package api

import (
	"context"
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
)

const (
	// Version is API version.
	Version = "0.1.0"
)

var (
	// ErrEmptyRequest is error when there is no valid request data.
	ErrEmptyRequest = errors.New("empty request")
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
	Err      string `json:"error"`
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

// importRequestItem is a data item of import request.
type importRequestItem struct {
	Original string `json:"url"`
	Short    string `json:"short"`
}

// importResponseItem is a result item in import response.
type importResponseItem struct {
	Short string `json:"short"`
	Err   string `json:"error"`
}

// importResponse is a response for import request.
type importResponse struct {
	Err    int                  `json:"errcode"`
	Msg    string               `json:"msg"`
	Result []importResponseItem `json:"result"`
}

// exportRequest is a data item of export request.
type exportRequest struct {
	Group  string    `json:"group"`
	Tag    string    `json:"tag"`
	Active bool      `json:"active"`
	Period [2]string `json:"period"`
	Page   int       `json:"page"`
}

// exportResponseItem is a result item in export response.
type exportResponseItem struct {
	ID       string `json:"id"`
	Short    string `json:"short"`
	Original string `json:"url"`
	Group    string `json:"group"`
	Tag      string `json:"tag"`
	Created  string `json:"created"`
}

// exportResponse is a response for export request.
type exportResponse struct {
	Err    int                  `json:"errcode"`
	Msg    string               `json:"msg"`
	Pages  [3]int               `json:"pages"`
	Result []exportResponseItem `json:"result"`
}

// parsePeriod parses period string dates.
func (e *exportRequest) parsePeriod() ([2]*time.Time, error) {
	const layout = "2006-01-02"
	var result [2]*time.Time
	for i, v := range e.Period {
		if v != "" {
			t, err := time.Parse(layout, v)
			if err != nil {
				return result, err
			}
			result[i] = &t
		}
	}
	return result, nil
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
			IsAPI:     true,
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
			c.L.Debug.Printf("invalid short URL [%v] was skipped: %v", link, err)
			continue
		}
		if l, ok := trim.IsShort(link); ok {
			links = append(links, l)
		}
	}
	if len(links) == 0 {
		return core.ErrHandler{Err: ErrEmptyRequest, Status: http.StatusNoContent}
	}
	cus, err := trim.MultiLengthen(ctx, links)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	items := make([]addResponseItem, len(cus))
	for i, cu := range cus {
		id := cu.Cu.String()
		items[i] = addResponseItem{
			ID:       id,
			Short:    c.Address(id),
			Original: cu.Cu.Original,
			Err:      cu.Err,
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
		return core.ErrHandler{Err: ErrEmptyRequest, Status: http.StatusNoContent}
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
		return core.ErrHandler{Err: ErrEmptyRequest, Status: http.StatusNoContent}
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
		return core.ErrHandler{Err: ErrEmptyRequest, Status: http.StatusNoContent}
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

// HandlerImport imports predefined short URLs.
func HandlerImport(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	var imprs []importRequestItem
	user, err := auth.ExtractUser(ctx)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	if !user.HasRole("admin") {
		return core.ErrHandler{Err: errors.New("permissions error"), Status: http.StatusForbidden}
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&imprs)
	if (err != nil) && (err != io.EOF) {
		return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
	}
	n := len(imprs)
	if n == 0 {
		return core.ErrHandler{Err: ErrEmptyRequest, Status: http.StatusNoContent}
	}
	links := make(map[string]*trim.ReqParams, n)
	for _, impr := range imprs {
		params := &trim.ReqParams{
			Original:  impr.Original,
			Tag:       "",
			NotDirect: false,
			IsAPI:     true,
			TTL:       nil,
			Group:     "",
			Cb:        trim.CallBack{},
		}
		err = params.Valid()
		if err != nil {
			return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
		}
		links[impr.Short] = params
	}
	cus, err := trim.Import(ctx, links)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	items := make([]importResponseItem, len(cus))
	for i, cu := range cus {
		if cu.Err != "" {
			items[i] = importResponseItem{Err: cu.Err}
		} else {
			items[i] = importResponseItem{Short: cu.Cu.String()}
		}
	}
	result := &importResponse{
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

// HandlerExport exports URLs data.
func HandlerExport(ctx context.Context, w http.ResponseWriter, r *http.Request) core.ErrHandler {
	const (
		layout   = "2006-01-02"
		pageSize = 1000
	)
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

	exp := &exportRequest{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(exp)
	if (err != nil) && (err != io.EOF) {
		return core.ErrHandler{Err: err, Status: http.StatusBadRequest}
	}
	period, err := exp.parsePeriod()
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	filter := trim.Filter{
		Group:    exp.Group,
		Tag:      exp.Tag,
		Period:   period,
		Active:   exp.Active,
		Page:     exp.Page,
		PageSize: pageSize,
	}
	cus, pages, err := trim.Export(ctx, filter)
	if err != nil {
		return core.ErrHandler{Err: err, Status: http.StatusInternalServerError}
	}
	items := make([]exportResponseItem, len(cus))
	for i, cu := range cus {
		id := cu.String()
		items[i] = exportResponseItem{
			ID:       id,
			Short:    c.Address(id),
			Original: cu.Original,
			Group:    cu.Group,
			Tag:      cu.Tag,
			Created:  cu.Created.UTC().Format(layout),
		}
	}
	result := &exportResponse{
		Err:    0,
		Msg:    "ok",
		Pages:  pages,
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
