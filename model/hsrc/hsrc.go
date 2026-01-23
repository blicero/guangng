// /home/krylon/go/src/github.com/blicero/guangng/model/hsrc/hsrc.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-23 15:43:59 krylon>

package hsrc

//go:generate stringer -type=HostSource

// HostSource signifies how a Host found its way into the database.
type HostSource uint8

const (
	_                    = iota
	Generator HostSource = iota
	XFR
	MX
	NS
	User
)
