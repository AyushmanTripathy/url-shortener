package internal

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var Db *sql.DB

const InsertRedirectQuery = `INSERT INTO redirects (code, url) VALUES ($1, $2);`

const GetRedirectURLQuery = `SELECT url FROM redirects WHERE code = $1;`

const UpdateRedirectURLQuery = `UPDATE redirects SET url = $2 WHERE code = $1;`

func InitializeDBConnection() {
	dbUrl, found := os.LookupEnv("DB_URL")
	if !found {
		log.Fatal("DB_URL not found")
	}

	var err error
	if Db, err = sql.Open("pgx", dbUrl); err != nil {
		log.Fatal("Failed to connect to DB", err)
	}

	if err := Db.Ping(); err != nil {
		log.Fatal("Failed to ping DB", err)
	}
	log.Print("DB Ping successfull")
}
