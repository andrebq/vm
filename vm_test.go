package vm

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/yuin/gopher-lua"
)

func TestPrintln(t *testing.T) {
	vm := New()
	buf := &bytes.Buffer{}
	vm.err = buf

	now := time.Now()
	vm.GiveCapability("get_time", func(L *lua.LState) int {
		// giving the capability to read the current time
		// but since we control it, the get_time can only read
		// the same value
		L.Push(lua.LString(now.Format(time.RFC3339)))
		return 1
	})

	if err := vm.Run("print(get_time())"); err != nil {
		t.Fatal(err)
	}

	if buf.String() != fmt.Sprintf("%v\n", now.Format(time.RFC3339)) {
		t.Errorf("unexpected output [%q]", buf.String())
	}
}
