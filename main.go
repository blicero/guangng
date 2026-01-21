// /home/krylon/go/src/github.com/blicero/guangng/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-21 18:45:13 krylon>

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/nexus"
)

const (
	defaultACnt = 8
	defaultNCnt = 8
	defaultXCnt = 2
)

func printVer() {
	fmt.Printf("%s %s, built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))
}

func main() {
	printVer()

	var (
		err              error
		nx               *nexus.Nexus
		aCnt, nCnt, xCnt int
	)

	flag.IntVar(&aCnt, "acnt", defaultACnt, "Number of address generator workers")
	flag.IntVar(&nCnt, "ncnt", defaultNCnt, "Number of name resolution workers")
	flag.IntVar(&xCnt, "xcnt", defaultXCnt, "Number of AXFR workers")

	flag.Parse()

	if nx, err = nexus.New(aCnt, nCnt, xCnt); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create Nexus: %s\n",
			err.Error())
		os.Exit(1)
	}

	nx.Start()

	var (
		ticker = time.NewTicker(common.ActiveTimeout)
		sigQ   = make(chan os.Signal, 1)
	)

	defer ticker.Stop()

	signal.Notify(sigQ, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			if !nx.IsActive() {
				fmt.Fprintf(
					os.Stderr,
					"Nexus has stopped. So long, suckers!\n")
				return
			}
		case s := <-sigQ:
			fmt.Fprintf(
				os.Stderr,
				"Received signal: %s\n",
				s)
			nx.Stop()
			// Ideally, we should somehow wait for all subsystems to stop.
			return
		}
	}
} // func main()
