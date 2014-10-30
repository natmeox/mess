package mess

import (
	"database/sql"
	"log"
	"sync"
)

type WorldStore interface {
	ThingForId(id int) *Thing
	CreateThing(name string, creator *Thing, parent *Thing) (thing *Thing)
	MoveThing(thing *Thing, target *Thing) (ok bool)
}

type DatabaseWorld struct {
	db *sql.DB
}

func (w *DatabaseWorld) ThingForId(id int) (thing *Thing) {
	thing = &Thing{}
	thing.Id = id

	row := w.db.QueryRow("SELECT name, description, creator, created, parent FROM thing WHERE id = $1",
		id)
	var parent sql.NullInt64
	err := row.Scan(&thing.Name, &thing.Description, &thing.Creator, &thing.Created, &parent)
	if err != nil {
		log.Println("Error finding thing", id, ":", err.Error())
		return nil
	}
	if parent.Valid {
		thing.Parent = int(parent.Int64)
	}

	thing.Contents = make([]int, 0, 2)

	rows, err := w.db.Query("SELECT id FROM thing WHERE parent = $1", id)
	if err != nil {
		log.Println("Error finding contents", id, ":", err.Error())
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var childId int
		if err := rows.Scan(&childId); err != nil {
			log.Println("Error finding contents", id, ":", err.Error())
			return nil
		}

		thing.Contents = append(thing.Contents, childId)
	}
	if err := rows.Err(); err != nil {
		log.Println("Error finding contents", id, ":", err.Error())
		return nil
	}

	return
}

func (w *DatabaseWorld) CreateThing(name string, creator *Thing, parent *Thing) (thing *Thing) {
	thing = &Thing{
		Name:        name,
		Description: "",
		Creator:     creator.Id,
		Parent:      parent.Id,
		Contents:    make([]int, 0),
	}

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

type ActiveWorld struct {
	sync.Mutex
	Things map[int]*Thing
	Next   WorldStore
}

func (w *ActiveWorld) ThingForId(id int) (thing *Thing) {
	w.Lock()
	defer w.Unlock()

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