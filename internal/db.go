package internal

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var Db *sql.DB

const insertRedirectQuery = `INSERT INTO redirects (code, url) VALUES ($1, $2);`

const GetRedirectURLQuery = `SELECT url FROM redirects WHERE code = $1;`

const updateRedirectURLQuery = `UPDATE redirects SET url = $2 WHERE code = $1;`

const insertHistoryQuery = `INSERT INTO history (code, url) VALUES ($1, $2)`

const InsertLogQuery = `INSERT INTO logs (code, edge, user_agent) VALUES ($1, $2, $3)`

func InsertRedirectEntry(code string, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, insertRedirectQuery, code, url)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, insertHistoryQuery, code, url)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func UpdateRedirectEntry(code string, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, updateRedirectURLQuery, code, url)
	if err != nil {
		return err
	}

	if rows, err := res.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("Alias entry not found")
	}

	_, err = tx.ExecContext(ctx, insertHistoryQuery, code, url)
	if err != nil {
		return err
	}

	return tx.Commit()
}

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
