package mess

import (
	"database/sql"
	"encoding/json"
	"github.com/jmoiron/sqlx/types"
	"log"
	"sync"
)

type WorldStore interface {
	ThingForId(id int) *Thing
	CreateThing(name string, creator *Thing, parent *Thing) (thing *Thing)
	MoveThing(thing *Thing, target *Thing) (ok bool)
	SaveThing(thing *Thing) (ok bool)
}

type DatabaseWorld struct {
	db *sql.DB
}

func (w *DatabaseWorld) ThingForId(id int) (thing *Thing) {
	if id == 0 {
		return nil
	}

	thing = NewThing()
	thing.Id = id

	row := w.db.QueryRow("SELECT type, name, creator, created, parent, tabledata FROM thing WHERE id = $1",
		id)
	var creator sql.NullInt64
	var parent sql.NullInt64
	var tabledata types.JsonText
	err := row.Scan(&thing.Type, &thing.Name, &creator, &thing.Created, &parent, &tabledata)
	if err != nil {
		log.Println("Error finding thing", id, ":", err.Error())
		return nil
	}
	if creator.Valid {
		thing.Creator = int(creator.Int64)
	}
	if parent.Valid {
		thing.Parent = int(parent.Int64)
	}
	err = tabledata.Unmarshal(&thing.Table)
	if err != nil {
		log.Println("Error finding table data for thing", id, ":", err.Error())
		return nil
	}

	// Find thing's contents.
	contentRows, err := w.db.Query("SELECT id FROM thing WHERE parent = $1", id)
	if err != nil {
		log.Println("Error finding contents", id, ":", err.Error())
		return nil
	}
	defer contentRows.Close()
	for contentRows.Next() {
		var childId int
		if err := contentRows.Scan(&childId); err != nil {
			log.Println("Error finding contents", id, ":", err.Error())
			return nil
		}

		thing.Contents = append(thing.Contents, childId)
	}
	if err := contentRows.Err(); err != nil {
		log.Println("Error finding contents", id, ":", err.Error())
		return nil
	}

	return
}

func (w *DatabaseWorld) CreateThing(name string, creator *Thing, parent *Thing) (thing *Thing) {
	thing = NewThing()
	thing.Name = name
	thing.Creator = creator.Id
	thing.Parent = parent.Id

	row := w.db.QueryRow("INSERT INTO thing (name, creator, parent) VALUES ($1, $2, $3) RETURNING id, created",
		thing.Name, thing.Creator, thing.Parent)
	err := row.Scan(&thing.Id, &thing.Created)
	if err != nil {
		log.Println("Error creating a thing", name, ":", err.Error())
		return nil
	}

	return
}

func (w *DatabaseWorld) MoveThing(thing *Thing, target *Thing) (ok bool) {
	_, err := w.db.Exec("UPDATE thing SET parent = $1 WHERE id = $2",
		target.Id, thing.Id)
	if err != nil {
		log.Println("Error moving a thing", thing.Id, ":", err.Error())
		return false
	}
	return true
}

func (w *DatabaseWorld) SaveThing(thing *Thing) (ok bool) {
	tabletext, err := json.Marshal(thing.Table)
	if err != nil {
		log.Println("Error serializing table data for thing", thing.Id, ":", err.Error())
		return false
	}
	_, err = w.db.Exec("UPDATE thing SET tabledata = $1 WHERE id = $2",
		types.JsonText(tabletext), thing.Id)
	if err != nil {
		log.Println("Error saving a thing", thing.Id, ":", err.Error())
		return false
	}
	return true
}

type ActiveWorld struct {
	sync.Mutex
	Things map[int]*Thing
	Next   WorldStore
}

func (w *ActiveWorld) ThingForId(id int) (thing *Thing) {
	w.Lock()
	defer w.Unlock()

	if id == 0 {
		return
	}

	thing, ok := w.Things[id]
	if ok {
		return
	}

	thing = w.Next.ThingForId(id)
	w.Things[id] = thing

	return
}

func (w *ActiveWorld) CreateThing(name string, creator *Thing, parent *Thing) (thing *Thing) {
	thing = w.Next.CreateThing(name, creator, parent)
	parent.Contents = append(parent.Contents, thing.Id)
	w.Things[thing.Id] = thing
	return
}

func (w *ActiveWorld) MoveThing(thing *Thing, target *Thing) (ok bool) {
	if !w.Next.MoveThing(thing, target) {
		return false
	}

	oldParent := w.ThingForId(thing.Parent)

	for i, c := range oldParent.Contents {
		if c != thing.Id {
			continue
		}

		// It matched, so splice out the i'th element.
		copy(oldParent.Contents[i:], oldParent.Contents[i+1:])
		oldParent.Contents = oldParent.Contents[:len(oldParent.Contents)-1]
		log.Println("Removed", thing, "from parent", oldParent, ", remaining things:", oldParent.Contents)
		break
	}

	thing.Parent = target.Id
	target.Contents = append(target.Contents, thing.Id)

	return true
}

func (w *ActiveWorld) SaveThing(thing *Thing) (ok bool) {
	if w.Next.SaveThing(thing) {
		// This must be the newest version of thing in memory. Make sure it's the one we're giving out from now on (just in case).
		w.Things[thing.Id] = thing
		return true
	}
	return false
}
