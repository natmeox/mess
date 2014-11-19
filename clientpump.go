package mess

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
)

type ClientPump struct {
	ToServer chan string
	writer *bufio.Writer
	conn net.Conn
}

var clientLock sync.Mutex
var clients map[net.Conn]*ClientPump = make(map[net.Conn]*ClientPump)

func NewClientPump(conn net.Conn) *ClientPump {
	clientLock.Lock()
	defer clientLock.Unlock()

	client := &ClientPump{make(chan string), bufio.NewWriter(conn), conn}
	clients[conn] = client

	// Start the client service.
	go client.Read()

	return client
}

func (client *ClientPump) Close() {
	log.Println("Possibly closing ClientPump", client)
	clientLock.Lock()

	if _, ok := clients[client.conn]; !ok {
		// We already closed.
		log.Println("Avoided duplicate Close() of ClientPump", client)
		clientLock.Unlock()
		return
	}

	// Remove us so we can finish the connection in peace.
	client.Remove()
	clientLock.Unlock()

	log.Println("All flushed. Commencing close of the ClientPump", client)
	client.conn.Close()
	close(client.ToServer)
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

		log.Println("Reading client", client, "received >", text)
		client.ToServer <- text
	}
	log.Println("Reader for", client, "stopped")
	client.Close()
}

func (client *ClientPump) Send(text string) {
	log.Println("Sending", text, "to", client)

	_, err := client.writer.WriteString(text)
	if err == nil {
		err = client.writer.WriteByte('\n')
		if err == nil {
			err = client.writer.Flush()
		}
	}
	if err != nil {
		log.Println("Sending text to", client, "failed:", err)
		client.Close()
		return
	}
}
