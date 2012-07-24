package main

import (
    "strings"
    uuid "github.com/nu7hatch/gouuid"
)

type Object struct {
    id string
    name string
    properties map[string] string
    clients map[*Client] bool
    parent *Object
}

var objectForId map[string] *Object = make(map[string] *Object)

func NewObject() (object *Object) {
    idId, err := uuid.NewV4()
    if err != nil {
        return nil
    }
    id := idId.String()

    object = &Object{id, "object", make(map[string] string), make(map[*Client] bool), nil}
    objectForId[id] = object
    return
}

func NewPlayer(name string) (obj *Object) {
    obj = NewObject()
    obj.name = name
    return
}

func GetObject(id string) (object *Object) {
    object = objectForId[id]
    if object == nil {
        // TODO: load this object from the store
    }
    return
}

func Game(client *Client, account *Account) {
    // yays a fun
    // TODO: attach the client to the account's object.

    // Connect the client.
    player := GetObject(account.objectId)
    player.clients[client] = true
    defer delete(player.clients, client)

    INPUT: for input := range client.ToServer {
        parts := strings.SplitN(input, " ", 2)
        command := strings.ToLower(parts[0])

        if strings.HasPrefix("derp", command) {
            client.ToClient <- "~derp~"
            continue INPUT
        } else if strings.HasPrefix("whoami", command) {
            client.ToClient <- player.name
            continue INPUT
        } else {
            client.ToClient <- "I didn't understand what you meant by '" + command + "'."
        }
    }
}
