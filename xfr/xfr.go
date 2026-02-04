// /home/krylon/go/src/github.com/blicero/guangng/xfr/xfr.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-02-04 14:34:17 krylon>

// Package xfr handles zone transfers, an attempt to get more Hosts into the
// database, as the Generator itself is kind of slow.
package xfr

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/blicero/guangng/blacklist"
	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/database"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model"
	"github.com/blicero/guangng/model/hsrc"
	"github.com/blicero/guangng/model/subsystem"
	dns "github.com/tonnerre/golang-dns"
)

// XFR attempts to perform zone transfers.
type XFR struct {
	log       *log.Logger
	active    atomic.Bool
	xcnt      atomic.Int32
	idCounter atomic.Int64
	goalCnt   int
	cmdQ      chan bool
	xfrQ      chan *model.Zone
	hostQ     chan *model.Host
	res       *dns.Client
	pool      *database.Pool
	blName    *blacklist.BlacklistName
	blAddr    *blacklist.BlacklistAddr
}

// New returns a new XFR instance.
func New(cnt int) (*XFR, error) {
	var (
		err  error
		xcnt = max(cnt, 2)
		x    = &XFR{
			goalCnt: cnt,
		}
	)

	if x.log, err = common.GetLogger(logdomain.XFR); err != nil {
		return nil, err
	} else if x.pool, err = database.NewPool(4); err != nil {
		x.log.Printf("[ERROR] Failed to create DB pool: %s\n",
			err.Error())
		return nil, err
	}

	// x.xcnt.Store(int32(cnt))
	x.cmdQ = make(chan bool, xcnt)
	x.xfrQ = make(chan *model.Zone, xcnt)
	x.hostQ = make(chan *model.Host, xcnt)
	x.res = new(dns.Client)
	x.blAddr = blacklist.NewBlacklistAddr()
	x.blName = blacklist.NewBlacklistName()

	x.res.Net = "tcp"

	return x, nil
} // func New(cnt int) (*XFR, error)

func (x *XFR) getID() int {
	var val = x.idCounter.Add(1)
	return int(val)
} // func (x *XFR) getID() int

// IsActive returns the XFR engine's active flag.
func (x *XFR) IsActive() bool {
	return x.active.Load()
} // func (x *XFR) IsActive() bool

// Start sets the XFR engine's active flag and starts the set number of
// worker goroutines.
func (x *XFR) Start() {
	x.active.Store(true)

	go x.hostWorker()
	go x.xfrFeeder()

	for range x.goalCnt {
		go x.xfrWorker(x.getID())
	}
} // func (x *XFR) Start()

// Stop clears the XFR engine's active flag (if set).
func (x *XFR) Stop() {
	x.active.Store(false)
} // func (x *XFR) Stop()

// StartOne starts an additional worker.
func (x *XFR) StartOne() {
	go x.xfrWorker(x.getID())
} // func (x *XFR) StartOne()

// StopOne stops one worker.
func (x *XFR) StopOne() {
	x.cmdQ <- true
} // func (x *XFR) StopOne()

func (x *XFR) WorkerCount() int {
	return int(x.xcnt.Load())
} // func (x *XFR) WorkerCount() int

func (x *XFR) System() subsystem.ID {
	return subsystem.XFR
} // func (x *XFR) System() subsystem.ID

