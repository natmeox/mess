package mess

import (
	"bufio"
	"log"
	"net"
	"strings"
)

type ClientPump struct {
	ToClient chan string
	ToServer chan string

	conn net.Conn
}

var clients map[net.Conn]*ClientPump = make(map[net.Conn]*ClientPump)

func NewClientPump(conn net.Conn) *ClientPump {
	client := &ClientPump{make(chan string), make(chan string), conn}
	clients[conn] = client

	// Start the client service.
	go client.Send()
	go client.Read()

	return client
}

func (client *ClientPump) Close() {
	close(client.ToClient)
	client.conn.Close()
	client.Remove()
}

func (client *ClientPump) Equal(other *ClientPump) bool {
	return client.conn == other.conn
}

func (client *ClientPump) Remove() {
	log.Println("Removing client", client)
	delete(clients, client.conn)
}

func (client *ClientPump) Read() {
	reader := bufio.NewReader(client.conn)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Reading client", client, "ended with error:", err)
			break
		}
		text = strings.TrimRight(text, "\r\n")

		// Let QUIT quit right up front (for now).
		if text == "QUIT" {
			log.Println("Reading client", client, "ended due to client QUIT")
			break
		}

		log.Println("Reading client", client, "received >", text)
		client.ToServer <- text
	}
	client.Remove()
	log.Println("Reader for", client, "stopped")
}

func (client *ClientPump) Send() {
	writer := bufio.NewWriter(client.conn)
	for text := range client.ToClient {
		log.Println("Sending", text, "to", client)

		_, err := writer.WriteString(text)
		if err == nil {
			err = writer.WriteByte('\n')
			if err == nil {
				err = writer.Flush()
			}
		}
		if err != nil {
			log.Println("Sending text to", client, "failed:", err)
			client.Close()
			continue // quits when ToClient is closed
		}
	}
	log.Println("Sending to client", client, "quitting")
	client.conn.Close()
	client.Remove()
}
