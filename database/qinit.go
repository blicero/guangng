// /home/krylon/go/src/github.com/blicero/guangng/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-21 19:13:33 krylon>

package database

var qInit = []string{
	`
CREATE TABLE host (
    id INTEGER PRIMARY KEY,
    addr TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    added INTEGER NOT NULL,
    last_contact INTEGER NOT NULL DEFAULT 0,
    sysname TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    source INTEGER NOT NULL,
    CHECK (source BETWEEN 1 AND 5)
) STRICT
`,
	"CREATE INDEX host_contact_idx ON host (last_contact)",
	"CREATE UNIQUE INDEX host_addr_idx ON host (addr)",
	`
CREATE TABLE svc (
    id INTEGER PRIMARY KEY,
    host_id INTEGER NOT NULL,
    port INTEGER NOT NULL,
    success INTEGER NOT NULL,
    response TEXT,
    timestamp INTEGER NOT NULL,
    CHECK (port BETWEEN 1 AND 65535),
    FOREIGN KEY (host_id) REFERENCES host (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE
) STRICT
`,
	"CREATE INDEX svc_host_idx ON svc (host_id)",
	`
CREATE TRIGGER host_contact_tr
AFTER INSERT ON svc
BEGIN
    UPDATE host
    SET last_contact = unixepoch()
    WHERE id = NEW.host_id;
END
`,
	`
CREATE TABLE xfr (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    added INTEGER NOT NULL,
    start INTEGER,
    end INTEGER,
    status INTEGER NOT NULL DEFAULT 0,
    CHECK ((end IS NULL) OR (start IS NOT NULL))
) STRICT
`,
	"CREATE INDEX xfr_start_idx ON xfr (start)",
	"CREATE INDEX xfr_end_idx ON xfr (end)",
	"CREATE INDEX xfr_end_null_idx ON xfr (end IS NULL)",
}
