package mess

import (
	"fmt"
	"github.com/aarzilli/golua/lua"
	"log"
	"strings"
	"unsafe"
)

const ThingMetaTableName = "Mess.Thing"

type ThingProgram struct {
	Text  string
	Error error
	state *lua.State
}

func NewProgram(text string) (p *ThingProgram) {
	p = &ThingProgram{
		Text: text,
	}
	p.compile()
	return p
}

func pushValue(state *lua.State, value interface{}) error {
	switch v := value.(type) {
	default:
		return fmt.Errorf("An item of unknown type was included in the environment or arguments of a Lua call (skipping it): %v",
			value)
	case nil:
		log.Println("Pushing nil onto lua stack")
		state.PushNil()
	case string:
		log.Println("Pushing string onto lua stack")
		state.PushString(v)
	case int:
		log.Println("Pushing int onto lua stack")
		state.PushInteger(int64(v))
	case int64:
		log.Println("Pushing int64 onto lua stack")
		state.PushInteger(v)
	case float64:
		log.Println("Pushing float64 onto lua stack")
		state.PushNumber(v)
	case bool:
		log.Println("Pushing bool onto lua stack")
		state.PushBoolean(v)

	case map[string]interface{}:
		log.Println("Pushing map[string]interface{} onto lua stack")
		state.CreateTable(0, len(v))
		for name, value := range v {
			err := pushValue(state, value)
			if err != nil {
				// error means nothing was added to stack. So pop our new table so *we* leave nothing added to the stack.
				state.Pop(1)
				return err
			}
			state.SetField(-2, name)
		}
		// then leave the table on the stack

	case ThingType:
		// These are singleton sentinel values, so load them from Lua-land.
		state.GetGlobal("world")
		state.GetField(-1, strings.Title(v.String()))
		state.Remove(-2)

	case *Thing:
		log.Println("Pushing *Thing onto lua stack")
		return pushValue(state, v.Id)
	case ThingId:
		log.Println("Pushing ThingId onto lua stack")
		// We're pushing a ThingId, so make a new userdata for it, with the Thing metatable.
		userdata := state.NewUserdata(uintptr(unsafe.Sizeof(int64(0))))
		thingPtr := (*int64)(userdata)
		*thingPtr = int64(v)
		if !state.IsUserdata(-1) {
			log.Println("!!! HOGAD JUST PUSHED NEW USERDATA BUT IT ISN'T OMG !!!")
		}
		log.Println("Pushed ThingId", *thingPtr, "onto lua stack")

		// Now make it act like a Thing.
		state.LGetMetaTable(ThingMetaTableName) // ( udata -- udata mtbl )
		state.SetMetaTable(-2)                  // ( udata mtbl -- udata )

		// Let's just check that it's that, for sures.
		if !state.IsUserdata(-1) {
			log.Println("!!! WOOP WOOP DID NOT SET METATABLE RIGHT :( !!!")
		}
	}
	return nil
}

func checkThing(state *lua.State, argNum int) *Thing {
	userdata := state.CheckUdata(argNum, ThingMetaTableName)
	if userdata == nil {
		state.ArgError(argNum, "`Thing` expected")
	}

	var thingPtr *int64
	thingPtr = (*int64)(userdata)
	thingId := ThingId(*thingPtr)

	thing := World.ThingForId(thingId)
	if thing == nil {
		state.ArgError(argNum, "`Thing` argument is no longer valid")
	}

	return thing
}

type MessThingMember func(state *lua.State, thing *Thing) int

func MessThingContents(state *lua.State, thing *Thing) int {
	// make a new table
	state.CreateTable(len(thing.Contents), 0) // ( -- tbl )
	// for each content add a new lua-space Thing
	for i, contentId := range thing.Contents {
		state.PushInteger(int64(i))
		pushValue(state, contentId)
		state.SetTable(-3) // ( tbl key val -- tbl )
	} // ( tbl -- tbl )
	return 1
}

func MessThingFindnearMethod(state *lua.State, thing *Thing) int {
	state.PushGoFunction(func(state *lua.State) int {
		thing := checkThing(state, 1)
		text := state.CheckString(2)
		if text == "" {
			state.ArgError(2, "cannot find empty string")
		}

		otherThing := thing.FindNear(text)
		if otherThing == nil {
			return 0
		}
		pushValue(state, otherThing.Id)
		return 1
	})
	return 1
}

