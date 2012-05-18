package main

import (
    "bytes"
    "container/list"
    "log"
    "net"
    "strings"
)

type Client struct {
    Name string
    Incoming chan string
    Outgoing chan string
    Conn net.Conn
    Quit chan bool
    ClientList *list.List
}

func (c *Client) Read(buffer []byte) bool {
    bytesread, error := c.Conn.Read(buffer)
    if error != nil {
        c.Close()
        log.Println(error)
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
    if bytes.Equal([]byte(c.Name), []byte(other.Name)) {
        if c.Conn == other.Conn {
            return true
        }
    }
    return false
}

func (c *Client) Remove() {
    for entry := c.ClientList.Front(); entry != nil; entry = entry.Next() {
        client := entry.Value.(Client)
        if c.Equal(&client) {
            log.Println("Removing client ", c.Name)
            c.ClientList.Remove(entry)
        }
    }
}

func IOHandler(Incoming <-chan string, clientList *list.List) {
    for {
        log.Println("IOHandler: Waiting for input")
        input := <-Incoming
        log.Println("IOHandler: Handling", input)
        for e := clientList.Front(); e != nil; e = e.Next() {
            client := e.Value.(Client)
            client.Incoming <- input
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

        if text == "QUIT" {
            client.Close()
            break
        }

        log.Println("ClientReader received", client.Name, ">", text)
        send := client.Name + "> " + text
        client.Outgoing <- send

        // Then reset the buffer.
        for i := 0; i < 2048; i++ {
            buffer[i] = 0x00
        }
    }

    client.Outgoing <- client.Name + " has left chat"
    log.Println("ClientReader stopped for", client.Name)
}

func ClientSender(client *Client) {
    for {
        select {
            case buffer := <-client.Incoming:
                log.Println("ClientSender sending", buffer, "to", client.Name);
                client.Conn.Write([]byte(buffer))
                client.Conn.Write([]byte("\n"))

            case <-client.Quit:
                log.Println("Client ", client.Name, " quitting")
                client.Conn.Close()
                break  // ??
        }
    }
}

func ClientHandler(conn net.Conn, ch chan string, clientList *list.List) {
    buffer := make([]byte, 1024)
    bytesread, error := conn.Read(buffer)
    if error != nil {
        log.Println("Client connection error: ", error)
        return
    }

    name := string(buffer[0:bytesread])
    name = strings.TrimRight(name, "\r\n")
    newClient := &Client{name, make(chan string), ch, conn, make(chan bool), clientList}

    go ClientSender(newClient)
    go ClientReader(newClient)
    clientList.PushBack(*newClient)
    ch <- string(name + " has joined the chat")
}
