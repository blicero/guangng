// /home/krylon/go/src/github.com/blicero/guang/frontend/ajax_types.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2026-02-03 15:30:51 krylon>

package web

import (
	"time"
)

type ajaxData struct {
	Status    bool
	Message   string
	Timestamp time.Time
}

// type ajaxDataPorts struct {
// 	ajaxData
// 	Count   int64
// 	Results map[uint16][]data.ScanResult
// }

type ajaxCtlResponse struct {
	ajaxData
	NewCnt int
}

type ajaxWorkerCnt struct {
	ajaxData
	GeneratorAddress int
	GeneratorName    int
	XFR              int
	Scanner          int
}
