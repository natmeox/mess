package main

import (
	"bufio"
	"log"
	"net"
	"strings"
)

type Client struct {
	ToClient chan string
	ToServer chan string

	conn net.Conn
}

var clients map[net.Conn]*Client = make(map[net.Conn]*Client)

func NewClient(conn net.Conn) *Client {
	client := &Client{make(chan string), make(chan string), conn}
	clients[conn] = client

	// Start the client service.
	go client.Send()
	go client.Read()

	return client
}

func (client *Client) Close() {
	close(client.ToClient)
	client.conn.Close()
	client.Remove()
}

func (client *Client) Equal(other *Client) bool {
	return client.conn == other.conn
}

func (client *Client) Remove() {
	log.Println("Removing client", client)
	delete(clients, client.conn)
}

func (client *Client) Read() {
	reader := bufio.NewReader(client.conn)
	for {
		text, error := reader.ReadString('\n')
		if error != nil {
			log.Println("Reading client", client, "ended with error:", error)
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

	client.Close()
	log.Println("Reader for", client, "stopped")
}

func (client *Client) Send() {
	writer := bufio.NewWriter(client.conn)
	for text := range client.ToClient {
		log.Println("Sending", text, "to", client)

		_, error := writer.WriteString(text)
		if error == nil {
			error = writer.WriteByte('\n')
			if error == nil {
				error = writer.Flush()
			}
		}
		if error != nil {
			log.Println("Sending text to", client, "failed:", error)
			client.Close()
			continue // quits when ToClient is closed
		}
	}
	log.Println("Sending to client", client, "quitting")
}
