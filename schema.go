package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type finDB struct {
	db *sql.DB
}

func NewFinDB() (*finDB, error) {
	var f finDB
	var err error

	f.db, err = sql.Open("sqlite", "./fin.db")
	if err != nil {
		return nil, err
	}
	f.db.Ping()

	err = f.seedDatabase()
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (f finDB) seedDatabase() error {
	_, err = f.db.Exec(`
		CREATE TABLE if not exists accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id TEXT NOT NULL,
			balance REAL NOT NULL,
			name TEXT NOT NULL
		)
	;`)

	if err != nil {
		return err
	}

	_, err = f.db.Exec(`
		CREATE TABLE if not exists tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			access_token TEXT NOT NULL,
			item_id TEXT NOT NULL,
			request_id TEXT NOT NULL
		)
	;`)

	if err != nil {
		return err
	}

	_, err = f.db.Exec(`
		CREATE TABLE if not exists transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			amount REAL NOT NULL,
			account TEXT NOT NULL,
			merchant TEXT NULL,
			date TEXT NOT NULL,
			category TEXT NOT NULL
		)
	;`)

	if err != nil {
		return err
	}

	return nil
}
