// /home/krylon/go/src/github.com/blicero/guangng/database/xfr.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-15 21:13:38 krylon>

package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/blicero/guangng/database/query"
	"github.com/blicero/guangng/model"
)

func (db *Database) XFRAdd(zone *model.Zone) error {
	const qid query.ID = query.XFRAdd
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(zone.Name, zone.Added.Unix()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot add zone %s to database: %w",
				zone.Name,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close() // nolint: errcheck

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			var ex = fmt.Errorf("failed to get ID for newly added zone %s: %w",
				zone.Name,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		zone.ID = id
		return nil
	}
} // func (db *Database) XFRAdd(zone *model.Zone) error

func (db *Database) XFRGetByName(name string) (*model.Zone, error) {
	const qid query.ID = query.XFRGetByName
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(name); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			added, start, finish int64
			zone                 = &model.Zone{Name: name}
		)

		if err = rows.Scan(&zone.ID, &added, &start, &finish, &zone.Status); err != nil {
			var ex = fmt.Errorf("failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		zone.Added = time.Unix(added, 0)
		if start != -1 {
			zone.Started = time.Unix(start, 0)
		}
		if finish != -1 {
			zone.Finished = time.Unix(finish, 0)
		}

		return zone, nil
	}

	return nil, nil
} // func (db *Database) XFRGetByName(name string) (*model.Zone, error)
