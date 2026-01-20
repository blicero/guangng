// /home/krylon/go/src/github.com/blicero/guangng/xfr/xfr.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-20 14:35:57 krylon>

// Package xfr handles zone transfers, an attempt to get more Hosts into the
// database, as the Generator itself is kind of slow.
package xfr

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/database"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model"
)

// XFR attempts to perform zone transfers.
type XFR struct {
	log    *log.Logger
	active atomic.Bool
	xcnt   atomic.Int32
	cmdQ   chan bool
	xfrQ   chan *model.Zone
	hostQ  chan *model.Host
}

// New returns a new XFR instance.
func New(cnt int) (*XFR, error) {
	var (
		err error
		x   = new(XFR)
	)

	if x.log, err = common.GetLogger(logdomain.XFR); err != nil {
		return nil, err
	}

	x.cmdQ = make(chan bool, cnt)
	x.xfrQ = make(chan *model.Zone, cnt)
	x.hostQ = make(chan *model.Host, cnt)

	return x, nil
} // func New(cnt int) (*XFR, error)

// IsActive returns the XFR engine's active flag.
func (x *XFR) IsActive() bool {
	return x.active.Load()
} // func (x *XFR) IsActive() bool

// Start sets the XFR engine's active flag and starts the set number of
// worker goroutines.
func (x *XFR) Start() {
	x.log.Println("[ERROR] IMPLEMENTME!!!")
	x.active.Store(true)

	go x.hostWorker()
} // func (x *XFR) Start()

// Stop clears the XFR engine's active flag (if set).
func (x *XFR) Stop() {
} // func (x *XFR) Stop()

func (x *XFR) hostWorker() {
	x.log.Println("[DEBUG] hostWorker starting up...")
	defer x.log.Println("[DEBUG] hostWorker quitting...")

	var (
		err    error
		db     *database.Database
		ticker *time.Ticker
	)

	if db, err = database.Open(common.DbPath); err != nil {
		x.log.Printf("[CRITICAL] Cannot open database: %s\n",
			err.Error())
		x.active.Store(false)
		return
	}

	defer db.Close() // nolint: errcheck

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for x.active.Load() {
		select {
		case <-ticker.C:
			continue
		case h := <-x.hostQ:
			if err = db.HostAdd(h); err != nil {
				x.log.Printf("[ERROR] Failed to add Host %s (%s) to database: %s\n",
					h.Name,
					h.AStr(),
					err.Error())
			}
		}
	}
} // func (x *XFR) hostWorker()

func (x *XFR) xfrFeeder() {
	x.log.Println("[DEBUG] xfrFeeder starting up...")
	defer x.log.Println("[DEBUG] xfrFeeder quitting...")

	var (
		err    error
		db     *database.Database
		ticker *time.Ticker
	)

	if db, err = database.Open(common.DbPath); err != nil {
		x.log.Printf("[CRITICAL] Failed to open database: %s\n",
			err.Error())
		x.active.Store(false)
		return
	}

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for x.active.Load() {
		var (
			xlist     []*model.Zone
			batchSize int = int(x.xcnt.Load())
		)

		if xlist, err = db.XFRGetUnfinished(batchSize); err != nil {
			x.log.Printf("[ERROR] Failed to get %d unfinished XFRs: %s\n",
				batchSize,
				err.Error())
			x.active.Store(false)
			return
		} else if len(xlist) == 0 {
			time.Sleep(common.ActiveTimeout)
			continue
		}

		for _, z := range xlist {
		SEND:
			select {
			case <-ticker.C:
				if !x.active.Load() {
					return
				}
				goto SEND
			case x.xfrQ <- z:
				continue
			}
		}
	}
} // func (x *XFR) xfrFeeder()

func (x *XFR) xfrWorker(id int) {
	x.log.Printf("[DEBUG] xfrWorker#%02d starting up...\n", id)
	defer x.log.Printf("[DEBUG] xfrWorker#%02d quitting...\n", id)

	var (
		err    error
		cnt    int64
		db     *database.Database
		ticker *time.Ticker
	)

	if db, err = database.Open(common.DbPath); err != nil {
		x.log.Printf("[CRITICAL] xfrWorker#%02d: Failed to open database: %s\n",
			id,
			err.Error())
	}

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for x.active.Load() {
		select {
		case <-ticker.C:
			continue
		case <-x.cmdQ:
			return
		case z := <-x.xfrQ:
			x.log.Printf("[DEBUG] Attempt AXFR of %s...\n", z.Name)
			if cnt, err = x.doXFR(z); err != nil {
				x.log.Printf("[ERROR] AXFR of %s failed: %s\n",
					z.Name,
					err.Error())
			}
		}
	}
} // func (x *XFR) xfrWorker(id int)

func (x *XFR) doXFR(z *model.Zone) (int64, error) {
	x.log.Printf("[DEBUG] Attempt AXFR of %s...\n",
		z.Name)

	return 0, nil
} // func (x *XFR) doXFR(z *model.Zone) (int64, error)
