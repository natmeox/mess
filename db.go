package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	_ "github.com/mattn/go-sqlite3"
	"strings"
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
	schemaSql := string(schemaBytes)
	schemaStatements := strings.Split(schemaSql, ";\n")

	for _, schemaStatement := range schemaStatements {
		_, err = db.Exec(schemaStatement)
		if err != nil {
			return
		}
	}

	return
}

func CloseDatabase() {
	if db == nil {
		return
	}
	db.Close()
	db = nil;
}
