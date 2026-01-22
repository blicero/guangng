// /home/krylon/go/src/github.com/blicero/guangng/database/service.go
// -*- mode: go; coding: utf-8; -*-
// Created on 22. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-22 18:46:03 krylon>

package database

// func (db *Database) ServiceAdd(h *model.Host, s *model.Service) error {
// 	const qid query.ID = query.ServiceAdd
// 	var (
// 		err  error
// 		stmt *sql.Stmt
// 	)

// 	if stmt, err = db.getQuery(qid); err != nil {
// 		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
// 			qid,
// 			err.Error())
// 		panic(err)
// 	} else if db.tx != nil {
// 		stmt = db.tx.Stmt(stmt)
// 	}

// 	var (
// 		rows *sql.Rows
// 		now  = time.Now()
// 	)

// EXEC_QUERY:
// 	if rows, err = stmt.Query(host.AStr(), host.Name, now.Unix(), host.Source); err != nil {
// 		if worthARetry(err) {
// 			waitForRetry()
// 			goto EXEC_QUERY
// 		} else {
// 			err = fmt.Errorf("cannot add Host %s/%s to database: %w",
// 				host.Name,
// 				host.AStr(),
// 				err)
// 			db.log.Printf("[ERROR] %s\n", err.Error())
// 			return err
// 		}
// 	} else {
// 		var id int64

// 		defer rows.Close() // nolint: errcheck

// 		if !rows.Next() {
// 			// CANTHAPPEN
// 			db.log.Printf("[ERROR] Query %s did not return a value\n",
// 				qid)
// 			return fmt.Errorf("query %s did not return a value", qid)
// 		} else if err = rows.Scan(&id); err != nil {
// 			var ex = fmt.Errorf("failed to get ID for newly added host %s/%s: %w",
// 				host.Name,
// 				host.AStr(),
// 				err)
// 			db.log.Printf("[ERROR] %s\n", ex.Error())
// 			return ex
// 		}

// 		host.ID = id
// 		host.Added = now
// 		return nil
// 	}
// }
// func (db *Database) ServiceAdd(h *model.Host, s *model.Service) error
