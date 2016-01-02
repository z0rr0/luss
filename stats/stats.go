// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package stats contains method to collect and return
// info about activities.
//
// It uses https://docs.mongodb.org/v3.0/reference/operator/aggregation-date/
// to collect different info about requests.
package stats

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/z0rr0/luss/conf"
	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/trim"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2/bson"
)

var (
	// logger is a logger for error messages
	logger = log.New(os.Stderr, "LOGGER [stats]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// GeoData is geographic information.
type GeoData struct {
	IP        string  `bson:"ip"`
	Country   string  `bson:"country"`
	City      string  `bson:"city"`
	Tz        string  `bson:"tz"`
	Latitude  float64 `bson:"lat"`
	Longitude float64 `bson:"lon"`
}

// Track is information about users requests.
type Track struct {
	ID      bson.ObjectId `bson:"_id"`
	Short   string        `bson:"short"`
	URL     string        `bson:"url"`
	Group   string        `bson:"group"`
	Tag     string        `bson:"tag"`
	Geo     GeoData       `bson:"geo"`
	Created time.Time     `bson:"ts"`
}

// Tracker saves info about short URL activities.
func Tracker(ctx context.Context, cu *trim.CustomURL, addr string) error {
	c, err := conf.FromContext(ctx)
	if err != nil {
		logger.Println(err)
		return err
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		logger.Println(err)
		return err
	}
	geo := GeoData{IP: host}
	record, err := c.GeoDB.City(net.ParseIP(host))
	if err != nil {
		logger.Println(err)
	} else {
		geo.Country = record.Country.Names["en"]
		geo.City = record.City.Names["en"]
		geo.Latitude = record.Location.Latitude
		geo.Longitude = record.Location.Longitude
		geo.Tz = record.Location.TimeZone
	}
	s, err := db.NewSession(c.Conn, true)
	if err != nil {
		logger.Println(err)
		return err
	}
	defer s.Close()
	coll, err := db.Coll(s, "tracks")
	if err != nil {
		logger.Println(err)
		return err
	}
	err = coll.Insert(bson.M{
		"short": trim.Encode(cu.ID),
		"url":   cu.Original,
		"group": cu.Group,
		"tag":   cu.Tag,
		"geo":   geo,
		"ts":    time.Now().UTC(),
	})
	if err != nil {
		logger.Println(err)
	}
	return err
}
