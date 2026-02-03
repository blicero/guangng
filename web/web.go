// /home/krylon/go/src/github.com/blicero/guangng/web/web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 26. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-02-03 17:44:22 krylon>

// Package web provides a web-based UI.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/database"
	"github.com/blicero/guangng/logdomain"
	"github.com/blicero/guangng/model/subsystem"
	"github.com/blicero/guangng/nexus"
	"github.com/gorilla/mux"
)

const (
	cacheControl = "max-age=3600, public"
	noCache      = "no-store, max-age=0"
	tmplFolder   = "assets/templates"
)

//go:embed assets
var assets embed.FS

// Server provides a web-based UI
type Server struct {
	addr      string
	log       *log.Logger
	pool      *database.Pool // nolint: unused
	lock      sync.RWMutex   // nolint: unused
	active    atomic.Bool
	router    *mux.Router
	tmpl      *template.Template
	web       http.Server
	mimeTypes map[string]string
	nx        *nexus.Nexus
}

// Create returns a new web Server.
func Create(addr string, nx *nexus.Nexus) (*Server, error) {
	var (
		err error
		msg string
		srv = &Server{
			addr: addr,
			nx:   nx,
			mimeTypes: map[string]string{
				".css":  "text/css",
				".map":  "application/json",
				".js":   "text/javascript",
				".png":  "image/png",
				".jpg":  "image/jpeg",
				".jpeg": "image/jpeg",
				".webp": "image/webp",
				".gif":  "image/gif",
				".json": "application/json",
				".html": "text/html",
			},
		}
	)

	if srv.log, err = common.GetLogger(logdomain.Web); err != nil {
		return nil, err
	} else if srv.pool, err = database.NewPool(4); err != nil {
		srv.log.Printf("[CRITICAL] Cannot open database pool: %s\n",
			err.Error())
		return nil, err
	}

	var templates []fs.DirEntry
	var tmplRe = regexp.MustCompile("[.]tmpl$")

	if templates, err = assets.ReadDir(tmplFolder); err != nil {
		srv.log.Printf("[ERROR] Cannot read embedded templates: %s\n",
			err.Error())
		return nil, err
	}

	srv.tmpl = template.New("").Funcs(funcmap)
	for _, entry := range templates {
		var (
			content []byte
			path    = filepath.Join(tmplFolder, entry.Name())
		)

		if !tmplRe.MatchString(entry.Name()) {
			continue
		} else if content, err = assets.ReadFile(path); err != nil {
			msg = fmt.Sprintf("Cannot read embedded file %s: %s",
				path,
				err.Error())
			srv.log.Printf("[CRITICAL] %s\n", msg)
			return nil, errors.New(msg)
		} else if srv.tmpl, err = srv.tmpl.Parse(string(content)); err != nil {
			msg = fmt.Sprintf("Could not parse template %s: %s",
				entry.Name(),
				err.Error())
			srv.log.Println("[CRITICAL] " + msg)
			return nil, errors.New(msg)
		} else if common.Debug {
			srv.log.Printf("[TRACE] Template \"%s\" was parsed successfully.\n",
				entry.Name())
		}
	}

	srv.router = mux.NewRouter()
	srv.web.Addr = addr
	srv.web.ErrorLog = srv.log
	srv.web.Handler = srv.router

	// Register URL handlers
	srv.router.HandleFunc("/favicon.ico", srv.handleFavIco)
	srv.router.HandleFunc("/static/{file}", srv.handleStaticFile)
	srv.router.HandleFunc("/{index:(?i:index|main|start)$}", srv.handleMain)

	// AJAX Handlers
	srv.router.HandleFunc(
		"/ajax/worker_count",
		srv.handleLoadWorkerCount)
	srv.router.HandleFunc(
		"/ajax/spawn_worker/{subsys:(?:\\d+)}/{cnt:(?:\\d+)$}",
		srv.handleSpawnWorker)
	srv.router.HandleFunc(
		"/ajax/stop_worker/{subsys:(?:\\d+)}/{cnt:(?:\\d+)$}",
		srv.handleStopWorker)

	srv.router.HandleFunc(
		"/ajax/beacon",
		srv.handleBeacon)

	return srv, nil
} // func Create(addr string, nx *nexus.Nexus) (*Server, error)

// IsActive returns the Server's active flag.
func (srv *Server) IsActive() bool {
	return srv.active.Load()
} // func (srv *Server) IsActive() bool

// Stop clears the Server's active flag.
func (srv *Server) Stop() {
	srv.active.Store(false)
} // func (srv *Server) Stop()