func MessThingFindinsideMethod(state *lua.State, thing *Thing) int {
	state.PushGoFunction(func(state *lua.State) int {
		thing := checkThing(state, 1)
		text := state.CheckString(2)
		if text == "" {
			state.ArgError(2, "cannot find empty string")
		}

		otherThing := thing.FindInside(text)
		if otherThing == nil {
			return 0
		}
		pushValue(state, otherThing.Id)
		return 1
	})
	return 1
}

func MessThingMovetoMethod(state *lua.State, thing *Thing) int {
	state.PushGoFunction(func(state *lua.State) int {
		source := checkThing(state, 1)
		target := checkThing(state, 2)

		ok := source.MoveTo(target)

		state.PushBoolean(ok)
		return 1
	})
	return 1
}

func MessThingName(state *lua.State, thing *Thing) int {
	state.PushString(thing.Name)
	return 1
}

func MessThingPronounsubMethod(state *lua.State, thing *Thing) int {
	state.PushGoFunction(func(state *lua.State) int {
		thing := checkThing(state, 1)
		text := state.CheckString(2)

		for code, pronoun := range thing.Pronouns() {
			lowerCode := fmt.Sprintf(`%%%s`, code)
			upperCode := fmt.Sprintf(`%%%s`, strings.ToUpper(code))
			text = strings.Replace(text, lowerCode, pronoun, -1)
			text = strings.Replace(text, upperCode, strings.ToTitle(pronoun), -1)
		}

		text = strings.Replace(text, `%n`, thing.Name, -1)
		text = strings.Replace(text, `%N`, thing.Name, -1)

		state.Pop(2)           // ( udataThing str -- )
		state.PushString(text) // ( -- str' )
		return 1
	})
	return 1
}

func MessThingTellMethod(state *lua.State, thing *Thing) int {
	state.PushGoFunction(func(state *lua.State) int {
		thing := checkThing(state, 1)
		text := state.CheckString(2)

		if thing.Client != nil {
			thing.Client.ToClient <- text
		}
		state.Pop(2) // ( udataThing strText -- )
		return 0
	})
	return 1
}

func MessThingTellallMethod(state *lua.State, thing *Thing) int {
	state.PushGoFunction(func(state *lua.State) int {
		place := checkThing(state, 1)
		text := state.CheckString(2)

		// If arg 3 is present, it should be a table of Things to exclude.
		excludes := make(map[ThingId]bool)
		if 2 < state.GetTop() {
			if !state.IsTable(3) {
				state.ArgError(3, "expected `table` for exclude argument if present")
			}
			numExcludes := int(state.ObjLen(3))
			for i := 0; i < numExcludes; i++ {
				state.RawGeti(3, i+1)
				exclude := checkThing(state, -1)
				excludes[exclude.Id] = true
			}
		}

		for _, content := range place.GetContents() {
			if excludes[content.Id] {
				continue
			}
			if content.Client != nil {
				content.Client.ToClient <- text
			}
		}

		return 0
	})
	return 1
}

func MessThingType(state *lua.State, thing *Thing) int {
	pushValue(state, thing.Type)
	return 1
}

var MessThingMembers map[string]MessThingMember = map[string]MessThingMember{
	"contents":   MessThingContents,
	"findinside": MessThingFindinsideMethod,
	"findnear":   MessThingFindnearMethod,
	"moveto":     MessThingMovetoMethod,
	"name":       MessThingName,
	"pronounsub": MessThingPronounsubMethod,
	"tell":       MessThingTellMethod,
	"tellall":    MessThingTellallMethod,
	"type":       MessThingType,
}

