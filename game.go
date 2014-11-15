package mess

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type ThingId int
type ThingIdList []ThingId
type ThingType int

const (
	RegularThing ThingType = iota
	PlaceThing
	PlayerThing
	ActionThing
	ProgramThing
)

func ThingTypeForName(name string) ThingType {
	switch name {
	case "place":
		return PlaceThing
	case "player":
		return PlayerThing
	case "action":
		return ActionThing
	case "program":
		return ProgramThing
	}
	return RegularThing
}

func (tt ThingType) String() string {
	switch tt {
	case PlaceThing:
		return "place"
	case PlayerThing:
		return "player"
	case ActionThing:
		return "action"
	case ProgramThing:
		return "program"
	}
	return "thing"
}

func (tl *ThingIdList) Things() []*Thing {
	things := make([]*Thing, len(*tl))
	for i := 0; i < len(*tl); i++ {
		things[i] = World.ThingForId((*tl)[i])
	}
	return things
}

type Thing struct {
	Id      ThingId
	Type    ThingType
	Name    string
	Parent  ThingId
	Creator ThingId
	Created time.Time

	Owner     ThingId
	AdminList ThingIdList
	AllowList ThingIdList
	DenyList  ThingIdList

	Contents ThingIdList
	Table    map[string]interface{}
	Program  *ThingProgram

	Client *ClientPump
}

func NewThing() (thing *Thing) {
	thing = &Thing{
		Contents: make([]ThingId, 0),
		Table:    make(map[string]interface{}),
	}
	return
}

func (thing *Thing) GetURL() string {
	return fmt.Sprintf("/%s/%d/", thing.Type.String(), thing.Id)
}

func (thing *Thing) GetParent() *Thing {
	return World.ThingForId(thing.Parent)
}

func (thing *Thing) GetOwner() *Thing {
	return World.ThingForId(thing.Owner)
}

func (thing *Thing) OwnedById(playerId ThingId) bool {
	if thing.Type == PlayerThing {
		return thing.Id == playerId
	}
	return thing.Owner == playerId
}

func (thing *Thing) EditableById(playerId ThingId) bool {
	switch {
	case thing.Type == PlayerThing:
		return thing.Id == playerId
	case thing.Owner == playerId:
		return true
	}

	for _, adminId := range thing.AdminList {
		if adminId == playerId {
			return true
		}
	}
	return false
}

func (thing *Thing) DeniedById(playerId ThingId) bool {
	for _, deniedId := range thing.DenyList {
		if deniedId == playerId {
			return true
		}
	}
	return false
}

func (thing *Thing) GetContents() (contents []*Thing) {
	for _, thingId := range thing.Contents {
		content := World.ThingForId(thingId)
		if content.Type != ActionThing {
			contents = append(contents, content)
		}
	}
	return
}

func (thing *Thing) GetActions() (actions []*Thing) {
	for _, thingId := range thing.Contents {
		action := World.ThingForId(thingId)
		if action.Type == ActionThing {
			actions = append(actions, action)
		}
	}
	return
}

func (thing *Thing) ActionMatches(command string) bool {
	if thing.Type != ActionThing {
		return false
	}

	if strings.ToLower(thing.Name) == command {
		log.Println("Found action", thing, "has name", command, "!")
		return true
	}

	// Check our aliases.
	if aliases, ok := thing.Table["aliases"]; ok {
		if aliasesList, ok := aliases.([]interface{}); ok {
			for _, alias := range aliasesList {
				if aliasStr, ok := alias.(string); ok {
					if strings.ToLower(aliasStr) == command {
						log.Println("Found action", thing, "has alias", command, "!")
						return true
					}
				}
			}
		} else {
			log.Println("Found action", thing, "but could not cast its aliases list to a []string; skipping")
		}
	} else {
		log.Println("Found action", thing, "but it has no aliases; skipping")
	}

	return false
}

func (thing *Thing) ActionTarget() (target *Thing) {
	if targetId, ok := thing.Table["target"]; ok {
		// JSON numbers are float64s. :|
		// TODO: shouldn't these be links? how can we have thing references in our json data
		if targetIdNum, ok := targetId.(float64); ok {
			targetIdInt := ThingId(targetIdNum)
			target = World.ThingForId(targetIdInt)
		}
	}
	return
}

func (thing *Thing) TryToCall(name string, env map[string]interface{}, args ...interface{}) {
	prog := thing.Program
	if prog == nil {
		// No program to run, a-OK.
		return
	}

	err := prog.TryToCall(name, env, args...)
	if err == nil {
		// Success!
		return
	}

	// Notify the thing's owner of the error.
	owner := thing
	if thing.Type != PlayerThing {
		owner = World.ThingForId(thing.Owner)
	}
	ownClient := owner.Client
	if ownClient != nil {
		ownClient.ToClient <- fmt.Sprintf("Error with your program '%s': %s",
			thing.Name, err.Error())
	}
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
		Things: make(map[ThingId]*Thing),
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

	if target == nil {
		client.ToClient <- fmt.Sprintf("Not sure what you meant by \"%s\".", rest)
		return
	}

	client.ToClient <- target.Name
	desc, ok := target.Table["description"].(string)
	if ok && desc != "" {
		client.ToClient <- desc
	} else {
		client.ToClient <- "You see nothing special."
	}

	target.TryToCall("Looked", map[string]interface{}{
		"me":     char.Id,
		"here":   char.Parent,
		"target": target.Id,
	}, char)
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
			// Look up the environment for an action with that command.
			var action *Thing
			thisThing := char
		FindThing:
			for thisThing != nil {
				for _, thingId := range thisThing.Contents {
					thing := World.ThingForId(thingId)
					if thing.ActionMatches(command) {
						action = thing
						break FindThing
					}
				}

				// No actions on thisThing matched. Try up the environment.
				thisThing = World.ThingForId(thisThing.Parent)
			}
			if action == nil {
				log.Println("Found no action", command, ", womp womp")
				client.ToClient <- fmt.Sprintf("Oops, not sure what you mean by \"%s\".", command)
				break Command
			}

			// Can I use this action?
			if action.DeniedById(char.Id) {
				// TODO: action failure messages? once we have them? maybe?
				client.ToClient <- fmt.Sprintf("You can't use that.")
				break Command
			}

			target := action.ActionTarget()
			if target == nil {
				client.ToClient <- fmt.Sprintf("Nothing happens.")
				break Command
			}
			log.Println("Action", command, "has target", target)

			// Can we use that target?
			if target.DeniedById(char.Id) {
				// TODO: action failure messages? once we have them? maybe?
				client.ToClient <- fmt.Sprintf("You can't use that.")
				break Command
			}

			// TODO: move to target.Parent if PlayerThing or RegularThing?
			switch target.Type {
			case PlaceThing:
				log.Println("Target is a place, moving player there")
				World.MoveThing(char, target)
				GameLook(client, char, "")
			case ProgramThing:
				log.Println("Target is a program object")
				target.TryToCall("Run", map[string]interface{}{
					"me":      char.Id,
					"here":    char.Parent,
					"target":  action.Id, // the "trigger"
					"command": parts[0],  // un-lowered
				}, rest)
			default: // player, action, regular thing
				client.ToClient <- fmt.Sprintf("Nothing happens.")
			}
		}
	}
}
