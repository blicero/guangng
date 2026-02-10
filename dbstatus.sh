#!/bin/sh
# /home/krylon/go/src/github.com/blicero/guangng/dbstatus.sql
# Time-stamp: <2026-02-10 19:05:10 krylon>
# created on 07. 02. 2026 by Benjamin Walkenhorst
# (c) 2026 Benjamin Walkenhorst
# Use at your own risk!

sqlite3 ~/.guangng.d/guangng.db <<EOF
SELECT COUNT(id) AS host_cnt FROM host;

SELECT COUNT(id) AS svc_cnt FROM svc WHERE COALESCE(response, '') <> '';

SELECT COUNT(id) AS xfr_cnt FROM xfr WHERE status <> 0;

EOF

