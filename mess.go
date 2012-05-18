package main

import (
    "log"
    "net"
)

func main() {
    log.Println("Hello Server!")

    service := "localhost:9988"
    tcpAddr, error := net.ResolveTCPAddr("tcp", service)
    if error != nil {
        log.Println("Could not resolve address")
        return
    }

    netListen, error := net.Listen(tcpAddr.Network(), tcpAddr.String())
    if error != nil {
        log.Println("Could not listen on address:", error)
        return
    }

    defer netListen.Close()

    for {
        log.Println("Waiting for clients")
        connection, error := netListen.Accept()

        if error != nil {
            log.Println("Error accepting client: ", error)
            continue
        }

        go ClientHandler(connection)
    }
}
