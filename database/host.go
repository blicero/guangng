// /home/krylon/go/src/github.com/blicero/guangng/database/host.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-02-11 15:03:02 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/blicero/guangng/database/query"
	"github.com/blicero/guangng/model"
)

// HostAdd adds a Host to the Database.
func (db *Database) HostAdd(host *model.Host) error {
	const qid query.ID = query.HostAdd
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
	if rows, err = stmt.Query(host.AStr(), host.Name, now.Unix(), host.Source); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot add Host %s/%s to database: %w",
				host.Name,
				host.AStr(),
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
			var ex = fmt.Errorf("failed to get ID for newly added host %s/%s: %w",
				host.Name,
				host.AStr(),
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		host.ID = id
		host.Added = now
		return nil
	}
} // func (db *Database) HostAdd(host *model.Host) error

// HostGetByID looks up a Host by its ID.
func (db *Database) HostGetByID(id int64) (*model.Host, error) {
	const qid query.ID = query.HostGetByID
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
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			added, contact int64
			addr           string
			host           = &model.Host{ID: id}
		)

		if err = rows.Scan(&addr, &host.Name, &added, &contact, &host.Sysname, &host.Location, &host.Source); err != nil {
			var ex = fmt.Errorf("failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if host.Addr = net.ParseIP(addr); host.Addr == nil {
			err = fmt.Errorf("could not parse IP address %q",
				addr)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return nil, err
		}

		host.Added = time.Unix(added, 0)
		host.LastContact = time.Unix(contact, 0)
		return host, nil
	}

	return nil, nil
} // func (db *Database) HostGetByID(id int64) (*model.Host, error)

// HostGetMap returns a map of all Hosts, using their IDs as keys.
func (db *Database) HostGetMap() (map[int64]*model.Host, error) {
	const qid query.ID = query.HostGetAll

	var err error
	var msg string
	var stmt *sql.Stmt
	var hosts map[int64]*model.Host

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

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(-1); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying all hosts: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	hosts = make(map[int64]*model.Host)

	for rows.Next() {
		var added, contact int64
		var addrStr string
		var host = new(model.Host)

		if err = rows.Scan(
			&host.ID,
			&addrStr,
			&host.Name,
			&added,
			&contact,
			&host.Sysname,
			&host.Location,
			&host.Source); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		host.Addr = net.ParseIP(addrStr)
		host.Added = time.Unix(added, 0)
		host.LastContact = time.Unix(contact, 0)
		// hosts = append(hosts, host)
		hosts[host.ID] = host
	}

	return hosts, nil
} // func (db *Database) HostGetMap() (map[int64]*model.Host, error)

// HostGetRandom returns up to <max> Hosts randomly picked from the database.
func (db *Database) HostGetRandom(max int) ([]*model.Host, error) {
	const qid query.ID = query.HostGetRandom
	var err error
	var msg string
	var stmt *sql.Stmt
	var hosts []*model.Host

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

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(max); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying %d random hosts: %s",
				max, err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	hosts = make([]*model.Host, 0, max)

	for rows.Next() {
		var added, contact int64
		var addrStr string
		var host = new(model.Host)

		if err = rows.Scan(
			&host.ID,
			&addrStr,
			&host.Name,
			&added,
			&contact,
			&host.Sysname,
			&host.Location,
			&host.Source); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		host.Addr = net.ParseIP(addrStr)
		host.Added = time.Unix(added, 0)
		host.LastContact = time.Unix(contact, 0)
		hosts = append(hosts, host)
	}

	return hosts, nil
} // func (db *Database) HostGetRandom(n int) ([]*model.Host, error)

// HostGetMissingLocation returns a slice of all Hosts that have no location.
func (db *Database) HostGetMissingLocation() ([]*model.Host, error) {
	const qid query.ID = query.HostGetMissingLocation
	var err error
	var msg string
	var stmt *sql.Stmt
	var hosts []*model.Host

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

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying hosts without location: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	hosts = make([]*model.Host, 0, 16)

	for rows.Next() {
		var added, contact int64
		var addrStr string
		var host = new(model.Host)

		if err = rows.Scan(
			&host.ID,
			&addrStr,
			&host.Name,
			&added,
			&contact,
			&host.Sysname,
			&host.Location,
			&host.Source); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		host.Addr = net.ParseIP(addrStr)
		host.Added = time.Unix(added, 0)
		host.LastContact = time.Unix(contact, 0)
		hosts = append(hosts, host)
	}

	return hosts, nil
} // func (db *Database) HostGetMissingLocation() ([]*model.Host, error)

// HostGetCnt returns the number of Hosts in the Database.
func (db *Database) HostGetCnt() (int64, error) {
	const qid query.ID = query.HostGetCnt
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
} // func (db *Database) HostGetCnt() (int64, error)

// HostUpdateLocation sets a Host's location.
func (db *Database) HostUpdateLocation(h *model.Host, location string) error {
	const qid query.ID = query.HostUpdateLocation
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

EXEC_QUERY:
	if _, err = stmt.Exec(location, h.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot update location of Host %s/%s: %w",
				h.Name,
				h.AStr(),
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	h.Location = location
	return nil
} // func (db *Database) HostUpdateLocation(h *model.Host, location string) error

// HostUpdateSysname sets a Host's sysname.
func (db *Database) HostUpdateSysname(h *model.Host, sysname string) error {
	const qid query.ID = query.HostUpdateSysname
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

EXEC_QUERY:
	if _, err = stmt.Exec(sysname, h.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot update sysname of Host %s/%s: %w",
				h.Name,
				h.AStr(),
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	h.Sysname = sysname
	return nil
} // func (db *Database) HostUpdateSysname(h *model.Host, sysname string) error
