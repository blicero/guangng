// /home/krylon/go/src/github.com/blicero/guangng/blacklist/blacklist.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-12 14:40:21 krylon>

package blacklist

import (
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
)

type NameBlacklistItem struct {
	HitCount atomic.Int32
	pattern  *regexp.Regexp
}

func NewNameItem(pattern string) *NameBlacklistItem {
	var item = &NameBlacklistItem{
		pattern: regexp.MustCompile(pattern),
	}

	return item
} // func NewNameItem(pattern string) *NameBlacklistItem

func (i *NameBlacklistItem) Match(name string) bool {
	if i.pattern.MatchString(name) {
		i.HitCount.Add(1)
		return true
	}

	return false
} // func (i *NameBlacklistItem) Match(name string) bool

type NameItemList []*NameBlacklistItem

func (nl NameItemList) Len() int      { return len(nl) }
func (nl NameItemList) Swap(i, j int) { nl[i], nl[j] = nl[j], nl[i] }
func (nl NameItemList) Less(i, j int) bool {
	return nl[i].HitCount.Load() < nl[j].HitCount.Load()
} // func (nl NameItemList) Less(i, j int) bool

type NameBlacklist struct {
	items []*NameBlacklistItem
	lock  sync.Mutex
}

func NewNameBlacklist() *NameBlacklist {
	var list = &NameBlacklist{
		items: make([]*NameBlacklistItem, len(defaultNamePatterns)),
	}

	for i, p := range defaultNamePatterns {
		list.items[i] = NewNameItem(p)
	}

	return list
} // func NewNameBlacklist() *NameBlacklist

func (bl *NameBlacklist) Match(name string) bool {
	for _, i := range bl.items {
		if status := i.Match(name); status {
			bl.lock.Lock()
			sort.Sort(NameItemList(bl.items))
			bl.lock.Unlock()
			return true
		}
	}

	return false
} // func (bl *NameBlacklist) Match(name string) bool
