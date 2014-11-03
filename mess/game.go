package mess

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"
)

type ThingType int

const (
	RegularThing ThingType = iota
	PlaceThing
	PlayerThing
	ExitThing
	ProgramThing
)

type Thing struct {
	Id      int
	Type    ThingType
	Name    string
	Creator int
	Created time.Time

	Client   *ClientPump
	Parent   int
	Contents []int // ID numbers

	Table map[string]interface{}
}

func NewThing() (thing *Thing) {
	thing = &Thing{
		Contents: make([]int, 0),
		Table:    make(map[string]interface{}),
	}
	return
}

var World WorldStore
var Accounts AccountStore

func GameInit() {
	db, err := OpenDatabase()
	if err != nil {
		log.Println("Error connecting to database:", err)
		return
	}

	World = &ActiveWorld{
		Things: make(map[int]*Thing),
		Next:   db,
	}
	Accounts = db
}

func Identify(source *Thing, name string) *Thing {
	nameLower := strings.ToLower(name)
	if nameLower == "me" {
		return source
	}

	here := World.ThingForId(source.Parent)
	if nameLower == "here" {
		return here
	}

	for _, otherId := range here.Contents {
		otherThing := World.ThingForId(otherId)
		otherNameLower := strings.ToLower(otherThing.Name)
		if strings.HasPrefix(otherNameLower, nameLower) {
			return otherThing
		}
	}

	return nil
}

func GameLook(client *ClientPump, char *Thing, rest string) {
	if rest == "" {
		rest = "here"
	}
	target := Identify(char, rest)

	if target != nil {
		client.ToClient <- target.Name
		desc, ok := target.Table["description"].(string)
		if ok && desc != "" {
			client.ToClient <- desc
		} else {
			client.ToClient <- "You see nothing special."
		}
		return
	}

	client.ToClient <- fmt.Sprintf("Not sure what you meant by \"%s\".", rest)
}

func GameSay(client *ClientPump, char *Thing, rest string) {
	client.ToClient <- fmt.Sprintf("You say, \"%s\"", rest)

	text := fmt.Sprintf("%s says, \"%s\"", char.Name, rest)
	parent := World.ThingForId(char.Parent)
	for _, otherId := range parent.Contents {
		if otherId == char.Id {
			continue
		}
		otherChar := World.ThingForId(otherId)
		if otherChar.Client != nil {
			otherChar.Client.ToClient <- text
		}
	}
}

func GameClient(client *ClientPump, account *Account) {
	char := World.ThingForId(account.Character)
	if char.Client != nil {
		// TODO: kill the old one???
	}
	char.Client = client

	// We just arrived from the welcome screen, so "look" around.
	// TODO: motd?
	GameLook(client, char, "")

	for input := range client.ToServer {
		parts := strings.SplitN(input, " ", 2)
		command := strings.ToLower(parts[0])
		rest := ""
		if len(parts) > 1 {
			rest = parts[1]
		}
		log.Println("Unused portion of command:", rest)

	Command:
		switch command {
		case "quit":
			client.ToClient <- "Thanks for spending time with the mess today!"
			close(client.ToClient)
			break
		case "look":
			GameLook(client, char, rest)
		case "say":
			GameSay(client, char, rest)
		default:
			// Look up the environment for an exit with that command.
			var exit *Thing
			thisThing := char
		FindThing:
			for thisThing != nil {
				for _, thingId := range thisThing.Contents {
					thing := World.ThingForId(thingId)
					if thing.Type != ExitThing {
						log.Println("Found place contents", thing, "but it's not an Exit (",
							ExitThing, "), it's a", thing.Type, "; skipping")
						continue
					}

					if strings.ToLower(thing.Name) == command {
						log.Println("Found exit", thing, "has name", command, "!")
						exit = thing
						break FindThing
					}
					if aliases, ok := thing.Table["aliases"]; ok {
						if aliasesList, ok := aliases.([]interface{}); ok {
							for _, alias := range aliasesList {
								if aliasStr, ok := alias.(string); ok {
									if strings.ToLower(aliasStr) == command {
										log.Println("Found exit", thing, "has alias", command, "!")
										exit = thing
										break FindThing
									}
								}
							}
						}
					}
				}

				thisThing = World.ThingForId(thisThing.Parent)
			}
			if exit != nil {
				if targetId, ok := exit.Table["target"]; ok {
					// JSON numbers are float64s. :|
					if targetIdNum, ok := targetId.(float64); ok {
						target := World.ThingForId(int(targetIdNum))
						World.MoveThing(char, target)

						// We moved so let's have a new look shall we.
						GameLook(client, char, "")

						break Command
					} else {
						log.Println("Exit", exit, "has a target", targetId, "that's not an int but a", reflect.TypeOf(targetId))
					}
				} else {
					log.Println("Exit", exit, "doesn't have a target")
				}
			}

			// Didn't find such an exit.
			client.ToClient <- fmt.Sprintf("Oops, not sure what you mean by \"%s\".", command)
		}
	}
}
