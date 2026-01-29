// /home/krylon/go/src/github.com/blicero/guangng/nexus/nexus.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-28 17:01:24 krylon>

package nexus

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/generator"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model/subsystem"
	"github.com/blicero/guangng/scanner"
	"github.com/blicero/guangng/xfr"
)

// Nexus coordinates the various subsystems.
type Nexus struct {
	log    *log.Logger
	lock   sync.RWMutex
	active atomic.Bool
	gen    *generator.Generator
	xfr    *xfr.XFR
	scn    *scanner.Scanner
}

// New returns a new Nexus.
func New(gaCnt, gnCnt, xcnt, scnt int) (*Nexus, error) {
	var (
		err error
		nx  = new(Nexus)
	)

	if nx.log, err = common.GetLogger(logdomain.Nexus); err != nil {
		return nil, err
	} else if nx.gen, err = generator.New(gaCnt, gnCnt); err != nil {
		nx.log.Printf("[CRITICAL] Failed to create Generator: %s\n",
			err.Error())
		return nil, err
	} else if nx.xfr, err = xfr.New(xcnt); err != nil {
		nx.log.Printf("[CRITICAL] Failed to create XFR Engine: %s\n",
			err.Error())
		return nil, err
	} else if nx.scn, err = scanner.New(scnt); err != nil {
		nx.log.Printf("[CRITICAL] Failed to create Scanner: %s\n",
			err.Error())
		return nil, err
	}

	return nx, nil
} // func New(gaCnt, gnCnt int) (*Nexus, error)

// IsActive returns the status of the Nexus' active flag.
func (nx *Nexus) IsActive() bool {
	return nx.active.Load()
} // func (nx *Nexus) IsActive() bool

// Start the various subsystems.
func (nx *Nexus) Start() {
	nx.log.Println("[INFO] Starting subsystems...")
	nx.active.Store(true)
	nx.gen.Start()
	nx.xfr.Start()
	nx.scn.Start()
} // func (nx *Nexus) Start()

// Stop all running subsystems.
func (nx *Nexus) Stop() {
	nx.log.Println("[INFO] Stopping subsystems...")
	nx.active.Store(false)
	nx.gen.Stop()
	nx.xfr.Stop()
	nx.scn.Stop()
} // func (nx *Nexus) Stop()

// GetActiveFlag returns the active flag of the specified subsystem.
func (nx *Nexus) GetActiveFlag(sub subsystem.ID) bool {
	switch sub {
	case subsystem.Generator, subsystem.GeneratorAddress, subsystem.GeneratorName:
		return nx.gen.IsActive()
	case subsystem.XFR:
		return nx.xfr.IsActive()
	case subsystem.Scanner:
		return nx.scn.IsActive()
	default:
		var err = fmt.Errorf("invalid subsystem ID: %s (%d)",
			sub,
			sub)
		nx.log.Printf("[ERROR] %s\n", err.Error())
		panic(err)
	}
} // func (nx *Nexus) GetActiveFlag(sub subsystem.ID) bool

// GetWorkerCount returns the number of active workers in a subsystem.
// func (nx *Nexus) GetWorkerCount(sub subsystem.ID) int {
// 	switch sub {
// 	case subsystem.Generator:
// 		return nx.gen.
// 	}
// } // func (nx *Nexus) GetWorkerCount(sub subsystem.ID) int
