// /home/krylon/go/src/github.com/blicero/guangng/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-12 14:25:07 krylon>

package guangng

import (
	"fmt"

	"github.com/blicero/guangng/common"
)

func main() {
	fmt.Printf("%s %s, built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))
	fmt.Println("Nothing to see here, move along...")
} // func main()
