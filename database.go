package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

type Conn struct {
	db *sql.DB
}

func NewConn(db *sql.DB) Conn {
	return Conn{
		db,
	}
}

func (conn *Conn) Start() (*Transaction, error) {
	tx, err := conn.db.Begin()
	if err != nil {
		return nil, err
	}

	inst := Transaction{
		tx,
	}

	return &inst, err
}

func SetupDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "app.db")
	if err != nil {
		log.Fatal(err)
	}

	bytes, err := os.ReadFile("./schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	schema := string(bytes)

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}

	return db
}