func (x *XFR) hostWorker() {
	x.log.Println("[DEBUG] hostWorker starting up...")
	defer x.log.Println("[DEBUG] hostWorker quitting...")

	var (
		err    error
		db     *database.Database
		ticker *time.Ticker
	)

	db = x.pool.Get()
	defer x.pool.Put(db)

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

	db = x.pool.Get()
	defer x.pool.Put(db)

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for x.active.Load() {
		var (
			xlist     []*model.Zone
			batchSize = int(x.xcnt.Load())
		)

		x.log.Printf("[TRACE] Query for up to %d unfinished XFRs\n", batchSize)
		if batchSize == 0 {
			if !x.active.Load() {
				return
			}
			time.Sleep(common.ActiveTimeout)
			continue
		}

		if xlist, err = db.XFRGetUnfinished(batchSize); err != nil {
			x.log.Printf("[ERROR] Failed to get %d unfinished XFRs: %s\n",
				batchSize,
				err.Error())
			x.active.Store(false)
			return
		} else if len(xlist) == 0 {
			x.log.Println("[DEBUG] No unfinished XFRs were found, maybe next time...")
			time.Sleep(common.ActiveTimeout)
			continue
		}

		for _, z := range xlist {
		SEND:
			select {
			case <-ticker.C:
				if !x.active.Load() {
					x.log.Println("[TRACE] XFR engine has been stopped, I'm going home.")
					return
				}
				goto SEND
			case x.xfrQ <- z:
				x.log.Printf("[TRACE] DNS zone %s has been submitted for AXFR\n",
					z.Name)
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
		ticker *time.Ticker
	)

	x.xcnt.Add(1)
	defer x.xcnt.Add(-1)

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for x.active.Load() {
		select {
		case <-ticker.C:
			continue
		case <-x.cmdQ:
			x.log.Printf("[DEBUG] xfrWorker#%02d Somebody told me to stop? Fine by me, have a nice day!",
				id)
			return
		case z := <-x.xfrQ:
			x.log.Printf("[DEBUG] Attempt AXFR of %s...\n", z.Name)
			if cnt, err = x.doXFR(z); err != nil {
				x.log.Printf("[ERROR] AXFR of %s failed: %s\n",
					z.Name,
					err.Error())
			} else {
				x.log.Printf("[DEBUG] AXFR of %s completed, %d RRs were processed.\n",
					z.Name,
					cnt)
			}
		}
	}
} // func (x *XFR) xfrWorker(id int)

func (x *XFR) doXFR(z *model.Zone) (int64, error) {
	x.log.Printf("[DEBUG] Attempt AXFR of %s...\n",
		z.Name)

	var (
		err    error
		db     *database.Database
		cnt    int64
		status bool
		soa    []*net.NS
	)

	db = x.pool.Get()
	defer x.pool.Put(db)

	if err = db.XFRStart(z); err != nil {
		x.log.Printf("[ERROR] Failed to register XFR of %s in database: %s\n",
			z.Name,
			err.Error())
		return 0, err
	}

	defer func() {
		var ex error
		if ex = db.XFRFinish(z, status); ex != nil {
			x.log.Printf("[ERROR] Failed to register XFR of %s as finished: %s\n",
				z.Name,
				ex.Error())
		}
	}()

	if soa, err = net.LookupNS(z.Name); err != nil {
		x.log.Printf("[ERROR] failed to find nameservers for %s: %s\n",
			z.Name,
			err.Error())
		return 0, err
	} else if len(soa) == 0 {
		x.log.Printf("[TRACE] No nameservers were found for %s.\n",
			z.Name)
		return 0, nil
	} else if common.Debug {
		var servers = make([]string, len(soa))
		for i, s := range soa {
			servers[i] = s.Host
		}

		x.log.Printf("[TRACE] Found %d servers for %s: %s\n",
			len(soa),
			z.Name,
			strings.Join(servers, ", "))
	}

SOA_LOOP:
	for _, ns := range soa {
		if ns == nil {
			continue
		}
		cnt, err = x.queryXFR(z, net.ParseIP(ns.Host))
		if err == nil {
			status = true
			break SOA_LOOP
		}
	}

	return cnt, nil
} // func (x *XFR) doXFR(z *model.Zone) (int64, error)

func (x *XFR) queryXFR(z *model.Zone, srv net.IP) (int64, error) {
	var (
		err     error
		cnt     int64
		xfrMsg  dns.Msg
		envQ    chan *dns.Envelope
		dbgPath string
		dbgFh   *os.File
	)

	if srv == nil {
		return 0, errors.New("nameserver is nil")
	}

	// ...
	xfrMsg.SetAxfr(z.Name)
	x.log.Printf("[TRACE] Query %s for AXFR of %s\n",
		srv,
		z.Name)

	dbgPath = filepath.Join(common.XfrDbgPath, z.Name)
	if dbgFh, err = os.Create(dbgPath); err != nil {
		var xerr = fmt.Errorf("failed to create spool file for AXFR of %s: %s",
			z.Name,
			err)
		x.log.Printf("[ERROR] %s\n",
			xerr.Error())
		return 0, xerr
	}

	defer func() {
		dbgFh.Close() // nolint: errcheck
		if cnt == 0 {
			os.Remove(dbgPath) // nolint: errcheck
		}
	}()

	var ns = fmt.Sprintf("[%s]:53", srv)

	if envQ, err = x.res.TransferIn(&xfrMsg, ns); err != nil {
		var xerr = fmt.Errorf("failed to get AXFR of %s from %s: %w",
			z.Name,
			ns,
			err,
		)
		x.log.Printf("[DEBUG] %s\n", xerr.Error())
		return 0, xerr
	}

	for envelope := range envQ {
		if envelope.Error != nil {
			err = envelope.Error
			x.log.Printf("[TRACE] Error during AXFR of %s: %s\n",
				z.Name,
				err.Error())
			continue
		}

	RR_LOOP:
		for _, rr := range envelope.RR {
			var (
				addrList []string
				host     = new(model.Host)
			)

			fmt.Fprintln(dbgFh, rr.String()) // nolint: errcheck

			cnt++

			switch t := rr.(type) {
			case *dns.A:
				host.Addr = t.A
				host.Name = rr.Header().Name
				host.Source = hsrc.XFR
				if x.blName.Match(host.Name) || x.blAddr.Match(host.Addr) {
					continue RR_LOOP
				}

				x.hostQ <- host
			case *dns.NS:
				host.Name = rr.Header().Name
				if x.blName.Match(host.Name) {
					continue RR_LOOP
				} else if addrList, err = net.LookupHost(host.Name); err != nil {
					x.log.Printf("[TRACE] Failed to lookup NS %s: %s\n",
						host.Name,
						err.Error())
					continue RR_LOOP
				}

			ADDR_LOOP:
				for _, addr := range addrList {
					var nsHost = &model.Host{
						Name:   host.Name,
						Addr:   net.ParseIP(addr),
						Source: hsrc.NS,
					}

					if x.blAddr.Match(nsHost.Addr) {
						continue ADDR_LOOP
					}

					x.hostQ <- nsHost
				}
			case *dns.MX:
				host.Name = rr.Header().Name
				if x.blName.Match(host.Name) {
					continue RR_LOOP
				} else if addrList, err = net.LookupHost(host.Name); err != nil {
					continue RR_LOOP
				}

				for _, addr := range addrList {
					var mxHost = &model.Host{
						Name:   host.Name,
						Addr:   net.ParseIP(addr),
						Source: hsrc.MX,
					}

					if !x.blAddr.Match(mxHost.Addr) {
						x.hostQ <- mxHost
					}
				}
			case *dns.AAAA:
				host.Name = rr.Header().Name
				host.Addr = t.AAAA
				host.Source = hsrc.XFR

				if !x.blAddr.Match(host.Addr) && !x.blName.Match(host.Name) {
					x.hostQ <- host
				}
			}

		}
	}

	return cnt, err
} // func (x *XFR) queryXFR(z *model.Zone, srv net.IP) (int64, error)
