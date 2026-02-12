// /home/krylon/go/src/github.com/blicero/guangng/model/meta/meta.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 02. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-02-11 16:36:16 krylon>

// Package meta provides facilities to guesstimate the locations and operating
// systems of Hosts.
package meta

import (
	"errors"
	"fmt"
	"log"
	"net/netip"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/database"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model"
	"github.com/oschwald/geoip2-golang/v2"
)

const (
	geoIPCityPath    = "GeoLite2-City.mmdb"
	geoIPCountryPath = "GeoLite2-Country.mmdb"
	refreshInterval  = time.Minute * 15
)

var osList = []string{ // nolint: unused
	"Windows",
	"Ubuntu",
	"Debian",
	"CentOS",
	"Red Hat",
	"Fedora",
	"Yocto",
	"FreeBSD",
	"NetBSD",
	"OpenBSD",
	"DragonflyBSD",
	"RouterOS",
	"Linux",
	"JUNOS",
	"Cisco IOS",
	"SonicOS",
}

var osPatterns = map[string][]*regexp.Regexp{ // nolint: unused
	"Windows": {
		regexp.MustCompile("Microsoft"),
		regexp.MustCompile("Windows"),
	},
	"Debian": {
		regexp.MustCompile("(?i)Debian"),
		regexp.MustCompile("(?i)[.]deb"),
	},
	"Ubuntu": {
		regexp.MustCompile("(?i)ubuntu"),
	},
	"CentOS": {
		regexp.MustCompile("(?i)CentOS"),
	},
	"Red Hat": {
		regexp.MustCompile(`(?i)rhel\d+`),
		regexp.MustCompile("(?i)Red ?Hat"),
		regexp.MustCompile(`(?i)[.]el\d+[.]`),
	},
	"Fedora": {
		regexp.MustCompile("(?i)fedora"),
	},
	"Yocto Linux": {
		regexp.MustCompile("(?i)yocto"),
	},
	"FreeBSD": {
		regexp.MustCompile("(?i)FreeBSD"),
	},
	"OpenBSD": {
		regexp.MustCompile("(?i)OpenBSD"),
	},
	"DragonflyBSD": {
		regexp.MustCompile("Dragonfly"),
	},
	"NetBSD": {
		regexp.MustCompile("(?i)NetBSD"),
	},
	"RouterOS": {
		regexp.MustCompile("(?i)RouterOS"),
	},
	"Linux": {
		regexp.MustCompile("(?i)Linux"),
	},
	"JUNOS": {
		regexp.MustCompile("(?i:JUNOS|Juniper)"),
	},
	"Cisco IOS": {
		regexp.MustCompile("(?i)Cisco IOS Software"),
		regexp.MustCompile("(?i)Cisco Systems"),
	},
	"SonicOS": {
		regexp.MustCompile("(?i)SonicOS"),
		regexp.MustCompile("(?i)SonicWALL"),
	},
}

// Engine processes metadata on Hosts.
type Engine struct {
	citydb    *geoip2.Reader
	countrydb *geoip2.Reader
	log       *log.Logger
	active    atomic.Bool
	updateQ   chan bool
} // type MetaEngine struct

// OpenMetaEngine creates a new MetaEngine.
func OpenMetaEngine() (*Engine, error) {
	var (
		err                            error
		msg, countrydbPath, citydbPath string
		eng                            = new(Engine)
	)

	countrydbPath = filepath.Join(common.BaseDir, geoIPCountryPath)
	citydbPath = filepath.Join(common.BaseDir, geoIPCityPath)

	if eng.log, err = common.GetLogger(logdomain.MetaEngine); err != nil {
		return nil, err
	} else if eng.countrydb, err = geoip2.Open(countrydbPath); err != nil {
		msg = fmt.Sprintf("Error opening GeoIP database %s: %s",
			countrydbPath,
			err.Error())
		eng.log.Println(msg)
		return nil, errors.New(msg)
	} else if eng.countrydb == nil {
		msg = "opening GeoIP database did not return an error, but the geoip2.Reader was nil"
		eng.log.Println(msg)
		return nil, errors.New(msg)
	} else if eng.citydb, err = geoip2.Open(citydbPath); err != nil {
		msg = fmt.Sprintf("cannot open GeoIP database %s: %s",
			citydbPath,
			err.Error())
		eng.log.Printf("[ERROR] %s\n", msg)
		return nil, errors.New(msg)
	} else {
		eng.updateQ = make(chan bool, 1)
		return eng, nil
	}
} // func OpenMetaEngine() (*MetaEngine, error)

