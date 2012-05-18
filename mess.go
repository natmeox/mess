package main

import (
    "fmt"
    "net"
    "container/list"
    "bytes"
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
        Log(error)
        return false
    }
    Log("Read ", bytesread, " bytes")
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
            Log("Removing client ", c.Name)
            c.ClientList.Remove(entry)
        }
    }
}

func Log(v ...interface{}) {
    fmt.PrintLn(v...)
}

func IOHandler(Incoming <-chan string, clientList *list.List) {
    for {
        Log("IOHandler: Waiting for input")
        input := <-Incoming
        Log("IOHandler: Handling ", input)
        for e := clientList.Front(); e != nil; e = e.Next() {
            client := e.Value.(Client)
            client.Incoming <- input
        }
    }
}

func ClientReader(client *Client) {
    buffer := make([]byte, 2048)

    for client.Read(buffer) {
        if bytes.Equal(buffer, []byte("QUIT")) {
            client.Close()
            break
        }

        Log("ClientReader received ", client.Name, "> ", string(buffer))
        send := client.Name + "> " + string(buffer)
        client.Outgoing <- send
        for i := 0; i < 2048, i++ {
            buffer[i] = 0x00
        }
    }

    client.Outgoing <- client.Name + " has left chat"
    Log("ClientReader stopped for ", client.Name)
}

func ClientSender(client *Client) {
    for {
        select {
            case buffer := <-client.Incoming:
                Log("ClientSender sending ", string(buffer), " to ", client.Name);
                count := 0
                for i := 0; i < len(buffer); i++ {
                    if buffer[i] == 0x00 {
                        break
                    }
                    count++
                }

                Log("Send size: ", count)
                client.Conn.Write([]byte(buffer)[0:count])

            case <-client.Quit:
                Log("Client ", client.Name, " quitting")
                client.Conn.Close()
                break  // ??
        }
    }
}

func ClientHandler(conn net.Conn, ch chan string, clientList *list.List) {
    buffer := make([]byte, 1024)
    bytesread, error := conn.Read(buffer)
    if error != nil {
        Log("Client connection error: ", error)
        return
    }

    name := string(buffer[0:bytesread])
    newClient := &Client{name, make(chan string), ch, conn, make(chan bool), clientList}

    go ClientSender(newClient)
    go ClientReader(newClient)
    clientList.PushBack(*newClient)
    ch <- string(name + " has joined the chat")
}

func main() {
    Log("Hello Server!")

    clientList := list.New()
    in := make(chan string)
    go IOHandler(in, clientList)

    service := ":9988"
    tcpAddr, error := net.ResolveTCPAddr("tcp", service)
    if error != nil {
        Log("Could not resolve address")
        return
    }

    netListen, error := net.Listen(tcpAddr.Network(), tcpAddr.String())
    if error != nil {
        Log("Could not listen on address: ", error)
        return
    }

    defer netListen.Close()

    for {
        Log("Waiting for clients")
        connection, error := netListen.Accept()

        if error != nil {
            Log("Error accepting client: ", error)
            continue
        }

        go ClientHandler(connection, in, clientList)
    }
}
