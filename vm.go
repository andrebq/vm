package vm

import (
	"fmt"
	"io"
	"os"

	"github.com/yuin/gopher-lua"
)

type (
	V struct {
		state *lua.LState
		err   io.Writer
	}
)

// New returns a new instance of a VM with the least ammount of
// functions and packages loaded
//
// Permissions can be added by the host application
func New() *V {
	v := &V{
		state: lua.NewState(lua.Options{
			CallStackSize:       1000,
			IncludeGoStackTrace: false,
			SkipOpenLibs:        true,
		}),
		err: os.Stderr,
	}
	v.GiveCapability("print", v.printFn)
	return v
}

func (v *V) Run(code string) error {
	return v.state.DoString(code)
}

// Add the given function to the global state
// TODO(andre): should we hide the fact this is a LuaVM?
func (v *V) GiveCapability(name string, fn lua.LGFunction) {
	v.state.SetGlobal(name, v.state.NewFunction(fn))
}

func (v *V) printFn(L *lua.LState) int {
	args := L.GetTop()
	for i := 1; i <= args; i++ {
		lv := L.Get(i)
		if i == 1 {
			v.printf("%v", lv.String())
		} else {
			v.printf(" %v", lv.String())
		}
	}
	v.printf("\n")
	return 0
}

func (v *V) printf(msg string, args ...interface{}) {
	fmt.Fprintf(v.err, msg, args...)
}
