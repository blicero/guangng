// /home/krylon/go/src/github.com/blicero/guangng/blacklist/blacklist.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-02-05 14:23:41 krylon>

package blacklist

import (
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
)

// BlacklistItemName is a pattern to match hostnames.
type BlacklistItemName struct {
	HitCount atomic.Int32
	pattern  *regexp.Regexp
}

// NewNameItem creates a new BlacklistItemName.
func NewNameItem(pattern string) *BlacklistItemName {
	var item = &BlacklistItemName{
		pattern: regexp.MustCompile(pattern),
	}

	return item
} // func NewNameItem(pattern string) *NameBlacklistItem

// Match returns true if the item's pattern matches the given hostname.
func (i *BlacklistItemName) Match(name string) bool {
	if i.pattern.MatchString(name) {
		i.HitCount.Add(1)
		return true
	}

	return false
} // func (i *NameBlacklistItem) Match(name string) bool

type NameItemList []*BlacklistItemName

func (nl NameItemList) Len() int      { return len(nl) }
func (nl NameItemList) Swap(i, j int) { nl[i], nl[j] = nl[j], nl[i] }
func (nl NameItemList) Less(i, j int) bool {
	return nl[i].HitCount.Load() < nl[j].HitCount.Load()
} // func (nl NameItemList) Less(i, j int) bool

type BlacklistName struct {
	items NameItemList
	lock  sync.RWMutex
}

func NewBlacklistName() *BlacklistName {
	var list = &BlacklistName{
		items: make(NameItemList, len(defaultNamePatterns)),
	}

	for i, p := range defaultNamePatterns {
		list.items[i] = NewNameItem(p)
	}

	return list
} // func NewNameBlacklist() *NameBlacklist

func (bl *BlacklistName) Match(name string) bool {
	bl.lock.RLock()
	for _, i := range bl.items {
		if status := i.Match(name); status {
			bl.lock.RUnlock()
			bl.lock.Lock()
			sort.Sort(bl.items)
			bl.lock.Unlock()
			return true
		}
	}

	bl.lock.RUnlock()
	return false
} // func (bl *NameBlacklist) Match(name string) bool
