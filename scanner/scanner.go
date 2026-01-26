// /home/krylon/go/src/github.com/blicero/guangng/scanner/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 22. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-26 15:11:26 krylon>

// Package scanner implements scanning ports. Duh.
package scanner

import (
	"log"
	"math/rand"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/database"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model"
	"github.com/blicero/guangng/model/hsrc"
)

const maxErr = 5

var wwwPat *regexp.Regexp = regexp.MustCompile("(?i)^www")
var ftpPat *regexp.Regexp = regexp.MustCompile("(?i)^ftp")
var mxPat *regexp.Regexp = regexp.MustCompile("(?i)^(?:mx|mail|smtp|pop|imap)")
var newline = regexp.MustCompile("[\r\n]+$") // nolint: unused

// Ports is the list of ports (TCP and UDP) we consider interesting.
var Ports []uint16 = []uint16{
	21,
	22,
	23,
	25,
	53,
	79,
	80,
	110,
	143,
	161,
	443,
	631,
	1024,
	4444,
	2525,
	5353,
	5800,
	5900,
	8000,
	8080,
	8081,
}

type scanProposal struct {
	host  *model.Host
	ports map[uint16]*model.Service
}

type scanResult struct {
	host *model.Host
	svc  *model.Service
}

// Scanner wraps all the state need to run the portscanner subsystem across
// multiple worker goroutines.
type Scanner struct {
	log     *log.Logger
	scnt    atomic.Int32
	goalCnt atomic.Int32
	idCnt   int
	active  atomic.Bool
	pool    *database.Pool
	hostQ   chan scanProposal
	resQ    chan *scanResult
	cmdQ    chan bool
}

// New creates and returns a fresh Scanner instance.
func New(cnt int) (*Scanner, error) {
	var (
		err  error
		scnt = max(cnt, 2)
		scn  = new(Scanner)
	)

	if scn.log, err = common.GetLogger(logdomain.Scanner); err != nil {
		return nil, err
	} else if scn.pool, err = database.NewPool(scnt); err != nil {
		scn.log.Printf("[CRITICAL] Failed to open DB pool: %s\n",
			err.Error())
		return nil, err
	}

	scn.goalCnt.Store(int32(cnt))
	scn.hostQ = make(chan scanProposal, max(2, scnt/2))
	scn.resQ = make(chan *scanResult, scnt)
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
		var hosts []*model.Host

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
			var prop = scanProposal{
				host: h,
			}

			if prop.ports, err = db.ServiceGetByHost(h); err != nil {
				scn.log.Printf("[ERROR] Failed to get scanned ports for %s (%s): %s\n",
					h.Name,
					h.AStr(),
					err.Error())
				if errcnt++; errcnt > maxErr {
					scn.active.Store(false)
					return
				}

			}
		SEND:
			select {
			case <-ticker.C:
				if !scn.active.Load() {
					return
				}
				goto SEND
			case scn.hostQ <- prop:
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
			if err = db.ServiceAdd(res.host, res.svc); err != nil {
				scn.log.Printf("[ERROR] Failed to add scanned Port %s:%d to database - %s\n",
					res.host.AStr(),
					res.svc.Port,
					err.Error())
			}
		}
	}
} // func (scn *Scanner) collector()

func (scn *Scanner) scanWorker(id int) {
	scn.log.Printf("[TRACE] scanWorker#%02d reporting for duty\n", id)
	defer scn.log.Printf("[TRACE] scanWorker#%02d quitting. Bye.\n", id)

	scn.scnt.Add(1)
	defer scn.scnt.Add(-1)

	var ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for scn.active.Load() {
		select {
		case <-ticker.C:
			continue
		case <-scn.cmdQ:
			return
		case prop := <-scn.hostQ:
			var (
				err  error
				port uint16
				res  *scanResult
			)
			// Deal with it!
			if port = scn.pickPort(prop); port == 0 {
				continue
			}

			scn.log.Printf("[TRACE] scanWorker#%02d about to scan %s:%d\n",
				id,
				prop.host.AStr(),
				port)

			// Let's scan a port!
			if res, err = scn.probePort(prop.host, port); err != nil {
				scn.log.Printf("[ERROR] scanWorker#%02d failed to scan %s:%d - %s\n",
					id,
					prop.host.AStr(),
					port,
					err.Error())
			} else {
				scn.resQ <- res
			}

		}
	}
} // func (scn *Scanner) scanWorker(id int)

func (scn *Scanner) pickPort(prop scanProposal) uint16 {
	var (
		host  = prop.host
		ports = prop.ports
	)

	switch host.Source {
	case hsrc.MX:
		for _, p := range []uint16{25, 110, 143, 587} {
			if ports[p] == nil {
				return p
			}
		}
	case hsrc.NS:
		if ports[53] == nil {
			return 53
		}
	}

	if ftpPat.MatchString(host.Name) && ports[21] == nil {
		return 21
	} else if wwwPat.MatchString(host.Name) {
		for _, p := range []uint16{80, 443, 8000, 8080} {
			if ports[p] == nil {
				return p
			}
		}
	} else if mxPat.MatchString(host.Name) {
		for _, p := range []uint16{25, 110, 143, 587} {
			if ports[p] == nil {
				return p
			}
		}
	}

	indexlist := rand.Perm(len(Ports))
	for _, idx := range indexlist {
		if ports[Ports[idx]] == nil {
			return Ports[idx]
		}
	}

	return 0
} // func (scn *Scanner) pickPort(prop scanProposal) uint16
