package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDatabase(filename string) (err error) {
	db, err = sql.Open("sqlite3", filename)
	if err != nil {
		return
	}

	// Is there a schema in there?
	_, selectErr := db.Exec("select 1 from object")
	if selectErr == nil {
		return
	}
	log.Println("Database is missing tables so repeating the schema")

	schemaBytes, err := ioutil.ReadFile("schema.sql")
	if err != nil {
		return
	}

	_, err = db.Exec(string(schemaBytes))
	return
}

func CloseDatabase() {
	if db == nil {
		return
	}
	db.Close()
	db = nil;
}
