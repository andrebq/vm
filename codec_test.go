package vm

import (
	"bytes"
	"reflect"
	"testing"
)

func TestReadWrite(t *testing.T) {
	list := []interface{}{
		float64(1),
		int64(2),
		"abc",
		Symbol("abc"),
		[]byte("abc"),
		make([]interface{}, 0),
	}
	buf := &bytes.Buffer{}
	tw := NewTypedWriter(buf)
	err := tw.WriteList(list)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := (NewTypedReader(bytes.NewBuffer(buf.Bytes()))).ReadList()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(list, decoded) {
		t.Errorf("should read %#v got %#v", decoded, list)
	}
}
