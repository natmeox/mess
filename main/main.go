package main

import (
	_ "database/sql"
	"encoding/json"
	"flag"
	_ "github.com/bmizerany/pq"
	"github.com/natmeox/mess"
	"log"
	"os"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "./config.json", "path to configuration file")

	configFile, err := os.Open(configPath)
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

	mess.Server()
}