// Close closes the MetaEngine.
func (m *Engine) Close() {
	m.countrydb.Close() // nolint: errcheck
} // func (m *MetaEngine) Close()

// Start starts the MetaEngine's worker that periodically attempts to fill in
// missing metadata on Hosts.
func (m *Engine) Start() {
	m.active.Store(true)
	go m.worker()
} // func (m *MetaEngine) Start()

// Stop tells the Engine's worker to quit.
func (m *Engine) Stop() {
	m.active.Store(false)
} // func (m *Engine) Stop()

// IsActive returns the Engine's active flag.
func (m *Engine) IsActive() bool {
	return m.active.Load()
} // func (m *Engine) IsActive() bool

// ForceUpdate triggers an immediate metadata update.
func (m *Engine) ForceUpdate() {
	m.updateQ <- true
} // func (m *Engine) ForceUpdate()

func (m *Engine) worker() {
	m.log.Println("[TRACE] MetaEngine worker starting up...")
	defer m.log.Println("[TRACE] MetaEngine worker quitting...")

	defer m.active.Store(false)

	var refreshTicker = time.NewTicker(refreshInterval)
	defer refreshTicker.Stop()

	var heartbeat = time.NewTicker(common.ActiveTimeout)
	defer heartbeat.Stop()

	for m.active.Load() {
		var err error

		select {
		case <-heartbeat.C:
			continue
		case <-m.updateQ:
			if err = m.updateMeta(); err != nil {
				m.log.Printf("[ERROR] Failed to fill in missing metadata: %s\n",
					err.Error())
			}
		case <-refreshTicker.C:
			if err = m.updateMeta(); err != nil {
				m.log.Printf("[ERROR] Failed to fill in missing metadata: %s\n",
					err.Error())
			}
		}
	}
} // func (m *MetaEngine) worker()

func (m *Engine) updateMeta() error {
	var (
		err    error
		status bool
		db     *database.Database
		hosts  []*model.Host
		cnt    int
	)

	m.log.Println("[TRACE] Let's see if we can fill in some blanks...")

	// ...
	if db, err = database.Open(common.DbPath); err != nil {
		return err
	}

	defer db.Close() // nolint: errcheck

	// I am not entirely sure if it is prudent to update all Hosts' location
	// in one big transaction. But that only occurred to me when I was nearly done
	// writing this code, so I guess I can at least test it and see how it
	// goes.
	if err = db.Begin(); err != nil {
		m.log.Printf("[ERROR] Failed to start DB transaction: %s\n",
			err.Error())
		return err
	}

	defer func() {
		if status {
			db.Commit() // nolint: errcheck
		} else {
			db.Rollback() // nolint: errcheck
		}
	}()

	if hosts, err = db.HostGetMissingLocation(); err != nil {
		m.log.Printf("[ERROR] Failed to get list of Hosts missing metadata: %s\n",
			err.Error())
		return err
	}

	for _, h := range hosts {
		var (
			country, city, location string
		)

		if country, err = m.LookupCountry(h); err != nil {
			m.log.Printf("[ERROR] Failed to look up country for Host %s/%s: %s\n",
				h.Name,
				h.AStr(),
				err.Error())
			continue
		} else if city, err = m.LookupCity(h); err != nil {
			m.log.Printf("[ERROR] Failed too look up city for Host %s/%s: %s\n",
				h.Name,
				h.AStr(),
				err.Error())
			continue
		}

		if country != "" && city != "" {
			location = fmt.Sprintf("%s, %s",
				country, city)
		} else if country != "" {
			location = country
		} else {
			m.log.Printf("[DEBUG] Could not determine location for %s/%s\n",
				h.Name,
				h.AStr())
			continue
		}

		if err = db.HostUpdateLocation(h, location); err != nil {
			m.log.Printf("[ERROR] Failed to update Host location for %s/%s: %s\n",
				h.Name,
				h.AStr(),
				err.Error())
			return err
		}

		cnt++

	}

	m.log.Printf("[DEBUG] Updated location for %d hosts.\n",
		cnt)

	status = true
	return nil
} // func (m *MetaEngine) updateMeta()

