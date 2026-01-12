// /home/krylon/go/src/github.com/blicero/guangng/blacklist/blacklist_addr.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-12 14:59:10 krylon>

package blacklist

import (
	"net"
	"sort"
	"sync"
	"sync/atomic"
)

// BlacklistItemAddr is an item in the BlacklistAddr to match an IP address against
// an IP network.
type BlacklistItemAddr struct {
	HitCount  atomic.Int32
	addrRange *net.IPNet
}

// NewAddrItem creates a new BlacklistItemAddr.
func NewAddrItem(network string) *BlacklistItemAddr {
	var (
		err  error
		item = new(BlacklistItemAddr)
	)

	if _, item.addrRange, err = net.ParseCIDR(network); err != nil {
		panic(err)
	}

	return item
} // func NewAddrItem(network string) *BlacklistItemAddr

// Match returns true if the given address is in the Item's network.
func (i *BlacklistItemAddr) Match(addr net.IP) bool {
	if i.addrRange.Contains(addr) {
		i.HitCount.Add(1)
		return true
	}

	return false
} // func (i *BlacklistItemAddr) Match(addr net.IP) bool

// AddrItemList is mainly a helper type for sorting.
type AddrItemList []*BlacklistItemAddr

func (al AddrItemList) Len() int      { return len(al) }
func (al AddrItemList) Swap(i, j int) { al[i], al[j] = al[j], al[i] }
func (al AddrItemList) Less(i, j int) bool {
	return al[i].HitCount.Load() < al[j].HitCount.Load()
}

// BlacklistAddr is a list of BlacklistItemAddr and a Mutex.
type BlacklistAddr struct {
	items AddrItemList
	lock  sync.Mutex
}

// NewBlacklistAddr create a new AddrBlacklist from the default network list.
func NewBlacklistAddr() *BlacklistAddr {
	var al = &BlacklistAddr{
		items: make(AddrItemList, len(defaultNetworks)),
	}

	for i, n := range defaultNetworks {
		al.items[i] = NewAddrItem(n)
	}

	return al
} // func NewBlacklistAddr() *AddrBlacklist

// Match checks if the given address is in any of the Blacklist's networks.
func (al *BlacklistAddr) Match(addr net.IP) bool {
	for _, item := range al.items {
		if item.Match(addr) {
			al.lock.Lock()
			sort.Sort(al.items)
			al.lock.Unlock()
			return true
		}
	}

	return false
} // func (al *AddrBlacklist) Match(addr net.IP) bool
