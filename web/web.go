// /home/krylon/go/src/github.com/blicero/guangng/web/web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 26. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-26 16:32:29 krylon>

// Package web provides a web-based UI.
package web

import (
	"embed"
	"log"
	"sync"
	"sync/atomic"
	"text/template"

	"github.com/blicero/guangng/database"
	"github.com/gorilla/mux"
)

const (
	cacheControl = "max-age=3600, public"
	noCache      = "no-store, max-age=0"
)

//go:embed assets
var assets embed.FS

type Server struct {
	addr   string
	log    *log.Logger
	pool   *database.Pool
	lock   sync.RWMutex
	active atomic.Bool
	router *mux.Router
	tmpl   *template.Template
}
