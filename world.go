package mess

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var thingIdListExp = regexp.MustCompile(`\d+`)

func (l *ThingIdList) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return errors.New("Scan source was not []byte")
	}

	matches := thingIdListExp.FindAllStringSubmatch(string(asBytes), -1)
	results := make([]ThingId, len(matches))
	for i, match := range matches {
		value64, err := strconv.ParseInt(match[0], 0, 64)
		if err != nil {
			return err
		}

		results[i] = ThingId(value64)
	}

	// Assign over ourself in place.
	(*l) = results
	return nil
}

func (l ThingIdList) Value() (driver.Value, error) {
	thingIds := make([]interface{}, len(l))
	for i, id := range l {
		thingIds[i] = int(id)
	}
	spacedList := fmt.Sprint(thingIds...)
	commaedList := strings.Replace(spacedList, " ", ",", -1)
	arrayLiteral := fmt.Sprintf("{%s}", commaedList)
	return []byte(arrayLiteral), nil
}

type WorldStore interface {
	ThingForId(id ThingId) *Thing
	CreateThing(name string, tt ThingType, creator *Thing, parent *Thing) (thing *Thing)
	MoveThing(thing *Thing, target *Thing) (ok bool)
	SaveThing(thing *Thing) (ok bool)
}

type DatabaseWorld struct {
	db *sql.DB
}

func (w *DatabaseWorld) ThingForId(id ThingId) (thing *Thing) {
	if id == 0 {
		return nil
	}

	thing = NewThing()
	thing.Id = id

	row := w.db.QueryRow("SELECT type, name, creator, created, owner, adminlist, allowlist, denylist, parent, tabledata, program FROM thing WHERE id = $1",
		id)
	var typetext string
	var creator sql.NullInt64
	var owner sql.NullInt64
	var parent sql.NullInt64
	var tabledata types.JsonText
	var program sql.NullString
	err := row.Scan(&typetext, &thing.Name, &creator, &thing.Created, &owner,
		&thing.AdminList, &thing.AllowList, &thing.DenyList, &parent, &tabledata,
		&program)
	if err != nil {
		log.Println("Error finding thing", id, ":", err.Error())
		return nil
	}
	thing.Type = ThingTypeForName(typetext)
	if creator.Valid {
		thing.Creator = ThingId(creator.Int64)
	}
	if owner.Valid {
		thing.Owner = ThingId(owner.Int64)
	}
	if parent.Valid {
		thing.Parent = ThingId(parent.Int64)
	}
	if program.Valid {
		thing.Program = NewProgram(program.String)
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
		var childId ThingId
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

func (w *DatabaseWorld) CreateThing(name string, tt ThingType, creator *Thing, parent *Thing) (thing *Thing) {
	thing = NewThing()
	thing.Name = name
	thing.Type = tt
	thing.Parent = parent.Id

	var creatorId sql.NullInt64
	if creator != nil && thing.Type != PlayerThing {
		creatorId.Int64 = int64(creator.Id)
		thing.Creator = creator.Id
		thing.Owner = creator.Id
	}

	row := w.db.QueryRow("INSERT INTO thing (name, type, creator, owner, parent) VALUES ($1, $2, $3, $4, $5) RETURNING id, created",
		thing.Name, thing.Type.String(), creatorId, creatorId, thing.Parent)
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

	var parent sql.NullInt64
	if thing.Parent != 0 {
		parent.Int64 = int64(thing.Parent)
		parent.Valid = true
	}
	var owner sql.NullInt64
	if thing.Owner != 0 {
		owner.Int64 = int64(thing.Owner)
		owner.Valid = true
	}
	var program sql.NullString
	if thing.Program != nil {
		program.String = thing.Program.Text
		program.Valid = true
	}

	// TODO: save the allow list
	_, err = w.db.Exec("UPDATE thing SET name = $1, parent = $2, owner = $3, adminlist = $4, denylist = $5, tabledata = $6, program = $7 WHERE id = $8",
		thing.Name, parent, owner, thing.AdminList, thing.DenyList,
		types.JsonText(tabletext), program, thing.Id)
	if err != nil {
		log.Println("Error saving a thing", thing.Id, ":", err.Error())
		return false
	}
	return true
}

type ActiveWorld struct {
	sync.Mutex
	Things map[ThingId]*Thing
	Next   WorldStore
}

func (w *ActiveWorld) ThingForId(id ThingId) (thing *Thing) {
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

func (w *ActiveWorld) CreateThing(name string, tt ThingType, creator *Thing, parent *Thing) (thing *Thing) {
	thing = w.Next.CreateThing(name, tt, creator, parent)
	if thing == nil {
		return
	}

	log.Println("Created a thing", thing, ", adding to parent's in-memory contents")
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
