// /home/krylon/go/src/github.com/blicero/guangng/database/04_database_svc_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 23. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-23 16:17:01 krylon>

package database

import (
	"testing"

	"github.com/blicero/guangng/model"
)

func TestServiceAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	type svcAddTest struct {
		svc  model.Service
		fail bool
	}

	var testCases = []svcAddTest{
		{
			svc: model.Service{
				HostID:   1,
				Port:     80,
				Success:  true,
				Response: "1337 h4x0r 5ty1e",
			},
		},
		{
			svc: model.Service{
				HostID: 1,
				Port:   23,
			},
		},
		{
			svc: model.Service{
				Port:     22,
				Success:  true,
				Response: "NotSoOpenSSH 0.42.23",
			},
			fail: true,
		},
	}

	for _, c := range testCases {
		var host model.Host

		if c.svc.HostID != 0 {
			host = tHosts[c.svc.HostID]
		}

		if err := tdb.ServiceAdd(&host, &c.svc); err != nil {
			if !c.fail {
				t.Errorf("Failed to add Service %d/%d to Database: %s",
					c.svc.HostID,
					c.svc.Port,
					err.Error())
			}
		} else if c.fail {
			t.Errorf("Adding Service %d/%d should have failed, but didn't",
				c.svc.HostID,
				c.svc.Port)
		}
	}
} // func TestServiceAdd(t *testing.T)
