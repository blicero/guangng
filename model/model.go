// /home/krylon/go/src/github.com/blicero/guangng/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-12 15:32:34 krylon>

// Package model provides the data types our application deals with.
package model

import (
	"fmt"
	"net"
	"time"
)

// HostSource signifies how a Host found its way into the database.
type HostSource uint8

const (
	Generator HostSource = iota + 1
	XFR
	MX
	NS
	User
)

// For once, I am not using stringer, that list won't change.
//
// But I am leaving this here: Add an X for each time that decision
// has bit me in the behind:
//

func (s HostSource) String() string {
	switch s {
	case Generator:
		return "Generator"
	case XFR:
		return "XFR"
	case MX:
		return "MX"
	case NS:
		return "NS"
	case User:
		return "User"
	default:
		panic(fmt.Sprintf("Invalid HostSource value %d", s))
	}
} // func (s HostSource) String() string

// Host is a host on the wide, wide Internet.
type Host struct {
	ID          int64
	Addr        net.IP
	Name        string
	Added       time.Time
	LastContact time.Time
	Sysname     string
	Location    string
	Source      HostSource
	astr        string
}

// AStr returns a string representation of the Host's IP address.
func (h *Host) AStr() string {
	if h.astr == "" {
		h.astr = h.Addr.String()
	}

	return h.astr
} // func (h *Host) AStr() string

// Zone is a DNS zone that we may attempt to perform a zone transfer on.
type Zone struct {
	ID       int64
	Name     string
	Added    time.Time
	Started  time.Time
	Finished time.Time
	Status   string
}

// Service represents a scanned port (success or not).
type Service struct {
	ID        int64
	HostID    int64
	Port      uint16
	Success   bool
	Response  string
	Timestamp time.Time
}
