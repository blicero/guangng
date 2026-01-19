// /home/krylon/go/src/github.com/blicero/guangng/model/subsystem/subsystem.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-19 19:06:12 krylon>

package subsystem

//go:generate stringer -type=ID

// ID represents a subsystem in the application.
type ID uint8

const (
	None ID = iota
	Generator
	GeneratorAddress
	GeneratorName
	XFR
	Scanner
)
