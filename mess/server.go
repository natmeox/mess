package mess

import (
	"fmt"
	"log"
	"net"
)

var Config struct {
	Dsn  string
	Port uint16
}

func Server() {
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
