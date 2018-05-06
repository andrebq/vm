package vm

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"
)

func TestLexer(t *testing.T) {
	input := bytes.NewBufferString(`(a 123 123.3 -123 -123.3 "str" "ab\"\\")`)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	time.AfterFunc(time.Second, func() {
		panic("1 sec is too much")
	})

	out := make(chan token)

	go func() {
		err := lexMachine(ctx, out, input)
		if err != nil {
			t.Fatal(err)
		}
	}()

	var acc []token
	for t := range out {
		acc = append(acc, t)
	}

	expected := []token{
		{name: "lopen"},
		{name: "symbol", value: []rune("a")},
		{name: "int_number", value: []rune("123")},
		{name: "decimal_number", value: []rune("123.3")},
		{name: "int_number", value: []rune("-123")},
		{name: "decimal_number", value: []rune("-123.3")},
		{name: "string", value: []rune("str")},
		{name: "string", value: []rune(`ab"\`)},
		{name: "lclose"},
	}

	if !reflect.DeepEqual(acc, expected) {
		t.Fatalf("expecting %v got %v", expected, acc)
	}
}
