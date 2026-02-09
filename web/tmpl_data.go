// /home/krylon/go/src/guangng/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2026-02-09 14:43:32 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

import (
	"github.com/blicero/guangng/model"
	"github.com/blicero/guangng/model/subsystem"
)

type tmplDataBase struct { // nolint: unused
	Title      string
	Debug      bool
	URL        string
	Subsystems []subsystem.ID
	GenActive  bool
	XFRActive  bool
	ScanActive bool
	GenAddrCnt int
	GenNameCnt int
	XFRCnt     int
	ScanCnt    int
	HostCnt    int64
	ZoneCnt    int64
	PortCnt    int64
}

// HostGenCnt returns the total number of workers in the Generator subsystem.
func (d *tmplDataBase) HostGenCnt() int {
	return d.GenAddrCnt + d.GenNameCnt
} // func (d *tmplDataIndex) HostGenCnt() int

type tmplDataIndex struct { // nolint: unused,deadcode
	tmplDataBase
}

type tmplDataByPort struct {
	tmplDataBase
	Ports map[uint16][]*model.Service
	Hosts map[int64]*model.Host
}
