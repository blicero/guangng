// /home/krylon/go/src/github.com/blicero/guangng/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-12 15:50:07 krylon>

package query

//go:generate stringer -type=ID

// ID represents a particular SQL query.
type ID uint8

const (
	HostAdd ID = iota
	HostGetByID
	HostGetByAddr
	HostGetAll
	HostGetRandom
	HostUpdateSysname
	HostUpdateLocation
	XFRAdd
	XFRGetByID
	XFRGetUnfinished
	XFRStart
	XFRFinish
)