func MessThingIndex(state *lua.State) int {
	log.Println("HEY WE MADE IT")
	printStackTypes(state)

	state.GetMetaTable(1)
	state.LGetMetaTable(ThingMetaTableName)
	isThing := state.RawEqual(-1, -2)
	state.Pop(2)
	if !isThing {
		log.Println("!!! OMG ARG #1 IS NOT A MESS.THING !!!")
	}

	fieldName := state.CheckString(2)
	log.Println("Arg #2 checks out, it's a string")
	thing := checkThing(state, 1)
	log.Println("So we're tryin'a look up", fieldName, "on thing", thing.Id)

	if member, ok := MessThingMembers[fieldName]; ok {
		return member(state, thing)
	}

	// That wasn't one of our members, so look it up in our Table.
	if data, ok := thing.Table[fieldName]; ok {
		// TODO: instead of pushing a whole map if the script asks for one, maybe we should use another kind of userdata that tracks the name & can access its submembers until the script asks for the leaf (or a non-existent branch)?
		pushValue(state, data)
		return 1
	}

	// uh... I guess we didn't do anything, so...?
	return 0
}

func TableFormat(state *lua.State) int {
	state.CheckType(1, lua.LUA_TTABLE)
	state.CheckType(2, lua.LUA_TTABLE) // ( ??? -- tbl tblFields )
	log.Println("Formatting a table by a table")
	printStackTypes(state)

	numFields := int(state.ObjLen(-1))
	fields := make([]string, numFields)
	maxFieldLen := make(map[string]int)
	for i := 0; i < numFields; i++ {
		state.RawGeti(-1, i+1) // ( tbl tblFields -- tbl tblFields strHeader )
		fieldName := state.ToString(-1)
		fields[i] = fieldName
		maxFieldLen[fieldName] = len(fieldName)
		state.Pop(1) // ( tbl tblFields strField -- tbl tblFields )
	}
	state.Pop(1) // ( tbl tblFields -- tbl )
	log.Println("Slurped up the fields list (table #2)")
	printStackTypes(state)

	numRows := int(state.ObjLen(-1))
	rows := make([]map[string]string, numRows)
	for i := 0; i < numRows; i++ {
		state.RawGeti(-1, i+1) // ( tbl -- tbl tblRow )
		row := make(map[string]string)
		for _, field := range fields {
			state.PushString(field) // ( tbl tblRow -- tbl tblRow strField )
			state.RawGet(-2)        // ( tbl tblRow strField -- tbl tblRow tblField )

			row[field] = state.ToString(-1)

			if maxFieldLen[field] < len(row[field]) {
				maxFieldLen[field] = len(row[field])
			}

			state.Pop(1) // ( tbl tblRow tblField -- tbl tblRow )
		}
		rows[i] = row
		state.Pop(1) // ( tbl tblRow -- tbl )
	}
	state.Pop(1) // ( tbl -- )
	log.Println("Slurped up the data table (table #1)")
	printStackTypes(state)

	// %5s  %10s  %13s
	fmtStrings := make([]string, numFields)
	for i, field := range fields {
		fmtStrings[i] = fmt.Sprintf("%%-%ds", maxFieldLen[field])
	}
	fmtString := strings.Join(fmtStrings, "  ")
	log.Println("Figured out the format string:", fmtString)

	rowStrings := make([]string, numRows+1)
	rowFields := make([]interface{}, numFields)
	for i, row := range rows {
		for j, field := range fields {
			rowFields[j] = row[field]
		}
		rowStrings[i+1] = fmt.Sprintf(fmtString, rowFields...)
	}
	for i := 0; i < numFields; i++ {
		rowFields[i] = strings.Title(fields[i])
	}
	rowStrings[0] = fmt.Sprintf(fmtString, rowFields...)
	log.Println("Yay formatted all the strings")

	formattedTable := strings.Join(rowStrings, "\n")
	state.PushString(formattedTable) // ( -- str )
	log.Println("All done formatting this table!")
	printStackTypes(state)

	return 1
}

func installWorld(state *lua.State) {
	log.Println("Installing world")
	printStackTypes(state)

	state.NewMetaTable(ThingMetaTableName)         // ( -- mtbl )
	state.SetMetaMethod("__index", MessThingIndex) // ( mtbl -- mtbl )
	state.Pop(1)                                   // ( mtbl -- )

	worldTable := map[string]interface{}{
	/*
		"Root":
		"Create": func...
	*/
	}
	pushValue(state, worldTable)

	// Install Thing types as singleton sentinel values. As userdata, these will only compare if the values are exactly equal.
	state.NewUserdata(uintptr(0))
	state.SetField(-2, "Player")
	state.NewUserdata(uintptr(0))
	state.SetField(-2, "Place")
	state.NewUserdata(uintptr(0))
	state.SetField(-2, "Program")
	state.NewUserdata(uintptr(0))
	state.SetField(-2, "Action")
	state.NewUserdata(uintptr(0))
	state.SetField(-2, "Thing")

	state.SetGlobal("world")

	state.GetGlobal("table") // ( -- tblTable )
	state.PushGoFunction(TableFormat)
	state.SetField(-2, "format")
	state.Pop(1) // ( tblTable -- )

	log.Println("Finished installing world")
	printStackTypes(state)
}

