package main

func Welcome(client *Client) {
    welcomeScreen := "~WELCOME~"
    client.ToClient <- welcomeScreen

    for command := range client.ToServer {
        if command == "derp" {
            client.ToClient <- "DERP INDEED"
        } else if command == "herp" {
            client.ToClient <- "~herp~"
        }
    }
}
