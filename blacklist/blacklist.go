// /home/krylon/go/src/github.com/blicero/guangng/blacklist/blacklist.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-11 18:21:49 krylon>

package blacklist

import (
	"regexp"
	"sync"
	"sync/atomic"
)

type BlacklistItem struct {
	HitCount atomic.Int32
}

type NameBlacklistItem struct {
	BlacklistItem
	pattern *regexp.Regexp
}

func (i *NameBlacklistItem) Match(name string) bool {
	if i.pattern.MatchString(name) {
		i.HitCount.Inc()
		return true
	}

	return false
} // func (i *NameBlacklistItem) Match(name string) bool

type NameBlacklist struct {
	items []NameBlacklistItem
	lock  sync.Mutex
}

func (bl *NameBlacklist) Match(name string) bool {
	for _, i := range bl.items {
		if status := i.Match(name); status {
			bl.lock.Lock()
		}
	}
}
