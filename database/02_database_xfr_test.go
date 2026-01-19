// /home/krylon/go/src/github.com/blicero/guangng/database/02_database_xfr_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-19 19:52:21 krylon>

package database

import (
	"testing"
	"time"

	"github.com/blicero/guangng/model"
)

func TestXFRAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err error
		z   = model.Zone{
			Name:  "example.com",
			Added: time.Now(),
		}
	)

	if err = tdb.XFRAdd(&z); err != nil {
		t.Errorf("Failed to add zone to XFR table: %s\n",
			err.Error())
	}
} // func TestXFRAdd(t *testing.T)