func (p *ThingProgram) compile() error {
	state := lua.NewState()
	state.OpenBase()
	state.OpenMath()
	state.OpenString()
	state.OpenTable()
	// Not IO & not OS.
	// Not package: all our packages are preloaded.

	// Install the `world` package.
	installWorld(state)

	err := state.DoString(p.Text)
	if err != nil {
		p.Error = err
	} else {
		p.state = state
	}
	return err
}

func printStackTypes(state *lua.State) {
	topIndex := state.GetTop()
	segments := make([]interface{}, topIndex+1)
	segments[0] = "Stack types:"
	for i := 1; i <= topIndex; i++ {
		segments[i] = state.LTypename(i)
	}
	log.Println(segments...)
}

func (p *ThingProgram) TryToCall(name string, env map[string]interface{}, args ...interface{}) error {
	if p.Error != nil {
		// A script that won't compile can't be called, but that counts as trying, so no error.
		return nil
	}
	state := p.state
	printStackTypes(state)

	// Find the thing named `name` in the global state.
	state.GetGlobal("_G") // ( -- tbl )
	for _, namePart := range strings.Split(name, ".") {
		state.GetField(-1, namePart) // ( tbl -- tbl tblNext? )
		state.Remove(-2)             // ( tbl tblNext? -- tblNext? )
		if state.IsNil(-1) {
			// One of our names was invalid, so we can't GetField() any further.
			break
		}
	} // ( tblNext? -- val? )
	printStackTypes(state)
	// TODO: could this be a Go function instead?
	if !state.IsFunction(-1) {
		state.Pop(1) // ( val? -- )
		// We were unable to find the function, but that counts as trying, so no error.
		return nil
	} // ( val? -- func )
	log.Println("Found our function", name)
	printStackTypes(state)

	// Put our local global variables in the global table.
	for name, value := range env { // ( func -- )
		log.Println("Adding", name, ":", value, "to softcode globals")
		printStackTypes(state)

		err := pushValue(state, value) // ( func -- func val? )
		if err != nil {                // if error, pushValue() left the stack at +0
			log.Println("Error pushing softcode global", name, "onto stack (using nil instead):", err.Error())
			state.PushNil() // ( -- func val )
		} // ( func val? -- func val )
		state.SetGlobal(name) // ( func val -- func )
	} // ( func -- func )
	printStackTypes(state)

	// Put our args on the stack.
	for i, value := range args { // ( func -- func args... )
		log.Println("Adding softcode arg", i, ":", value)
		err := pushValue(state, value)
		if err != nil {
			log.Println("Error pushing softcode arg", i, "on stack (using nil instead):", err.Error())
			state.PushNil()
		}
	} // ( func -- func args... )
	printStackTypes(state)

	// Make the call.
	funcPos := -1 - len(args) // so -1 with 0 args, -2 with 1 arg, etc
	if !state.IsFunction(funcPos) {
		log.Println("Pushed", len(args), "args onto the stack but the thing at position",
			funcPos, "is no longer our function but a", state.LTypename(funcPos), "??")
	}
	log.Println("Calling function at stack pos", funcPos, "with", len(args), "args")
	err := state.Call(len(args), 0) // ( func -- strErr? )
	log.Println("Whoa back from call!")
	printStackTypes(state)
	if err != nil {
		// Pop the error the pcall (it's actually a lua_pcall() ) left on the stack.
		state.Pop(1)
	}

	// All the provided environment stuff should use "reserved" names. So it's safe to just clear them out of the global table.
	for name, _ := range env {
		state.PushNil()       // ( -- nil )
		state.SetGlobal(name) // ( nil -- )
	} // ( -- )
	log.Println("Cleaned up globals")
	printStackTypes(state)

	return err
}
