package app

import (
	"database/sql"
	"time"
)

func NewDB() *sql.DB {
	connStr := "host=localhost port=5432 user=postgres password=root dbname=live_auction sslmode=disable"

	db, err := sql.Open("postgres", connStr)

	if err != nil {
		panic(err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(60 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	return db
}
