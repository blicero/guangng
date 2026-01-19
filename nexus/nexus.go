// /home/krylon/go/src/github.com/blicero/guangng/nexus/nexus.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-19 19:22:35 krylon>

package nexus

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/generator"
	"github.com/blicero/guangng/logdomain"
)

// Nexus coordinates the various subsystems.
type Nexus struct {
	log    *log.Logger
	lock   sync.RWMutex
	active atomic.Bool
	gen    *generator.Generator
}

// New returns a new Nexus.
func New(gaCnt, gnCnt int) (*Nexus, error) {
	var (
		err error
		nx  = new(Nexus)
	)

	if nx.log, err = common.GetLogger(logdomain.Nexus); err != nil {
		return nil, err
	} else if nx.gen, err = generator.New(gaCnt, gnCnt); err != nil {
		nx.log.Printf("[ERROR] Failed to create Generator: %s\n",
			err.Error())
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
} // func (nx *Nexus) Start()

// Stop all running subsystems.
func (nx *Nexus) Stop() {
	nx.log.Println("[INFO] Stopping subsystems...")
	nx.active.Store(false)
	nx.gen.Stop()
} // func (nx *Nexus) Stop()
