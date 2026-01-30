// /home/krylon/go/src/github.com/blicero/guangng/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-30 15:16:15 krylon>

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
	"github.com/blicero/guangng/web"
)

const (
	defaultACnt = 8
	defaultNCnt = 8
	defaultXCnt = 2
	defaultScnt = 4
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
		err                    error
		nx                     *nexus.Nexus
		srv                    *web.Server
		aCnt, nCnt, xCnt, sCnt int
		version                bool
		addr, defaultAddr      string
		delay                  int
	)

	defaultAddr = fmt.Sprintf("[::1]:%d", common.WebPort)

	flag.IntVar(&aCnt, "acnt", defaultACnt, "Number of address generator workers")
	flag.IntVar(&nCnt, "ncnt", defaultNCnt, "Number of name resolution workers")
	flag.IntVar(&xCnt, "xcnt", defaultXCnt, "Number of AXFR workers")
	flag.IntVar(&sCnt, "scnt", defaultScnt, "Number of scan workers")
	flag.BoolVar(&version, "version", false, "Display the version number and exit")
	flag.StringVar(&addr, "addr", defaultAddr, "Address for the web UI to listen on")
	flag.IntVar(&delay, "delay", 5, "Delay before starting all the moving parts")

	flag.Parse()

	if version {
		os.Exit(0)
	}

	if nx, err = nexus.New(aCnt, nCnt, xCnt, sCnt); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create Nexus: %s\n",
			err.Error())
		os.Exit(1)
	} else if srv, err = web.Create(addr, nx); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create web server: %s\n",
			err.Error())
		os.Exit(1)
	}

	fmt.Printf("WebUI is running on %s\n", addr)

	for i := range delay {
		fmt.Printf("\r%d                           ",
			(delay - (i + 1)))
		os.Stdout.Sync() // nolint: errcheck
		time.Sleep(time.Second)
	}

	fmt.Printf("\n")

	var (
		ticker = time.NewTicker(common.ActiveTimeout)
		sigQ   = make(chan os.Signal, 1)
	)

	defer ticker.Stop()

	signal.Notify(sigQ, os.Interrupt, syscall.SIGTERM)

	nx.Start()
	go srv.Run()

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
