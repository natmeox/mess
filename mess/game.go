package mess

import (
	"fmt"
	"log"
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

func (thing *Thing) ExitMatches(command string) bool {
	if thing.Type != ExitThing {
		return false
	}

	if strings.ToLower(thing.Name) == command {
		log.Println("Found exit", thing, "has name", command, "!")
		return true
	}

	// Check our aliases.
	if aliases, ok := thing.Table["aliases"]; ok {
		if aliasesList, ok := aliases.([]interface{}); ok {
			for _, alias := range aliasesList {
				if aliasStr, ok := alias.(string); ok {
					if strings.ToLower(aliasStr) == command {
						log.Println("Found exit", thing, "has alias", command, "!")
						return true
					}
				}
			}
		} else {
			log.Println("Found exit", thing, "but could not cast its aliases list to a []string; skipping")
		}
	} else {
		log.Println("Found exit", thing, "but it has no aliases; skipping")
	}

	return false
}

func (thing *Thing) ExitTarget() (target *Thing) {
	if targetId, ok := thing.Table["target"]; ok {
		// JSON numbers are float64s. :|
		if targetIdNum, ok := targetId.(float64); ok {
			targetIdInt := int(targetIdNum)
			target = World.ThingForId(targetIdInt)
		}
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
					if thing.ExitMatches(command) {
						exit = thing
						break FindThing
					}
				}

				// No exits on thisThing matched. Try up the environment.
				thisThing = World.ThingForId(thisThing.Parent)
			}
			if exit != nil {
				target := exit.ExitTarget()
				// TODO: run a program if target.Type == ProgramThing
				// TODO: move to target.Parent if PlayerThing or RegularThing
				if target != nil && target.Type == PlaceThing {
					World.MoveThing(char, target)

					// We moved so let's have a new look shall we.
					GameLook(client, char, "")

					break Command
				}
			}

			// Didn't find such an exit.
			client.ToClient <- fmt.Sprintf("Oops, not sure what you mean by \"%s\".", command)
		}
	}
}
