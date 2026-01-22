// /home/krylon/go/src/github.com/blicero/guangng/scanner/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 22. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-22 16:02:57 krylon>

// Package scanner implements scanning ports. Duh.
package scanner

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/database"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model"
)

const maxErr = 5

type scanResult struct {
	host *model.Host
	svc  *model.Service
}

type Scanner struct {
	log     *log.Logger
	scnt    atomic.Int32
	goalCnt atomic.Int32
	idCnt   int
	active  atomic.Bool
	hostQ   chan model.Host
	resQ    chan scanResult
	cmdQ    chan bool
}

func New(cnt int) (*Scanner, error) {
	var (
		err error
		scn = new(Scanner)
	)

	if scn.log, err = common.GetLogger(logdomain.Scanner); err != nil {
		return nil, err
	}

	scn.goalCnt.Store(int32(cnt))
	scn.hostQ = make(chan model.Host, min(2, cnt/2))
	scn.resQ = make(chan scanResult, cnt)
	scn.cmdQ = make(chan bool)

	return scn, nil
} // func New(cnt int) (*Scanner, error)

// WorkerCnt returns the number of active workers.
func (scn *Scanner) WorkerCnt() int32 {
	return scn.scnt.Load()
} // func (scn *Scanner) WorkerCnt() int32

// IsActive returns state of the Scanner's active flag.
func (scn *Scanner) IsActive() bool {
	return scn.active.Load()
} // func (scn *Scanner) IsActive() bool

// Start spawns the Scanner's workers.
func (scn *Scanner) Start() {
	scn.active.Store(true)
	// ...
	go scn.feeder()
	go scn.collector()

	for range scn.goalCnt.Load() {
		scn.idCnt++
		go scn.scanWorker(scn.idCnt)

	}
} // func (scn *Scanner) Start()

// Stop clears the Scanner's active flag.
func (scn *Scanner) Stop() {
	scn.active.Store(false)
} // func (scn *Scanner) Stop()

// StartOne starts one additional worker.
func (scn *Scanner) StartOne() {
} // func (scn *Scanner) StartOne()

// StopOne tells one worker to stop.
func (scn *Scanner) StopOne() {
	scn.cmdQ <- true
} // func (scn *Scanner) StopOne()

func (scn *Scanner) feeder() {
	var (
		err    error
		errcnt int
		db     *database.Database
		ticker *time.Ticker
	)

	scn.log.Println("[TRACE] Host Feeder starting up...")
	defer scn.log.Println("[TRACE] Host Feeder quitting.")

	if db, err = database.Open(common.DbPath); err != nil {
		scn.log.Printf("[CRITICAL] Feeder cannot open database: %s\n",
			err.Error())
		scn.active.Store(false)
		return
	}

	defer db.Close() // nolint: errcheck

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for scn.active.Load() {
		var hosts []model.Host

		if hosts, err = db.HostGetRandom(int(scn.scnt.Load())); err != nil {
			scn.log.Printf("[ERROR] Failed to get random Hosts to scan: %s\n",
				err.Error())
			errcnt++
			if errcnt > maxErr {
				scn.active.Store(false)
				return
			}
		}

		for _, h := range hosts {
		SEND:
			select {
			case <-ticker.C:
				if !scn.active.Load() {
					return
				}
				goto SEND
			case scn.hostQ <- h:
				continue
			}
		}
	}
} // func (scn *Scanner) feeder()

func (scn *Scanner) collector() {
	var (
		err    error
		db     *database.Database
		ticker *time.Ticker
	)

	scn.log.Println("[TRACE] Scan result collector starting up...")
	defer scn.log.Println("[TRACE] Scan result collector quitting.")

	if db, err = database.Open(common.DbPath); err != nil {
		scn.log.Printf("[CRITICAL] Failed to open database: %s\n",
			err.Error())
		scn.active.Store(false)
		return
	}

	defer db.Close() // nolint: errcheck

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for scn.active.Load() {
		select {
		case <-ticker.C:
			continue
		case res := <-scn.resQ:
			if res.svc.Success {
				scn.log.Printf("[DEBUG] Got one: %s:%d -- %s\n",
					res.host.Addr,
					res.svc.Port,
					res.svc.Response)
			}
		}
	}
} // func (scn *Scanner) collector()

func (scn *Scanner) scanWorker(id int) {
	var ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	scn.scnt.Add(1)
	defer scn.scnt.Add(-1)

	scn.log.Printf("[TRACE] scanWorker#%02d reporting for duty\n",
		id)

	for scn.active.Load() {
		select {
		case <-ticker.C:
			continue
		case <-scn.cmdQ:
			return
		case h := <-scn.hostQ:
			// Deal with it!
			scn.log.Printf("[TRACE] scanWorker#%02d about to scan Host %s/%s\n",
				id,
				h.Name,
				h.Addr)
		}
	}
} // func (scn *Scanner) scanWorker(id int)
