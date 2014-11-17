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

func MessThingName(state *lua.State, thing *Thing) int {
	state.PushString(thing.Name)
	return 1
}

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

func MessThingTellMethod(state *lua.State, thing *Thing) int {
	state.PushGoFunction(func(state *lua.State) int {
		text := state.CheckString(2)
		if text == "" {
			state.ArgError(2, "`string` text to tell expected")
		}
		thing := checkThing(state, 1)

		if thing.Client != nil {
			thing.Client.ToClient <- text
		}
		state.Pop(2) // ( udataThing strText -- )
		return 0
	})
	return 1
}

var MessThingMembers map[string]MessThingMember = map[string]MessThingMember{
	"name":     MessThingName,
	"contents": MessThingContents,
	"tell":     MessThingTellMethod,
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
	if fieldName == "" {
		state.ArgError(2, "`string` attribute name expected")
	}
	log.Println("Arg #2 checks out, it's a string")

	thing := checkThing(state, 1)
	log.Println("So we're tryin'a look up", fieldName, "on thing", thing.Id)

	if member, ok := MessThingMembers[fieldName]; ok {
		return member(state, thing)
	}

	// uh... I guess we didn't do anything, so...?
	return 0
}

func installWorld(state *lua.State) {
	log.Println("Installing world")
	printStackTypes(state)

	state.NewMetaTable(ThingMetaTableName)         // ( -- mtbl )
	state.SetMetaMethod("__index", MessThingIndex) // ( mtbl -- mtbl )
	state.Pop(1)                                   // ( mtbl -- )

	worldTable := map[string]interface{}{
	/*
		"type": map[string]interface{}{
			"Player":
			"Place":
			"Program":
			"Action":
			"Thing":
		},
		"Root":
		"Create": func...
	*/
	}
	pushValue(state, worldTable)
	state.SetGlobal("world")

	log.Println("Finished installing world")
	printStackTypes(state)
}

func (p *ThingProgram) compile() error {
	state := lua.NewState()
	state.OpenBase()
	state.OpenString()
	state.OpenTable()
	state.OpenMath()
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