// Run executes the Server's loop, waiting for new connections and starting
// goroutines to handle them.
func (srv *Server) Run() {
	var err error

	defer srv.log.Println("[INFO] Web server is shutting down")

	srv.active.Store(true)
	defer srv.active.Store(false)

	srv.log.Printf("[INFO] Web frontend is going online at %s\n", srv.addr)
	http.Handle("/", srv.router)

	if err = srv.web.ListenAndServe(); err != nil {
		if err.Error() != "http: Server closed" {
			srv.log.Printf("[ERROR] ListenAndServe returned an error: %s\n",
				err.Error())
		} else {
			srv.log.Println("[INFO] HTTP Server has shut down.")
		}
	}
} // func (srv *Server) Run()

//////////////////////////////////////////////////////////////////////////////
/// Handle requests //////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////

func (srv *Server) handleMain(w http.ResponseWriter, req *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		req.URL.EscapedPath())

	const tmplName = "main"

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataIndex{
			tmplDataBase: tmplDataBase{
				Title:      "Main",
				Debug:      common.Debug,
				URL:        req.URL.String(),
				Subsystems: subsystem.AllSubsystems(),
			},
			GenActive:  srv.nx.GetActiveFlag(subsystem.Generator),
			XFRActive:  srv.nx.GetActiveFlag(subsystem.XFR),
			ScanActive: srv.nx.GetActiveFlag(subsystem.Scanner),
			GenAddrCnt: srv.nx.GetWorkerCount(subsystem.GeneratorAddress),
			GenNameCnt: srv.nx.GetWorkerCount(subsystem.GeneratorName),
			XFRCnt:     srv.nx.GetWorkerCount(subsystem.XFR),
			ScanCnt:    srv.nx.GetWorkerCount(subsystem.Scanner),
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.HostCnt, err = db.HostGetCnt(); err != nil {
		srv.log.Printf("[ERROR] Failed to get number of Hosts from Database: %s\n",
			err.Error())
	} else if data.ZoneCnt, err = db.XFRGetCnt(); err != nil {
		srv.log.Printf("[ERROR] Failed to get number of zone transfers from Database: %s\n",
			err.Error())
	} else if data.PortCnt, err = db.ServiceGetCnt(); err != nil {
		srv.log.Printf("[ERROR] Failed to get number of scanned ports from Database: %s\n",
			err.Error())
	}

	w.Header().Set("Cache-Control", cacheControl)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		// srv.SendMessage(msg)
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleMain(w http.ResponseWriter, req *http.Request)

//////////////////////////////////////////////////////////////////////////////
/// AJAX handlers ////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////

func (srv *Server) handleLoadWorkerCount(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err error
		res = ajaxWorkerCnt{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	res.GeneratorAddress = srv.nx.GetWorkerCount(subsystem.GeneratorAddress)
	res.GeneratorName = srv.nx.GetWorkerCount(subsystem.GeneratorName)
	res.XFR = srv.nx.GetWorkerCount(subsystem.XFR)
	res.Scanner = srv.nx.GetWorkerCount(subsystem.Scanner)
	res.Status = true

	var outbuf []byte

	if outbuf, err = json.Marshal(&res); err != nil {
		res.Message = fmt.Sprintf("Error serializing Response to %s: %s",
			r.RemoteAddr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
	}

	srv.log.Printf("[DEBUG] Return worker count:\n%s\n\n",
		outbuf)

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(outbuf)), 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(200)
	w.Write(outbuf) // nolint: errcheck
} // func (srv *Server) handleLoadWorkerCount(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleSpawnWorker(w http.ResponseWriter, r *http.Request) {
	var (
		err            error
		facStr, cntStr string
		cnt, facID     int64
		fac            subsystem.ID
		res            = ajaxCtlResponse{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)

	vars := mux.Vars(r)

	facStr = vars["subsys"]
	cntStr = vars["cnt"]

	if cnt, err = strconv.ParseInt(cntStr, 10, 64); err != nil {
		res.Message = fmt.Sprintf("Cannot parse number of workers to spawn (%q): %s",
			cntStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	} else if facID, err = strconv.ParseInt(facStr, 10, 8); err != nil {
		res.Message = fmt.Sprintf("Cannot parse facility ID %q: %s",
			facStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	}

	fac = subsystem.ID(facID)

	srv.log.Printf("[DEBUG] Spawn %d %s workers.\n",
		cnt,
		fac)

	for range cnt {
		srv.nx.StartOne(fac)
	}

	res.Status = true
	res.Message = fmt.Sprintf("Started %d workers in %s", cnt, fac)
	res.NewCnt = srv.nx.GetWorkerCount(fac)

RESPOND:
	var outbuf []byte

	if outbuf, err = json.Marshal(&res); err != nil {
		res.Message = fmt.Sprintf("Error serializing Response to %s: %s",
			r.RemoteAddr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
	}

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(outbuf)), 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(200)
	w.Write(outbuf) // nolint: errcheck
} // func handleSpawnWorker(w http.ResponseWriter, req *http.Request)

func (srv *Server) handleStopWorker(w http.ResponseWriter, r *http.Request) {
	var (
		err            error
		facStr, cntStr string
		cnt, facID     int64
		fac            subsystem.ID
		res            = ajaxCtlResponse{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)

	vars := mux.Vars(r)

	facStr = vars["subsys"]
	cntStr = vars["cnt"]

	if cnt, err = strconv.ParseInt(cntStr, 10, 64); err != nil {
		res.Message = fmt.Sprintf("Cannot parse number of workers to spawn (%q): %s",
			cntStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	} else if facID, err = strconv.ParseInt(facStr, 10, 8); err != nil {
		res.Message = fmt.Sprintf("Cannot parse facility ID %q: %s",
			facStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	}

	fac = subsystem.ID(facID)

	srv.log.Printf("[DEBUG] Stop %d %s workers.\n",
		cnt,
		fac)

	for range cnt {
		srv.nx.StopOne(fac)
	}
	res.Status = true
	res.Message = fmt.Sprintf("Started one worker in %s", fac)

RESPOND:
	var outbuf []byte

	if outbuf, err = json.Marshal(&res); err != nil {
		res.Message = fmt.Sprintf("Error serializing Response to %s: %s",
			r.RemoteAddr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
	}

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(outbuf)), 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(200)
	w.Write(outbuf) // nolint: errcheck
} // func (srv *Server) handleStopWorker(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
	// It doesn't bother me enough to do anything about it other
	// than writing this comment, but this method is probably
	// grossly inefficient re memory.
	var timestamp = time.Now().Format(common.TimestampFormat)
	const appName = common.AppName + " " + common.Version
	var jstr = fmt.Sprintf(`{ "Status": true, "Message": "%s", "Timestamp": "%s", "Hostname": "%s" }`,
		appName,
		timestamp,
		hostname())
	var response = []byte(jstr)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.WriteHeader(200)
	w.Write(response) // nolint: errcheck,gosec
} // func (srv *WebFrontend) handleBeacon(w http.ResponseWriter, r *http.Request)

//////////////////////////////////////////////////////////////////////////////
/// Handle static assets /////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////

func (srv *Server) handleFavIco(w http.ResponseWriter, request *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		request.URL.EscapedPath())

	const (
		filename = "assets/static/favicon.ico"
		mimeType = "image/vnd.microsoft.icon"
	)

	w.Header().Set("Content-Type", mimeType)

	if !common.Debug {
		w.Header().Set("Cache-Control", "max-age=7200")
	} else {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
	}

	var (
		err error
		fh  fs.File
	)

	if fh, err = assets.Open(filename); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", filename)
		srv.sendErrorMessage(w, msg)
	} else {
		defer fh.Close() // nolint: errcheck
		w.WriteHeader(200)
		io.Copy(w, fh) // nolint: errcheck
	}
} // func (srv *Server) handleFavIco(w http.ResponseWriter, request *http.Request)

func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request) {
	// srv.log.Printf("[TRACE] Handle request for %s\n",
	// 	request.URL.EscapedPath())

	// Since we controll what static files the server has available, we
	// can easily map MIME type to slice. Soon.

	vars := mux.Vars(request)
	filename := vars["file"]
	path := filepath.Join("assets", "static", filename)

	var mimeType string

	srv.log.Printf("[TRACE] Delivering static file %s to client %s\n",
		filename,
		request.RemoteAddr)

	var match []string

	if match = common.SuffixPattern.FindStringSubmatch(filename); match == nil {
		mimeType = "text/plain"
	} else if mime, ok := srv.mimeTypes[match[1]]; ok {
		mimeType = mime
	} else {
		srv.log.Printf("[ERROR] Did not find MIME type for %s\n", filename)
	}

	w.Header().Set("Content-Type", mimeType)

	if common.Debug {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
	} else {
		w.Header().Set("Cache-Control", "max-age=7200")
	}

	var (
		err error
		fh  fs.File
	)

	if fh, err = assets.Open(path); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", path)
		srv.sendErrorMessage(w, msg)
	} else {
		defer fh.Close() // nolint: errcheck
		w.WriteHeader(200)
		io.Copy(w, fh) // nolint: errcheck
	}
} // func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request)

func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string) {
	html := `
<!DOCTYPE html>
<html>
  <head>
    <title>Internal Error</title>
  </head>
  <body>
    <h1>Internal Error</h1>
    <hr />
    We are sorry to inform you an internal application error has occured:<br />
    %s
    <p>
    Back to <a href="/index">Homepage</a>
    <hr />
    &copy; 2018 <a href="mailto:krylon@gmx.net">Benjamin Walkenhorst</a>
  </body>
</html>
`

	srv.log.Printf("[ERROR] %s\n", msg)

	output := fmt.Sprintf(html, msg)
	w.WriteHeader(500)
	_, _ = w.Write([]byte(output)) // nolint: gosec
} // func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string)
