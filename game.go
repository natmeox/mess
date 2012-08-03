package main

import (
	"database/sql"
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	"log"
	"strings"
)

type Object struct {
	id         uuid.UUID
	name       string
	properties map[string]string
	location   uuid.UUID
	contents   map[uuid.UUID]bool

	clients map[*Client]bool
}

var objectForId map[uuid.UUID]*Object = make(map[uuid.UUID]*Object)

func NewObject(name string) (object *Object) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil
	}

	object = &Object{*id, name, make(map[string]string), uuid.UUID([16]byte{}), make(map[uuid.UUID]bool), make(map[*Client]bool)}
	objectForId[*id] = object
	return
}

func NewPlayer(name string) (object *Object) {
	object = NewObject(name)

	log.Println("New character", name, "created")
	return
}

func GetObject(id uuid.UUID) (object *Object) {
	object = objectForId[id]
	if object != nil {
		log.Println("Found object", id, "already loaded, returning", object)
		return object
	}

	object, err := LoadObject(id)
	if err != nil {
		log.Println("Could not load object", id, err.Error())
		panic(1)
	}
	if object == nil {
		log.Println("ZOMG COULD NOT LOAD OBJ", id, "BUT NO ERR")
		panic(1)
	}

	objectForId[id] = object
	return object
}

func LoadObject(id uuid.UUID) (object *Object, err error) {
	log.Println("Loading object", id, "from database")

	rows, err := db.Query("select name, location from object where id = ? limit 1", id[0:16])
	if err != nil {
		return nil, err
	}

	var name string
	var location [16]byte
	if rows.Next() {
		rows.Scan(&name, &location)
		object = &Object{id, name, make(map[string]string), uuid.UUID(location), make(map[uuid.UUID]bool), make(map[*Client]bool)}
	} else {
		return nil, sql.ErrNoRows
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	log.Println("Yay, created", id, "object", object)

	rows, err = db.Query("select id from object where location = ?", id[0:16])
	if err != nil {
		return nil, err
	}

	var childId [16]byte
	for rows.Next() {
		rows.Scan(&childId)
		object.contents[uuid.UUID(childId)] = true
	}
	log.Println("Yay yay, filled in", id, "contents too")

	return
}

func (actor *Object) MoveTo(target *Object) {
	if actor.location == target.id {
		return
	}

	oldLocation := GetObject(actor.location)
	oldLocation.Departed(actor, oldLocation)
	delete(oldLocation.contents, actor.id)

	actor.location = target.id

	target.contents[actor.id] = true
	target.Arrived(actor, target)
}

func (object *Object) Connected(actor *Object) {
	if object == actor {
		return // you know you connected
	}

	if object.contents[actor.id] {
		for childId, _ := range object.contents {
			child := GetObject(childId)
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

	if object.contents[actor.id] {
		for childId, _ := range object.contents {
			child := GetObject(childId)
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
		for childId, _ := range target.contents {
			child := GetObject(childId)
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
		for childId, _ := range target.contents {
			child := GetObject(childId)
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
	if object.contents[actor.id] {
		for childId, _ := range object.contents {
			child := GetObject(childId)
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
	if name == "me" || name == "this" {
		return context
	}
	contextLocation := GetObject(context.location)
	if name == "" || name == "here" {
		return contextLocation
	}
	if strings.HasPrefix(contextLocation.name, name) {
		return contextLocation
	}
	for childId, _ := range contextLocation.contents {
		child := GetObject(childId)
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
		loc := GetObject(char.location)
		if loc == nil {
			log.Println("ZOMG OBJ FOR", char.location, "IS NULZL")
			panic(1)
		}
		loc.Connected(char)
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
			GetObject(char.location).Say(char, rest)
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
		GetObject(char.location).Disconnected(char)
	}
	delete(char.clients, client)
}
