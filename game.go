package main

import (
    "strings"
)

func Game(client *Client, account *Account) {
    // yays a fun
    // TODO: attach the client to the account's object.

    INPUT: for input := range client.ToServer {
        parts := strings.SplitN(input, " ", 2)
        command := strings.ToLower(parts[0])

        if strings.HasPrefix("derp", command) {
            client.ToClient <- "~derp~"
            continue INPUT
        } else {
            client.ToClient <- "I didn't understand what you meant by '" + command + "'."
        }
    }
}
