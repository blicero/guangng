// /home/krylon/go/src/github.com/blicero/guangng/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-02-07 16:56:15 krylon>

package database

import "github.com/blicero/guangng/database/query"

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
	query.HostGetCnt: `
SELECT COUNT(id) FROM host
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
	query.XFRAdd: `
INSERT INTO xfr (name, added)
         VALUES (   ?,     ?)
RETURNING id
`,
	query.XFRGetByName: `
SELECT
    id,
    added,
    COALESCE(start, -1),
    COALESCE(end, -1),
    status
FROM xfr
WHERE name = ?
`,
	query.XFRGetUnfinished: `
SELECT
    id,
    name,
    added,
    COALESCE(start, -1)
FROM xfr
WHERE end IS NULL
ORDER BY added
LIMIT ?
`,
	query.XFRGetCnt: "SELECT COUNT(id) FROM xfr",
	query.XFRStart:  "UPDATE xfr SET start = ? WHERE id = ?",
	query.XFRFinish: "UPDATE xfr SET end = ?, status = ? WHERE id = ?",
	query.ServiceAdd: `
INSERT INTO svc (host_id, port, success, response, timestamp)
         VALUES (      ?,    ?,       ?,        ?,         ?)
RETURNING id
`,
	query.ServiceGetByHost: `
SELECT
    id,
    port,
    success,
    COALESCE(response, ''),
    timestamp
FROM svc
WHERE host_id = ?
`,
	query.ServiceGetCnt: `
SELECT COUNT(id)
FROM svc
WHERE response IS NOT NULL
`,
	query.ServiceGetSuccess: `
SELECT
    id,
    host_id,
    port,
    success,
    response,
    timestamp
FROM svc
WHERE response IS NOT NULL AND response <> ''
`,
}
