// /home/krylon/go/src/github.com/blicero/chili/logdomain/logdomain.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-19 18:53:48 krylon>

package logdomain

// ID represents the various pieces of the application that may want to log messages.
type ID uint8

//go:generate stringer -type=ID

const (
	Common ID = iota
	Database
	DBPool
	Web
	Generator
	IPCache
	Blacklist
	XFR
	Scanner
	Nexus
)

// AllDomains returns a slice of all valid values for logdomain.ID
func AllDomains() []ID {
	return []ID{
		Common,
		Database,
		DBPool,
		Web,
		Generator,
		IPCache,
		Blacklist,
		XFR,
		Scanner,
		Nexus,
	}
} // func AllDomains() []ID
