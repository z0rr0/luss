// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package lru implements LRU cache methods to speed up search already saved short URLs.
//
// It is based on github.com/golang/groupcache
// and github.com/hashicorp/golang-lru,  but more simple and uses only string keys and values.
package lru

import (
    "container/list"
    "sync"
)

// Cache is an LRU cache. It is thread-safe for concurrent access.
type Cache struct {
    // maxEntries is the maximum number of cache entries.
    maxEntries int
    ll         *list.List
    cache      map[string]*list.Element
    m          sync.RWMutex
}

// entry is cache entry.
type entry struct {
    key   string
    value string
}

// New initializes new Cache storage.
func New(size int) *Cache {
    var c map[string]*list.Element
    if size > 0 {
        c = make(map[string]*list.Element, size+1)
    }
    return &Cache{maxEntries: size, ll: list.New(), cache: c}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value string) {
    if c.maxEntries < 1 {
        return
    }
    c.m.Lock()
    defer c.m.Unlock()

    if e, ok := c.cache[key]; ok {
        c.ll.MoveToFront(e)
        e.Value.(*entry).value = value
        return
    }
    e := c.ll.PushFront(&entry{key, value})
    c.cache[key] = e
    if c.ll.Len() > c.maxEntries {
        // remove oldest item
        old := c.ll.Back()
        if old != nil {
            c.removeElement(old)
        }
    }
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key string) {
    if c.maxEntries < 1 {
        return
    }
    c.m.Lock()
    defer c.m.Unlock()
    if e, ok := c.cache[key]; ok {
        c.removeElement(e)
    }
}

// removeElement the element from the cache.
func (c *Cache) removeElement(e *list.Element) {
    c.ll.Remove(e)
    en := e.Value.(*entry)
    delete(c.cache, en.key)
}

// Get a value from the cache by its key.
func (c *Cache) Get(key string) (string, bool) {
    if c.maxEntries < 1 {
        return "", false
    }
    c.m.Lock()
    defer c.m.Unlock()
    if e, ok := c.cache[key]; ok {
        c.ll.MoveToFront(e)
        return e.Value.(*entry).value, true
    }
    return "", false
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
    if c.maxEntries < 1 {
        return 0
    }
    c.m.RLock()
    defer c.m.RUnlock()
    return c.ll.Len()
}
