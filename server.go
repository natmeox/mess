package mess

import (
	"net"
)

func Server() {
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

		client := NewClient(conn)
		conn.Close()
	}
}
