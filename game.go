package main

import (
    "fmt"
    "log"
    "strings"
    uuid "github.com/nu7hatch/gouuid"
)

type Object struct {
    id string
    name string
    properties map[string] string
    location *Object
    contents map[*Object] bool

    clients map[*Client] bool
}

var objectForId map[string] *Object = make(map[string] *Object)

var playerHome *Object = NewObject()

func NewObject() (object *Object) {
    idId, err := uuid.NewV4()
    if err != nil {
        return nil
    }
    id := idId.String()

    object = &Object{id, "object", make(map[string] string), nil, make(map[*Object] bool), make(map[*Client] bool)}
    objectForId[id] = object
    return
}

func NewPlayer(name string) (object *Object) {
    object = NewObject()
    object.name = name

    object.MoveTo(playerHome)

    log.Println("New character", name, "created")
    return
}

func GetObject(id string) (object *Object) {
    object = objectForId[id]
    if object == nil {
        // TODO: load this object from the store
    }
    return
}

func (actor *Object) MoveTo(target *Object) {
    // TODO: erm... this should happen pretty atomically?
    if actor.location != nil {
        actor.location.Departed(actor, actor.location)

        delete(actor.location.contents, actor)
    }

    actor.location = target

    target.contents[actor] = true
    target.Arrived(actor, target)
}

func (object *Object) Connected(actor *Object) {
    if object == actor {
        return  // you know you connected
    }

    if object.contents[actor] {
        for child, _ := range object.contents {
            child.Connected(actor)
        }
    }

    text := fmt.Sprintf("%s wakes up.", actor.name)
    for client, _ := range object.clients {
        client.ToClient <- text
    }
}

func (object *Object) Disconnected(actor *Object) {
    if object == actor {
        return  // you know you disconnected
    }

    if object.contents[actor] {
        for child, _ := range object.contents {
            child.Disconnected(actor)
        }
    }

    text := fmt.Sprintf("%s falls asleep.", actor.name)
    for client, _ := range object.clients {
        client.ToClient <- text
    }
}

func (object *Object) Departed(actor *Object, target *Object) {
    if target == object {
        for child, _ := range target.contents {
            child.Departed(actor, target)
        }
    }

    if actor == object {
        return
    }

    text := fmt.Sprintf("%s has left.", actor.name)
    for client, _ := range object.clients {
        client.ToClient <- text
    }
}

func (object *Object) Arrived(actor *Object, target *Object) {
    if target == object {
        for child, _ := range target.contents {
            child.Arrived(actor, target)
        }
    }

    if actor == object {
        return
    }

    // Notify all the other people there.
    text := fmt.Sprintf("%s has arrived.", actor.name)
    for client, _ := range object.clients {
        client.ToClient <- text
    }
}

func (object *Object) Say(actor *Object, message string) {
    if object.contents[actor] {
        for child, _ := range object.contents {
            child.Say(actor, message)
        }
    }

    var text string
    if object == actor {
        text = fmt.Sprintf("You say, \"%s\"", message)
    } else {
        text = fmt.Sprintf("%s says, \"%s\"", actor.name, message)
    }

    for client, _ := range object.clients {
        client.ToClient <- text
    }
}

func (object *Object) SendClients(text string) {
    for client, _ := range object.clients {
        client.ToClient <- text
    }
}

func Game(client *Client, account *Account) {
    char := GetObject(account.objectId)

    char.clients[client] = true
    if len(char.clients) <= 1 {
        char.location.Connected(char)
    }
    log.Println("Character", char.name, "connected")

    for input := range client.ToServer {
        parts := strings.SplitN(input, " ", 2)
        command := strings.ToLower(parts[0])

        if command == "" {
        } else if strings.HasPrefix("derp", command) {
            client.ToClient <- "~derp~"
        } else if strings.HasPrefix("whoami", command) {
            client.ToClient <- char.name
        } else if strings.HasPrefix("say", command) {
            char.location.Say(char, parts[1])
        } else {
            client.ToClient <- "I didn't understand what you meant by '" + command + "'."
        }
    }

    log.Println("Character", char.name, "disconnected")
    if len(char.clients) <= 1 {
        char.location.Disconnected(char)
    }
    delete(char.clients, client)
}
