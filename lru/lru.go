// Copyright 2015 Alexander Zaytsev <thebestzorro@yandex.ru>
// Use of this source code is governed by a GPL-style
// license that can be found in the LICENSE file.

// Package lru implements LRU cache methods to speed up search already saved short URLs.
//
// It based on github.com/golang/groupcache
// and github.com/hashicorp/golang-lru,  but more simple.
package lru

import (
    "container/list"
    "sync"
)

// Key  is a type if caching key.
type Key string

// Value  is a type if caching value.
type Value string

// Cache is an LRU cache. It is thread-safe for concurrent access.
type Cache struct {
    // maxEntries is the maximum number of cache entries.
    maxEntries int
    ll         *list.List
    cache      map[Key]*list.Element
    m          sync.RWMutex
}

// entry is cache entry.
type entry struct {
    key   Key
    value Value
}

// New initializes new Cache storage.
func New(size int) *Cache {
    var c map[Key]*list.Element
    if size > 0 {
        c = make(map[Key]*list.Element, size+1)
    }
    return &Cache{maxEntries: size, ll: list.New(), cache: c}
}

// Add adds a value to the cache.
func (c *Cache) Add(key Key, value Value) {
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
func (c *Cache) Remove(key Key) {
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
    if c.maxEntries < 1 {
        return
    }
    c.ll.Remove(e)
    en := e.Value.(*entry)
    delete(c.cache, en.key)
}

// Get a value from the cache by its key.
func (c *Cache) Get(key Key) (Value, bool) {
    var empty Value
    if c.maxEntries < 1 {
        return empty, false
    }
    c.m.RLock()
    defer c.m.RUnlock()
    if e, ok := c.cache[key]; ok {
        c.ll.MoveToFront(e)
        return e.Value.(*entry).value, true
    }
    return empty, false
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