// LookupCountry attempts to determine what county a Host is located in.
func (m *Engine) LookupCountry(h *model.Host) (string, error) {
	var (
		err     error
		country *geoip2.Country
		addr    netip.Addr
		ok      bool
	)

	if addr, ok = netip.AddrFromSlice(h.Addr); !ok {
		return "", fmt.Errorf("cannot process IP address of Host %s/%s",
			h.Name,
			h.AStr())
	} else if country, err = m.countrydb.Country(addr); err != nil {
		return "", err
	}

	return country.Country.Names.German, nil
} // func (m *MetaEngine) LookupCountry(h *Host) (string, error)

// LookupCity attempts to determine what city a Host is located in.
func (m *Engine) LookupCity(h *model.Host) (string, error) {
	var (
		err  error
		city *geoip2.City
		addr netip.Addr
		ok   bool
	)

	if addr, ok = netip.AddrFromSlice(h.Addr); !ok {
		return "", fmt.Errorf("cannot process IP address of Host %s/%s",
			h.Name,
			h.AStr())
	} else if city, err = m.citydb.City(addr); err != nil {
		return "", err
	}

	return city.City.Names.German, nil
} // func (m *MetaEngine) LookupCity(h *Host) (string, error)

// LookupOperatingSystem attempts to determine what OS a Host is running.
// func (m *MetaEngine) LookupOperatingSystem(h *data.HostWithPorts) string {
// 	var results map[string]int = make(map[string]int)

// PORT:
// 	for _, port := range h.Ports {
// 		//for os, patterns := range osPatterns {
// 		for _, osname := range osList {
// 			patterns := osPatterns[osname]
// 			for _, pattern := range patterns {
// 				if port.Reply != nil && pattern.MatchString(*port.Reply) {
// 					results[osname]++
// 					continue PORT
// 				}
// 			}
// 		}
// 	}

// 	var (
// 		os     = "Unknown"
// 		hitCnt int
// 	)

// 	for system, cnt := range results {
// 		if cnt > hitCnt {
// 			os = system
// 			hitCnt = cnt
// 		}
// 	}

// 	return os
// } // func (m *MetaEngine) LookupOperatingSystem(h *HostWithPorts) string

// // UpdateMetadata refreshes the location and OS metadata for all hosts.
// func (m *MetaEngine) UpdateMetadata() error {
// 	var (
// 		err   error
// 		db    *database.HostDB
// 		hosts []data.Host
// 	)

// 	if db, err = database.OpenDB(common.DbPath); err != nil {
// 		m.log.Printf("[ERROR] Cannot open HostDB at %s: %s\n",
// 			common.DbPath,
// 			err.Error())
// 		return err
// 	}

// 	defer db.Close() // nolint: errcheck

// 	if hosts, err = db.HostGetAll(); err != nil {
// 		m.log.Printf("[ERROR] Cannot get all hosts: %s\n",
// 			err.Error())
// 		return err
// 	}

// 	for _, host := range hosts {
// 		var (
// 			city, country, location, os string
// 			hwp                         = data.HostWithPorts{Host: host}
// 		)

// 		if city, err = m.LookupCity(&host); err != nil {
// 			m.log.Printf("[ERROR] Cannot lookup city for %s: %s\n",
// 				host.Address,
// 				err.Error())
// 			city = ""
// 		} else if country, err = m.LookupCountry(&host); err != nil {
// 			m.log.Printf("[ERROR] Cannot lookup country for %s: %s\n",
// 				host.Address, err.Error())
// 			goto LOOKUP_OS
// 		}

// 		if city != "" {
// 			location = fmt.Sprintf("%s, %s",
// 				city, country)
// 		} else {
// 			location = country
// 		}

// 		if location == "" {
// 			goto LOOKUP_OS
// 		} else if err = db.HostSetLocation(&host, location); err != nil {
// 			m.log.Printf("[ERROR] Cannot set Location for %s to %q: %s\n",
// 				host.Address,
// 				location,
// 				err.Error())
// 		}

// 	LOOKUP_OS:
// 		if hwp.Ports, err = db.PortGetByHost(host.ID); err != nil {
// 			m.log.Printf("[ERROR] Failed to get scanned ports for %s: %s\n",
// 				host.Address,
// 				err.Error())
// 			continue
// 		} else if len(hwp.Ports) == 0 {
// 			continue
// 		}

// 		os = m.LookupOperatingSystem(&hwp)

// 		if err = db.HostSetOS(&host, os); err != nil {
// 			m.log.Printf("[ERROR] Failed to set OS on host %s to %s: %s\n",
// 				host.Address,
// 				os,
// 				err.Error())
// 		}
// 	}

// 	return nil
// } // func (m *MetaEngine) UpdateMetadata() error
