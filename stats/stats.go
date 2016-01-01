// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package stats contains method to collect and return
// info about activities.
package stats

import (
	"log"
	"os"
	"time"

	"github.com/z0rr0/luss/db"
	"github.com/z0rr0/luss/trim"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// logger is a logger for error messages
	logger = log.New(os.Stderr, "LOGGER [stats]: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// GeoData is geographic information.
type GeoData struct {
	IP        string `bson:"ip"`
	Country   string `bson:"country"`
	City      string `bson:"city"`
	Tz        string `bson:"tz"`
	Latitude  string `bson:"lat"`
	Longitude string `bson:"lon"`
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
func Tracker(s *mgo.Session, cu *trim.CustomURL) {
	coll, err := db.Coll(s, "tracker")
	if err != nil {
		logger.Println(err)
		return
	}
	err = coll.Insert(bson.M{
		"short": trim.Encode(cu.ID),
		"url":   cu.Original,
		"group": cu.Group,
		"tag":   cu.Tag,
		"geo":   GeoData{},
		"ts":    time.Now().UTC(),
	})
	if err != nil {
		logger.Println(err)
	}
}
