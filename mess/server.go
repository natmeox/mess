package mess

import (
	"database/sql"
	_ "github.com/bmizerany/pq"
	"log"
	"net"
)

var Config struct {
	Debug        bool
	Dsn          string
	GameAddress  string
	WebAddress   string
	CookieSecret string
}

func OpenDatabase() (*DatabaseWorld, error) {
	db, err := sql.Open("postgres", Config.Dsn)
	if err != nil {
		return nil, err
	}
	return &DatabaseWorld{db}, nil
}

func Server() {
	GameInit()

	go StartWeb()

	// TODO: listen on an SSL port too
	log.Println("Listening at address", Config.GameAddress)
	listener, err := net.Listen("tcp", Config.GameAddress)
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
