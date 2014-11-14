package mess

import (
	"fmt"
	"github.com/aarzilli/golua/lua"
	"log"
	"strings"
)

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

func (p *ThingProgram) compile() error {
	state := lua.NewState()
	state.OpenBase()
	state.OpenPackage()
	state.OpenString()
	state.OpenTable()
	state.OpenMath()
	// Not IO & not OS.

	err := state.DoString(p.Text)
	if err != nil {
		p.Error = err
	} else {
		p.state = state
	}
	return err
}

func pushValue(state *lua.State, value interface{}) error {
	switch v := value.(type) {
	case nil:
		state.PushNil()
	case string:
		state.PushString(v)
	case int:
		state.PushInteger(int64(v))
	case int64:
		state.PushInteger(v)
	case float64:
		state.PushNumber(v)
	case bool:
		state.PushBoolean(v)
	case *Thing:
		// TODO: pass along a proxied userdata thing.
		return fmt.Errorf("Unimplemented push of Thing onto lua stack for the whatsit, blubbins")
	default:
		return fmt.Errorf("An item of unknown type was included in the environment or arguments of a Lua call (skipping it): %v",
			value)
	}
	return nil
}

func (p *ThingProgram) TryToCall(name string, env map[string]interface{}, args ...interface{}) error {
	if p.Error != nil {
		// A script that won't compile can't be called, but that counts as trying, so no error.
		return nil
	}
	state := p.state

	// Find the thing named `name` in the global state.
	state.GetGlobal("_G")
	for _, namePart := range strings.Split(name, ".") {
		state.GetField(-1, namePart)
		state.Remove(-2)
		if state.IsNil(-1) {
			// One of our names was invalid, so we can't GetField() any further.
			break
		}
	}
	if state.IsNil(-1) {
		state.Pop(1)  // pop 1 thing, the nil
		// We were unable to find the function, but that counts as trying, so no error.
		return nil
	}

	// Put our local global variables in the global table.
	for name, value := range env {
		err := pushValue(state, value)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		state.SetGlobal(name)
	}

	// Put our args on the stack.
	for _, value := range args {
		err := pushValue(state, value)
		if err != nil {
			log.Println(err.Error())
			state.PushNil()
		}
	}

	// Make the call.
	err := state.Call(len(args), 0)

	// All the provided environment stuff should use "reserved" names. So it's safe to just clear them out of the global table.
	for name, _ := range env {
		state.PushNil()
		state.SetGlobal(name)
	}

	return err
}
