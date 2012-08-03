package main

import (
	"flag"
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
	var databaseFilename string
	var networkPort int

	flag.IntVar(&networkPort, "port", 9988, "port to listen for text connections on")
	flag.StringVar(&databaseFilename, "database", "./database", "path to the database file")

	err := InitDatabase(databaseFilename)
	if err != nil {
		log.Println("Error initializing database:", err.Error())
		panic(1)
	}
	defer CloseDatabase()

	service := fmt.Sprintf("localhost:%d", networkPort)
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
