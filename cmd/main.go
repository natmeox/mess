package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	_ "github.com/bmizerany/pq"
	"github.com/natmeox/mess"
	"log"
	"os"
	"path/filepath"
)

func installDatabase() {
	data, err := Asset("mess.sql")
	if err != nil {
		log.Println("Error finding SQL:", err)
		return
	}

	db, err := sql.Open("postgres", mess.Config.Dsn)
	if err != nil {
		log.Println("Error opening database:", err)
		return
	}

	_, err = db.Exec(string(data))
	if err != nil {
		log.Println("Error executing SQL:", err)
		return
	}

	log.Println("Yay! The database was created. Now run `mess` to start the server.")
}

func installSite() {
	for _, assetname := range AssetNames() {
		// Don't write out mess.sql, that's for installNewDatabase() to use.
		if assetname == "mess.sql" {
			continue
		}

		data, err := Asset(assetname)
		if err != nil {
			log.Println("Error reading internal asset", assetname, ":", err)
			return
		}

		// Write out config.json.sample as config.json.
		if assetname == "config.json.sample" {
			assetname = "config.json"
		}
		assetpath := filepath.Join(".", assetname)
		assetdir := filepath.Dir(assetpath)
		err = os.MkdirAll(assetdir, os.FileMode(0755))
		if err != nil {
			log.Println("Error creating directory", assetdir, "for assets:", err)
			return
		}

		assetfile, err := os.Create(assetpath)
		if err != nil {
			log.Println("Error opening asset file", assetpath, ":", err)
			return
		}

		_, err = assetfile.Write(data)
		if err != nil {
			log.Println("Error writing asset file", assetpath, ":", err)
			assetfile.Close()
			return
		}

		assetfile.Close()
	}

	log.Println("Installed files for your new mess site.")
	log.Println("Edit the config.json file with your database address, then run `mess --new-database` to set up the database.")
}

func main() {
	var configPath string
	var newSite bool
	var newDatabase bool
	flag.StringVar(&configPath, "config", "./config.json", "path to configuration file")
	flag.BoolVar(&newSite, "new-site", false, "install a new site & exit")
	flag.BoolVar(&newDatabase, "new-database", false, "install a new database & exit")

	flag.Parse()

	if newSite {
		installSite()
		return
	}

	configFile, err := os.Open(configPath)
	if _, ok := err.(*os.PathError); ok {
		log.Println("Could not open the configuration file", configPath, ".")
		log.Println("Run `mess --new-site` to install a default site.")
		return
	}
	if err != nil {
		log.Println(err)
		return
	}
	dec := json.NewDecoder(configFile)
	err = dec.Decode(&mess.Config)
	if err != nil {
		log.Println("Error decoding configuration:", err)
		return
	}

	if newDatabase {
		installDatabase()
		return
	}

	mess.Server()
}
