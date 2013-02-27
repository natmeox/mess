package mess

import (
	_ "database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/bmizerany/pq"
	"log"
	"os"
)

type ConfigStash struct {
	Dsn  string
	Port uint16
}

var Config ConfigStash

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "./config.json", "path to configuration file")

	configFile, err := os.Open(configPath)
	if err != nil {
		log.Println(err)
		return
	}
	dec := json.NewDecoder(configFile)
	err = dec.Decode(&Config)
	if err != nil {
		log.Println("Error decoding configuration:", err)
		return
	}

	Server()
}
