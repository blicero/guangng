// /home/krylon/go/src/github.com/blicero/guangng/nexus/nexus.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-16 19:15:21 krylon>

package nexus

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/blicero/guangng/generator"
)

type Nexus struct {
	log    *log.Logger
	lock   sync.RWMutex
	active atomic.Bool
	gen    *generator.Generator
}
