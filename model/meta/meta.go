// /home/krylon/go/src/github.com/blicero/guangng/model/meta/meta.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 02. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-02 13:01:16 krylon>

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
	"Suse Linux",
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
	"Suse Linux": {
		regexp.MustCompile("(?i)SUSE Linux"),
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
	}

	eng.updateQ = make(chan bool, 1)
	return eng, nil

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
	go func() {
		time.Sleep(time.Minute)
		m.updateQ <- true
	}()
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
		select {
		case <-heartbeat.C:
			continue
		case <-m.updateQ:
			m.guessOS()
			m.guessLocation()
		case <-refreshTicker.C:
			m.guessOS()
			m.guessLocation()
		}
	}
} // func (m *MetaEngine) worker()

func (m *Engine) guessLocation() {
	var (
		err   error
		db    *database.Database
		hosts []*model.Host
		cnt   int
	)

	m.log.Println("[TRACE] Let's see if we can fill in some blanks...")
	defer func() {
		m.log.Printf("[DEBUG] Updated location for %d hosts.\n",
			cnt)
	}()

	// db = m.pool.Get()
	// defer m.pool.Put(db)
	if db, err = database.Open(common.DbPath); err != nil {
		m.log.Printf("[ERROR] Failed to open database: %s\n",
			err.Error())
		return
	}
	defer db.Close() // nolint: errcheck

	if hosts, err = db.HostGetMissingLocation(); err != nil {
		m.log.Printf("[ERROR] Failed to get list of Hosts missing metadata: %s\n",
			err.Error())
		return
	}

	m.log.Printf("[DEBUG] Attempting to guess location for %d hosts.\n",
		len(hosts))

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
			// m.log.Printf("[DEBUG] Could not determine location for %s/%s\n",
			// 	h.Name,
			// 	h.AStr())
			continue
		}

		m.log.Printf("[DEBUG] Set location for %s/%s to %s\n",
			h.Name,
			h.AStr(),
			location)

		if err = db.HostUpdateLocation(h, location); err != nil {
			m.log.Printf("[ERROR] Failed to update Host location for %s/%s: %s\n",
				h.Name,
				h.AStr(),
				err.Error())
			return
		}

		cnt++
	}
} // func (m *MetaEngine) guessLocation()

func (m *Engine) guessOS() {
	var (
		err       error
		db        *database.Database
		svcMap    map[int64][]*model.Service
		updateCnt int
	)

	m.log.Println("[DEBUG] Attempting to guess OS of Hosts we scanned.")

	// db = m.pool.Get()
	// defer m.pool.Put(db)
	if db, err = database.Open(common.DbPath); err != nil {
		m.log.Printf("[ERROR] Failed to open database: %s\n",
			err.Error())
		return
	}
	defer db.Close() // nolint: errcheck

	if svcMap, err = db.ServiceGetSuccessByHost(); err != nil {
		m.log.Printf("[ERROR] Could not load list of scanned ports: %s\n",
			err.Error())
		return
	}

	m.log.Printf("[DEBUG] Gonna process %d responses from scanned ports\n",
		len(svcMap))

	for id, ports := range svcMap {
		var host *model.Host

		if host, err = db.HostGetByID(id); err != nil {
			m.log.Printf("[ERROR] Failed to get Host %d: %s\n",
				id,
				err.Error())
			continue
		} else if host == nil {
			m.log.Printf("[CANTHAPPEN] Did not find Host %d in database\n",
				id)
			continue
		}

		var (
			hits = make(map[string]int, len(osList))
			ok   bool
		)

		for _, port := range ports {
			for name, patterns := range osPatterns {
				for _, pat := range patterns {
					if pat.MatchString(port.Response) {
						var cnt = hits[name]
						hits[name] = cnt + 1
						ok = true
					}
				}
			}
		}

		if !ok {
			continue
		}

		var (
			name string
			cnt  int
		)

		for n, c := range hits {
			if c > cnt {
				name = n
				cnt = c
			}
		}

		if cnt == 0 || name == host.Sysname {
			continue
		} else if err = db.HostUpdateSysname(host, name); err != nil {
			m.log.Printf("[ERROR] Failed to update Sysname for Host %s/%s to %s: %s\n",
				host.Name,
				host.AStr(),
				name,
				err.Error())

		} else {
			updateCnt++
			m.log.Printf("[DEBUG] Host %s/%s appears to be running %s\n",
				host.Name,
				host.AStr(),
				name)
		}
	}

	m.log.Printf("[INFO] Updated OS on %d Hosts.\n", updateCnt)
} // func (m *Engine) guessOS()

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

// // LookupOperatingSystem attempts to determine what OS a Host is running.
// func (m *Engine) LookupOperatingSystem(h *model.Host, ports map[uint16]*model.Service) string {
// 	var results map[string]int = make(map[string]int)

// PORT:
// 	for _, port := range ports {
// 		//for os, patterns := range osPatterns {
// 		for _, osname := range osList {
// 			patterns := osPatterns[osname]
// 			for _, pattern := range patterns {
// 				if port.Response != "" && pattern.MatchString(port.Response) {
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
// }
// func (m *Engine) LookupOperatingSystem(h *model.Host, ports map[uint16]*model.Service) string

// UpdateMetadata refreshes the location and OS metadata for all hosts.
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
