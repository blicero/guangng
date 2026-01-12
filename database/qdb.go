// /home/krylon/go/src/github.com/blicero/guangng/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-12 16:02:38 krylon>

package database

import "github.com/blicero/guangng/database/query"

// nolint: unused
var qdb = map[query.ID]string{
	query.HostAdd: `
INSERT INTO host (addr, name, added, source)
          VALUES (   ?,    ?,     ?,      ?)
RETURNING id
`,
	query.HostGetByID: `
SELECT
    addr,
    name,
    added,
    last_contact,
    sysname,
    location,
    source
FROM host
WHERE id = ?
`,
	query.HostGetByAddr: `
SELECT
    id,
    name,
    added,
    last_contact,
    sysname,
    location,
    source
FROM host
WHERE addr = ?
`,
	query.HostGetAll: `
SELECT
    id,
    addr,
    name,
    added,
    last_contact,
    sysname,
    location,
    source
FROM host
LIMIT ?
`,
	query.HostGetRandom: `
SELECT id,
       addr,
       name,
       added,
       last_contact,
       sysname,
       location,
       source
FROM host
LIMIT ?
OFFSET ABS(RANDOM()) % MAX((SELECT COUNT(*) FROM host), 1)
`,
	query.HostUpdateSysname: `
UPDATE host
SET sysname = ?
WHERE id = ?
`,
	query.HostUpdateLocation: `
UPDATE host
SET location = ?
WHERE id = ?
`,
}
