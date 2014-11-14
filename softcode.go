package mess

import (
	"fmt"
	"github.com/aarzilli/golua/lua"
	"github.com/stevedonovan/luar"
	"log"
	"strings"
	"time"
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
	state := luar.Init()
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

func (p *ThingProgram) TryToCall(name string, env map[string]interface{}, args ...interface{}) error {
	if p.Error != nil {
		return nil
	}

	fn := luar.NewLuaObjectFromName(p.state, name)
	if fn == nil {
		return nil
	}

	// as in #define lua_pushglobaltable
	// lua.LUA_RIDX_GLOBALS == 2
	p.state.RawGeti(lua.LUA_REGISTRYINDEX, 2)
	// Put our local global variables in it.
	luar.Register(p.state, "*", luar.Map(env))
	p.state.Pop(-1)

	_, err := fn.Call(args...)
	return err
}
