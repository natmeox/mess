package mess

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type Room struct {
	Id int
	Name string
	Description string
	Creator int
	Created time.Time

	Characters []*Character
}

var RoomForId struct {
	sync.Mutex
	Rooms map[int]*Room
}

type Character struct {
	Id int
	Name string
	Description string

	Client *ClientPump
	Room *Room
}

var CharacterForId struct {
	sync.Mutex
	Characters map[int]*Character
}

func GameInit() {
	CharacterForId.Characters = make(map[int]*Character)
	RoomForId.Rooms = make(map[int]*Room)
}

func GetRoomForId(id int) (room *Room) {
	RoomForId.Lock()
	defer RoomForId.Unlock()

	room, ok := RoomForId.Rooms[id]
	if ok {
		return
	}

	room = &Room{}
	room.Id = id
	row := Db.QueryRow("SELECT name, description, creator, created FROM room WHERE id = $1",
		id)
	err := row.Scan(&room.Name, &room.Description, &room.Creator, &room.Created)
	if err != nil {
		log.Println("Error finding room", id, ":", err.Error())
		return nil
	}

	room.Characters = make([]*Character, 0, 2)

	RoomForId.Rooms[id] = room
	return
}

func GetCharacterForId(id int) (char *Character) {
	CharacterForId.Lock()
	defer CharacterForId.Unlock()

	char, ok := CharacterForId.Characters[id]
	if ok {
		return
	}

	char = &Character{}
	char.Id = id
	row := Db.QueryRow("SELECT name, description FROM character WHERE id = $1",
		id)
	err := row.Scan(&char.Name, &char.Description)
	if err != nil {
		log.Println("Error finding character", id, ":", err.Error())
		return nil
	}

	CharacterForId.Characters[id] = char
	return
}

func (char *Character) Move(room *Room) {
	oldroom := char.Room

	if oldroom != nil {
		if oldroom.Id == room.Id {
			log.Println("Supposed to move", char, "from", room, "to itself, skipping")
			return
		}

		oldroom.CharLeft(char)

		// Remove char from the room's Characters.
		for i, c := range oldroom.Characters {
			if c != char {
				continue
			}

			copy(oldroom.Characters[i:], oldroom.Characters[i+1:])
			oldroom.Characters = oldroom.Characters[:len(oldroom.Characters)-1]
			log.Println("Removed", char, "from room", oldroom, ", remaining characters:", oldroom.Characters)
			break
		}
	}

	char.Room = room

	if room != nil {
		room.Characters = append(room.Characters, char)
		room.CharArrived(char)
	}
}

func (room *Room) CharLeft(char *Character) {
	log.Println("Telling room", room, room.Name, "that character", char, char.Name, "is leaving.")
	text := fmt.Sprintf("%s left.", char.Name)
	for _, otherChar:= range room.Characters {
		otherChar.Client.ToClient <- text
	}
}

func (room *Room) CharArrived(char *Character) {
	// TODO: tell everyone in the room that they have ARRIVED
	text := fmt.Sprintf("%s arrived.", char.Name)
	for _, otherChar := range room.Characters {
		if otherChar == char {
			continue
		}
		otherChar.Client.ToClient <- text
	}

	char.Client.ToClient <- room.Name
	char.Client.ToClient <- room.Description
}

func GameLook(client *ClientPump, char *Character, rest string) {
	if rest == "" {
		client.ToClient <- char.Room.Name
		client.ToClient <- char.Room.Description
		return
	}

	restLower := strings.ToLower(rest)
	for _, otherChar := range char.Room.Characters {
		nameLower := strings.ToLower(otherChar.Name)
		if strings.HasPrefix(nameLower, restLower) {
			client.ToClient <- otherChar.Name
			client.ToClient <- otherChar.Description
			return
		}
	}

	client.ToClient <- fmt.Sprintf("Not sure what you meant by \"%s\".", rest)
}

func GameSay(client *ClientPump, char *Character, rest string) {
	client.ToClient <- fmt.Sprintf("You say, \"%s\"", rest)

	text := fmt.Sprintf("%s says, \"%s\"", char.Name, rest)
	for _, otherChar := range char.Room.Characters {
		if otherChar == char {
			continue
		}
		otherChar.Client.ToClient <- text
	}
}

func GameClient(client *ClientPump, account *Account) {
	// Is the 
	char := GetCharacterForId(account.Character)
	if char.Client != nil {
		// TODO: kill the old one???
	}
	char.Client = client

	// Everyone starts in room #1.
	room := GetRoomForId(1)
	char.Move(room)

	for input := range client.ToServer {
		parts := strings.SplitN(input, " ", 2)
		command := strings.ToLower(parts[0])
		rest := ""
		if len(parts) > 1 {
			rest = parts[1]
		}

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
				client.ToClient <- fmt.Sprintf("Oops, not sure what you mean by \"%s\".", command)
		}
	}

	// The channel ended, so the character is gone.
	char.Move(nil)
}
