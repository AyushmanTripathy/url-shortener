package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/glebarez/go-sqlite"
)

var db *sql.DB

const createTableQuery = 
	`CREATE TABLE IF NOT EXISTS redirects (
		code TEXT PRIMARY KEY,
		url TEXT NOT NULL
	);`

const insertRedirectQuery =
	`INSERT INTO redirects (code, url) VALUES (?, ?);`

const getRedirectURLQuery = 
	`SELECT url FROM redirects WHERE code = ?;`

func InitializeDBConnection() {
	dbUrl, found := os.LookupEnv("DB_URL")
	if !found {
		log.Fatal("DB_URL not found")
	}

	var err error
	if db, err = sql.Open("sqlite", dbUrl); err != nil {
		log.Fatal("Failed to connect to DB", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping DB", err)
	}
	log.Print("DB Ping successfull")

	if _, err = db.Exec(createTableQuery); err != nil {
		log.Fatal("Table creation failed", err)
	}
}
