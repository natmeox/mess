package mess

import (
	"database/sql"
	"fmt"
	_ "github.com/bmizerany/pq"
	"log"
	"net"
)

var Config struct {
	Dsn  string
	Port uint16
}

var Db *sql.DB

func OpenDatabase() (err error) {
	Db, err = sql.Open("postgres", Config.Dsn)
	return
}

func Server() {
	GameInit()

	err := OpenDatabase()
	if err != nil {
		log.Println("Error connecting to database:", err)
		return
	}

	// TODO: listen on an SSL port too
	log.Println("Listening at port", Config.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", Config.Port))
	if err != nil {
		log.Println("Error listening for connections:", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting client:", err)
			continue
		}

		client := NewClientPump(conn)
		go WelcomeClient(client)
	}
}
