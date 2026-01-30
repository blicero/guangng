// /home/krylon/go/src/github.com/blicero/guangng/database/service.go
// -*- mode: go; coding: utf-8; -*-
// Created on 22. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-30 17:28:29 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/blicero/guangng/database/query"
	"github.com/blicero/guangng/model"
)

// ServiceAdd adds a scanned port and the result to the database.
func (db *Database) ServiceAdd(h *model.Host, s *model.Service) error {
	const qid query.ID = query.ServiceAdd
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

	var (
		rows *sql.Rows
		now  = time.Now()
	)

EXEC_QUERY:
	if rows, err = stmt.Query(s.HostID, s.Port, s.Success, s.Response, now.Unix()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot add Service %s:%d to database: %w",
				h.AStr(),
				s.Port,
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
			var ex = fmt.Errorf("failed to get ID for newly added %s:%d: %w",
				h.AStr(),
				s.Port,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		s.ID = id
		s.Timestamp = now
		return nil
	}
} // func (db *Database) ServiceAdd(h *model.Host, s *model.Service) error

// ServiceGetByHost retrieves all ports that have been scanned for Host <h>,
// and the reply we've receveiced.
func (db *Database) ServiceGetByHost(h *model.Host) (map[uint16]*model.Service, error) {
	const qid query.ID = query.ServiceGetByHost
	var err error
	var msg string
	var stmt *sql.Stmt

GET_QUERY:
	if stmt, err = db.getQuery(qid); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			db.log.Printf("[ERROR] Error getting query %s: %s",
				qid,
				err.Error())
			return nil, err
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var (
		rows  *sql.Rows
		ports = make(map[uint16]*model.Service)
	)

EXEC_QUERY:
	if rows, err = stmt.Query(h.ID); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying services for Host %s (%s): %s",
				h.Name,
				h.AStr(),
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	for rows.Next() {
		var (
			svc          = &model.Service{HostID: h.ID}
			tstamp, port int64
		)

		if err = rows.Scan(
			&svc.ID,
			&port,
			&svc.Success,
			&tstamp); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		svc.Port = uint16(port)
		svc.Timestamp = time.Unix(tstamp, 0)
		ports[svc.Port] = svc
	}

	return ports, nil
} // func (db *Database) ServiceGetByHost(h *model.Host) (map[uint16]*model.Service, error)

// ServiceGetCnt returns the total number of scanned ports.
func (db *Database) ServiceGetCnt() (int64, error) {
	const qid query.ID = query.ServiceGetCnt
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return -1, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return -1, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var cnt int64
		if err = rows.Scan(&cnt); err != nil {
			var ex = fmt.Errorf("failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return -1, nil
		}

		return cnt, nil
	}

	return -1, nil
} // func (db *Database) ServiceGetCnt() (int64, error)
