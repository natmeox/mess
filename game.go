package main

import (
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	"log"
	"strings"
)

type Object struct {
	id         uuid.UUID
	name       string
	properties map[string]string
	location   *Object
	contents   map[*Object]bool

	clients map[*Client]bool
}

var objectForId map[uuid.UUID]*Object = make(map[uuid.UUID]*Object)

var playerHome *Object = NewObject()

func NewObject() (object *Object) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil
	}

	object = &Object{*id, "object", make(map[string]string), nil, make(map[*Object]bool), make(map[*Client]bool)}
	objectForId[*id] = object
	return
}

func NewPlayer(name string) (object *Object) {
	object = NewObject()
	object.name = name

	object.MoveTo(playerHome)

	log.Println("New character", name, "created")
	return
}

func GetObject(id uuid.UUID) (object *Object) {
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
		return // you know you connected
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
		return // you know you disconnected
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

func IdentifyNear(name string, context *Object) (target *Object) {
	if name == "" || name == "here" {
		return context.location
	}
	if name == "me" || name == "this" {
		return context
	}
	if strings.HasPrefix(context.location.name, name) {
		return context.location
	}
	for child, _ := range context.location.contents {
		if strings.HasPrefix(child.name, name) {
			return child
		}
	}
	return nil
}

func (target *Object) LookedAt(actor *Object) {
	text := fmt.Sprintf("%s looks at you.", actor.name)
	for client, _ := range target.clients {
		client.ToClient <- text
	}
}

func (actor *Object) LookAt(target *Object) {
	target.LookedAt(actor)

	text := fmt.Sprintf("The %s.", target.name)
	for client, _ := range actor.clients {
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

INPUT:
	for input := range client.ToServer {
		parts := strings.SplitN(input, " ", 2)

		command := strings.ToLower(parts[0])
		var rest string
		if len(parts) < 2 {
			rest = ""
		} else {
			rest = parts[1]
		}

		if command == "" {
		} else if strings.HasPrefix("whoami", command) {
			client.ToClient <- char.name
		} else if strings.HasPrefix("say", command) {
			char.location.Say(char, rest)
		} else if strings.HasPrefix("look", command) {
			// What does parts[1] refer to?
			target := IdentifyNear(rest, char)
			if target == nil {
				client.ToClient <- "I don't understand what you want to look at."
				continue INPUT
			}
			// What's its desc?
			char.LookAt(target)
		} else {
			client.ToClient <- "I don't understand what action you mean by '" + parts[0] + "'."
		}
	}

	log.Println("Character", char.name, "disconnected")
	if len(char.clients) <= 1 {
		char.location.Disconnected(char)
	}
	delete(char.clients, client)
}
