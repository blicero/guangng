// /home/krylon/go/src/guangng/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2026-01-30 14:22:22 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

type tmplDataBase struct { // nolint: unused
	Title string
	Debug bool
	URL   string
	// Messages   []*message
}

type tmplDataIndex struct { // nolint: unused,deadcode
	tmplDataBase
	GenActive  bool
	XFRActive  bool
	ScanActive bool
	GenAddrCnt int
	GenNameCnt int
	XFRCnt     int
	ScanCnt    int
	HostCnt    int64
}

// type tmplDataNetworkAll struct { // nolint: unused,deadcode
// 	tmplDataBase
// 	Networks []*model.Network
// 	DevCnt   map[int64]int
// 	Network  *model.Network
// 	Scans    map[int64]*scanner.ScanProgress
// }

// type tmplDataNetworkDetails struct {
// 	tmplDataBase
// 	Network *model.Network
// 	Devices []*model.Device
// }

// type tmplDataDeviceAll struct {
// 	tmplDataBase
// 	Devices []*model.Device
// 	Updates map[int64]*model.Updates
// 	Disk    map[int64]*model.DiskFree
// }

// func (d *tmplDataDeviceAll) DiskFree(devID int64) int64 {
// 	var (
// 		free *model.DiskFree
// 		ok   bool
// 	)

// 	if free, ok = d.Disk[devID]; ok {
// 		return free.PercentFree
// 	}

// 	return 100
// } // func (d *tmplDataDeviceAll) DiskFree(devID int64) (int64, bool)

// type tmplDataDeviceDetails struct {
// 	tmplDataBase
// 	Device  *model.Device
// 	Network *model.Network
// 	Uptime  *model.Uptime
// 	Updates *model.Updates
// }

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && gometalinter && go test -v" //
// End: //
