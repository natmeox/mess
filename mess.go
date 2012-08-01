package main

import (
    "fmt"
    "log"
    "net"
)

func assert(value bool, message string) {
    if value {
        return
    }

    fmt.Printf(message)
    panic(1)
}

func main() {
    log.Println("Hello Server!")

    NewWelcomeScreen()
    go WelcomeScreen.provideScreens()

    serve()
}

func serve() {
    InitDatabase()
    defer CloseDatabase()

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

        client := NewClient(connection)
        go WelcomeScreen.Welcome(client)
    }
}
