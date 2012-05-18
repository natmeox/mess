package main

import (
    "bytes"
    "container/list"
    "log"
    "net"
    "strings"
)

type Client struct {
    ToClient chan string
    ToServer chan string
    Conn net.Conn
    Quit chan bool
}

var ClientList list.List

func (c *Client) Name() string {
    return c.Conn.RemoteAddr().String()
}

func (c *Client) Read(buffer []byte) bool {
    bytesread, error := c.Conn.Read(buffer)
    if error != nil {
        c.Close()
        log.Println("Error reading from client", error)
        return false
    }
    log.Println("Read", bytesread, "bytes")
    return true
}

func (c *Client) Close() {
    c.Quit <- true
    c.Conn.Close()
    c.Remove()
}

func (c *Client) Equal(other *Client) bool {
    return c.Conn == other.Conn
}

func (c *Client) Remove() {
    for entry := ClientList.Front(); entry != nil; entry = entry.Next() {
        client := entry.Value.(Client)
        if c.Equal(&client) {
            log.Println("Removing client ", c.Name())
            ClientList.Remove(entry)
        }
    }
}

func ClientReader(client *Client) {
    buffer := make([]byte, 2048)
    for client.Read(buffer) {
        // Find just the text content of the buffer.
        count := bytes.IndexByte(buffer, byte(0x00))
        if count == -1 {
            count = len(buffer)
        }
        text := string(buffer[:count])
        text = strings.TrimRight(text, "\r\n")

        // Let QUIT quit right up front (for now).
        if text == "QUIT" {
            client.Close()
            break
        }

        log.Println("ClientReader received", client.Name(), ">", text)
        client.ToServer <- text

        // Then reset the buffer.
        for i := 0; i < 2048; i++ {
            buffer[i] = 0x00
        }
    }

    log.Println("ClientReader stopped for", client.Name())
}

func ClientSender(client *Client) {
    for {
        select {
            case text := <-client.ToClient:
                log.Println("ClientSender sending", text, "to", client.Name());
                client.Conn.Write([]byte(text))
                client.Conn.Write([]byte("\n"))

            case <-client.Quit:
                log.Println("Client", client.Name(), "quitting")
                client.Conn.Close()
                break  // ??
        }
    }
}

func ClientHandler(conn net.Conn) {
    newClient := &Client{make(chan string), make(chan string), conn, make(chan bool)}
    ClientList.PushBack(newClient)

    go ClientSender(newClient)
    go ClientReader(newClient)
    go Welcome(newClient)
}
