// /home/krylon/go/src/github.com/blicero/guangng/generator/generator.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-12 22:42:15 krylon>

package generator

import (
	"log"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/logdomain"
	"github.com/dgraph-io/badger"
)

type cache struct {
	log *log.Logger
	db  *badger.DB
}

func openCache() (*cache, error) {
	var (
		err     error
		ipcache = new(cache)
	)

	if ipcache.log, err = common.GetLogger(logdomain.IPCache); err != nil {
		return nil, err
	} else if ipcache.db, err = badger.Open(badger.DefaultOptions(common.CachePath)); err != nil {
		ipcache.log.Printf("[ERROR] Failed to open IP cache at %s: %s\n",
			common.CachePath,
			err.Error())
		return nil, err
	}

	return ipcache, nil
} // func openCache() (*cache, error)

type Generator struct {
	log *log.Logger
}
