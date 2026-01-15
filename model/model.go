// /home/krylon/go/src/github.com/blicero/guangng/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-15 20:57:06 krylon>

// Package model provides the data types our application deals with.
package model

import (
	"net"
	"regexp"
	"time"

	"github.com/blicero/guangng/model/hsrc"
)

var zonePat = regexp.MustCompile("^[^.]+[.](.*)$")

// Host is a host on the wide, wide Internet.
type Host struct {
	ID          int64
	Addr        net.IP
	Name        string
	Added       time.Time
	LastContact time.Time
	Sysname     string
	Location    string
	Source      hsrc.HostSource
	astr        string
}

// AStr returns a string representation of the Host's IP address.
func (h *Host) AStr() string {
	if h.astr == "" {
		h.astr = h.Addr.String()
	}

	return h.astr
} // func (h *Host) AStr() string

func (h *Host) Zone() string {
	var match = zonePat.FindStringSubmatch(h.Name)

	if match == nil {
		return "" // CANTHAPPEN!
	}

	return match[1]
} // func (h *Host) Zone() string

// Zone is a DNS zone that we may attempt to perform a zone transfer on.
type Zone struct {
	ID       int64
	Name     string
	Added    time.Time
	Started  time.Time
	Finished time.Time
	Status   bool
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
