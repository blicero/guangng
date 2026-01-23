// /home/krylon/go/src/github.com/blicero/guangng/database/03_database_host_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 23. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-23 16:00:50 krylon>

package database

import (
	"net"
	"testing"

	"github.com/blicero/guangng/model"
	"github.com/blicero/guangng/model/hsrc"
)

var tHosts map[int64]model.Host

func TestHostSource(t *testing.T) {
	for s := hsrc.Generator; s <= hsrc.User; s++ {
		t.Logf("HostSource.%s = %d",
			s,
			s)
	}
} // func TestHostSource(t *testing.T)

func TestHostAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	type hostAddTest struct {
		host model.Host
		fail bool
	}

	var testCases = []hostAddTest{
		{
			host: model.Host{
				Addr:   net.ParseIP("10.10.0.1"),
				Name:   "neuromancer.local",
				Source: hsrc.User,
			},
		},
		{
			host: model.Host{
				Addr:   net.ParseIP("fe80::2342"),
				Name:   "wintermute.local",
				Source: hsrc.User,
			},
		},
	}

	tHosts = make(map[int64]model.Host)

	for _, c := range testCases {
		if err := tdb.HostAdd(&c.host); err != nil {
			if !c.fail {
				t.Errorf("Failed to add Host %s (%s): %s",
					c.host.Name,
					c.host.Addr,
					err.Error())
			}
		} else if c.fail {
			t.Errorf("Adding Host %s (%s) should have failed but didn't",
				c.host.Name,
				c.host.AStr())
		} else if c.host.ID == 0 {
			t.Errorf("Host %s (%s) did not get an ID after adding",
				c.host.Name,
				c.host.AStr())
		} else {
			tHosts[c.host.ID] = c.host
		}
	}
} // func TestHostAdd(t *testing.T)
