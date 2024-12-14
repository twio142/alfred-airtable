package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Implement a simple cache using SQLite

type Metadata struct {
	cached_at  time.Time
	tags       []string
	categories []string
}

type Link struct {
	name          string
	description   string
	url           string
	category      string
	tags          []string
	created       time.Time
	last_modified time.Time
	record_url    string
	id            string
	done          bool
	lists         []List
}

type List struct {
	name          string
	description   string
	links         []Link
	status        string // To Do, In Progress, Done
	created       time.Time
	last_modified time.Time
	record_url    string
	id            string
}

const dbFile = "cache.db"

// Initialize the database
func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func setCache(key, value string) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()

	insertQuery := `
	INSERT INTO cache (key, value) VALUES (?, ?)
	ON CONFLICT(key) DO UPDATE SET value=excluded.value, timestamp=CURRENT_TIMESTAMP;
	`
	_, err = db.Exec(insertQuery, key, value)
	if err != nil {
		return err
	}

	return nil
}

func getCache(key string) (string, error) {
	db, err := initDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	selectQuery := `
	SELECT value FROM cache WHERE key = ?;
	`
	var value string
	err = db.QueryRow(selectQuery, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	return value, nil
}

func clearCache() error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()

	deleteQuery := `
	DELETE FROM cache;
	`
	_, err = db.Exec(deleteQuery)
	if err != nil {
		return err
	}

	return nil
}

