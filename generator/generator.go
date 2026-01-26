// /home/krylon/go/src/github.com/blicero/guangng/generator/generator.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-19 20:29:53 krylon>

package generator

import (
	"crypto/rand"
	"log"
	"net"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/blicero/guangng/blacklist"
	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/database"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model"
	"github.com/blicero/guangng/model/hsrc"
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

func (c *cache) check(addr net.IP) (bool, error) {
	var (
		err     error
		present bool
	)

	err = c.db.Update(func(tx *badger.Txn) error {
		if _, err := tx.Get(addr); err == badger.ErrKeyNotFound {
			var val = []byte{0x1}
			if err = tx.Set(addr, val); err != nil {
				c.log.Printf("[ERROR] Failed to add address %s to cache: %s\n",
					addr,
					err.Error())
				return err
			}

		} else if err != nil {
			c.log.Printf("[ERROR] Failed to lookup %s in cache: %s\n",
				addr,
				err.Error)
			return err
		} else {
			present = true
		}

		return nil
	})

	return present, err
} // func (c *cache) check(addr net.IP) bool

// Generator generates random Hosts, checking them against blacklists
// and ensuring that the IP address resolves to a valid PTR (i.e. that the
// generated Host is likely to exist on the Internet).
type Generator struct {
	log        *log.Logger
	cache      *cache
	blAddr     *blacklist.BlacklistAddr
	blName     *blacklist.BlacklistName
	ipQ        chan net.IP
	hostQ      chan *model.Host
	active     atomic.Bool
	iCnt, nCnt int
	ctlQAddr   chan bool
	ctlQName   chan bool
}

// New creates a new Generator.
// iCnt is the number of goroutines to spawn for generating IP addresses
// wCnt is the number of goroutines to spawn for resolving and checking hostnames.
func New(icnt, ncnt int) (*Generator, error) {
	var (
		err error
		gen = &Generator{
			iCnt: icnt,
			nCnt: ncnt,
		}
	)

	if gen.log, err = common.GetLogger(logdomain.Generator); err != nil {
		return nil, err
	} else if gen.cache, err = openCache(); err != nil {
		gen.log.Printf("[ERROR] Failed to open cache: %s\n",
			err.Error())
		return nil, err
	}

	gen.blAddr = blacklist.NewBlacklistAddr()
	gen.blName = blacklist.NewBlacklistName()
	gen.ipQ = make(chan net.IP, icnt)
	gen.hostQ = make(chan *model.Host, ncnt)
	gen.ctlQAddr = make(chan bool, icnt)
	gen.ctlQName = make(chan bool, ncnt)

	return gen, nil
} // func New() (*Generator, error)

// Start sets the Generator's active flag and spawns the worker goroutines.
func (gen *Generator) Start() {
	gen.active.Store(true)

	for i := range gen.iCnt {
		go gen.addrWorker(i)
	}

	for i := range gen.nCnt {
		go gen.nameWorker(i)
	}

	go gen.hostWorker()
} // func (gen *Generator) Start()

// Stop clears the Generator's active flag.
func (gen *Generator) Stop() {
	gen.active.Store(false)
} // func (gen *Generator) Stop()

// IsActive returns the Generator's active flag.
func (gen *Generator) IsActive() bool {
	return gen.active.Load()
} // func (gen *Generator) IsActive() bool

func (gen *Generator) addrWorker(id int) {
	const maxErr = 5

	gen.log.Printf("[DEBUG] addrWorker#%d starting up...\n", id)
	defer gen.log.Printf("[DEBUG] addrWorker#%d is quitting.", id)

	var ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for gen.active.Load() {
		var (
			err    error
			addr   net.IP
			errCnt int
		)

		if addr, err = gen.mkIP(); err != nil {
			gen.log.Printf("[ERROR] addrWorker#%d failed to generate IP address: %s\n",
				id,
				err.Error())
			errCnt++
			if errCnt >= maxErr {
				gen.log.Printf("[ERROR] addrWorker#%d failed %d times, I'll bail!",
					id,
					errCnt)
				return
			}
		}

	SEND_ADDR:
		select {
		case gen.ipQ <- addr:
			continue
		case <-gen.ctlQAddr:
			return
		case <-ticker.C:
			if gen.active.Load() {
				goto SEND_ADDR
			}
		}
	}
} // func (gen *Generator) addrWorker(id int)

func (gen *Generator) mkIP() (net.IP, error) {
	const maxErr = 5
	var (
		err               error
		octets            [4]byte
		bytesRead, errCnt int
	)

	for {
		if bytesRead, err = rand.Read(octets[:]); err != nil {
			gen.log.Printf("[ERROR] Failed to read random bytes: %s\n",
				err.Error())
			return nil, err
		} else if bytesRead != 4 {
			continue
		}

		var (
			known bool
			addr  = net.IPv4(octets[0], octets[1], octets[2], octets[3])
		)

		if known, err = gen.cache.check(addr); err != nil {
			gen.log.Printf("[ERROR] Failed to look up IP %s in cache: %s\n",
				addr,
				err.Error())
			errCnt++
			if errCnt >= maxErr {
				return nil, err
			}
		} else if known || gen.blAddr.Match(addr) {
			continue
		}

		return addr, nil
	}
} // func (gen *Generator) mkIP() (net.IP, error)

func (gen *Generator) nameWorker(id int) {
	var (
		err    error
		addr   net.IP
		host   *model.Host
		ticker *time.Ticker
	)

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for gen.active.Load() {
		select {
		case <-ticker.C:
			continue
		case <-gen.ctlQName:
			return
		case addr = <-gen.ipQ:
			if host, err = gen.processAddr(addr); err != nil {
				if !ignoreErr(err) {
					gen.log.Printf("[ERROR] nameWorker#%d failed to process IP address %s: %s\n",
						id,
						addr,
						err.Error())
				}
				continue
			} else if host != nil {
				gen.hostQ <- host
			}
		}
	}
} // func (gen *Generator) nameWorker(id int)

func ignoreErr(err error) bool {
	var (
		res bool
		msg = err.Error()
	)

	if strings.HasSuffix(msg, "no such host") {
		res = true
	} else if strings.HasSuffix(msg, "Temporary failure in name resolution") {
		res = true
	}

	return res
} // func ignoreErr(err error) bool

func isTransient(err error) bool {
	return strings.HasSuffix(err.Error(), "Temporary failure in name resolution")
} // func isTransient(err error) bool

func (gen *Generator) processAddr(addr net.IP) (*model.Host, error) {
	const (
		maxErr     = 5
		retryDelay = time.Millisecond * 250
	)
	var (
		err    error
		errCnt int
		names  []string
	)

RESOLVE:
	if names, err = net.LookupAddr(addr.String()); err != nil {
		if isTransient(err) {
			if errCnt < maxErr {
				errCnt++
				time.Sleep(retryDelay)
				goto RESOLVE
			}
		}
		if !ignoreErr(err) {
			gen.log.Printf("[ERROR] Failed to resolve address %s to name: %s\n",
				addr,
				err.Error())
		}
		return nil, err
	} else if len(names) == 0 {
		return nil, nil
	} else if gen.blName.Match(names[0]) {
		return nil, nil
	}

	var host = &model.Host{
		Addr:   addr,
		Name:   names[0],
		Added:  time.Now(),
		Source: hsrc.Generator,
	}

	return host, nil
} // func (gen *Generator) processAddr(addr net.IP) (*model.Host, error)

func (gen *Generator) hostWorker() {
	var (
		err    error
		db     *database.Database
		host   *model.Host
		ticker *time.Ticker
	)

	if db, err = database.Open(common.DbPath); err != nil {
		gen.log.Printf("[ERROR] hostWorker failed to open database: %s\n",
			err.Error())
		panic(err)
	}

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	for gen.active.Load() {
		select {
		case <-ticker.C:
			continue
		case host = <-gen.hostQ:
			if host == nil {
				gen.log.Println("[CANTHAPPEN] Received nil Host from hostQ!")
				continue
			} else if err = db.HostAdd(host); err != nil {
				gen.log.Printf("[ERROR] Failed to add Host to Database: %s\n",
					err.Error())
			} else {
				gen.checkXFR(host, db)
			}
		}
	}
} // func (gen *Generator) hostWorker()

var tldPat = regexp.MustCompile("^[^.]+[.]?$")

func (gen *Generator) checkXFR(host *model.Host, db *database.Database) {
	var (
		err error
		xfr *model.Zone
		dns = host.Zone()
	)

	if tldPat.MatchString(dns) {
		gen.log.Printf("[DEBUG] Zone %s looks like a top-level domain, so we skip it.\n",
			dns)
		return
	} else if xfr, err = db.XFRGetByName(dns); err != nil {
		gen.log.Printf("[ERROR] Failed to look up XFR of zone %s: %s\n",
			dns,
			err.Error())
		return
	} else if xfr != nil {
		return
	}

	xfr = &model.Zone{
		Name:  dns,
		Added: time.Now(),
	}

	if err = db.XFRAdd(xfr); err != nil {
		gen.log.Printf("[ERROR] Failed to add DNS zone %s to database: %s\n",
			dns,
			err.Error())
	}
} // func (gen *Generator) checkXFR(host *model.Host)
