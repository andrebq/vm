package vm

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"unicode"
)

type (
	emitter struct {
		ctx context.Context
		acc []rune
		out chan<- token
	}

	token struct {
		name  string
		value []rune
	}

	lexFn func(*emitter, *runeReader) (lexFn, error)
)

func lexMachine(ctx context.Context, out chan<- token, input io.Reader) error {
	defer close(out)
	cur := initialState
	e := &emitter{
		ctx: ctx,
		out: out,
	}
	rr := &runeReader{
		buf: bufio.NewReader(input),
	}
	for cur != nil {
		var err error
		cur, err = cur(e, rr)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return rr.errf("reached an error state before EOF or other error")
}

func (t token) String() string {
	return fmt.Sprintf("{%v: %v}", t.name,
		string(t.value))
}

func (e *emitter) len() int {
	return len(e.acc)
}

func (e *emitter) push(r rune) {
	e.acc = append(e.acc, r)
}

func (e *emitter) emit(name string) error {
	t := token{
		name: name,
	}
	t.value = e.acc
	e.acc = nil
	select {
	case <-e.ctx.Done():
		return e.ctx.Err()
	case e.out <- t:
		return nil
	}
}

func initialState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	if unicode.IsSpace(r) {
		rr.consume(sz)
		return initialState, nil
	}

	rr.consume(sz)

	switch {
	case r == ')':
		return initialState, e.emit("lclose")
	case r == '(':
		return initialState, e.emit("lopen")
	case r == '-':
		e.push(r)
		return maybeNumberState, nil
	case r == '"':
		return quotedStringState, nil
	case r == '`':
		return multilineStringState, nil
	case unicode.IsDigit(r):
		e.push(r)
		return numberState, nil
	default:
		e.push(r)
		return symbolState, nil
	}
}

func symbolState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	rr.consume(sz)
	switch {
	case unicode.IsSpace(r):
		return initialState, e.emit("symbol")
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' ||
		unicode.IsPunct(r):
		e.push(r)
		return symbolState, nil
	case r == '(':
		return nil, rr.errf("You need to include a whitespace between the symbol and the (")
	}
	return nil, rr.errf("unexpected rune %v", r)
}

func numberState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	switch {
	case unicode.IsSpace(r):
		rr.consume(sz)
		return initialState, e.emit("int_number")
	case r == '_':
		rr.consume(sz)
		return numberState, nil
	case r == '.':
		rr.consume(sz)
		e.push(r)
		return decimalState, nil
	case unicode.IsDigit(r):
		rr.consume(sz)
		e.push(r)
		return numberState, nil
	}
	return nil, rr.errf("unexpected %v", r)
}

func decimalState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	rr.consume(sz)
	switch {
	case unicode.IsSpace(r):
		return initialState, e.emit("decimal_number")
	case unicode.IsDigit(r):
		e.push(r)
		return decimalState, nil
	}
	return nil, rr.errf("unexpected %v", r)
}

func maybeNumberState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	rr.consume(sz)
	e.push(r)
	r, sz, err = rr.peekRune()
	if err != nil {
		return nil, err
	}
	rr.consume(sz)
	e.push(r)
	if unicode.IsDigit(r) {
		return numberState, nil
	}
	// we got a number
	return symbolState, nil
}

func quotedStringState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	rr.consume(sz)
	switch {
	case r == '\\':
		return escapeStringState, nil
	case r == '"':
		e.emit("string")
		return initialState, nil
	case r == '\r' || r == '\n':
		return nil, rr.errf("use '`' to encode multiline strings")
	}
	e.push(r)
	return quotedStringState, nil
}

func escapeStringState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	rr.consume(sz)
	switch {
	case r == '"':
		e.push('"')
	case r == '\\':
		e.push('\\')
	default:
		return nil, rr.errf("invalid escape quote %v", r)
	}
	return quotedStringState, nil
}

func multilineStringState(e *emitter, rr *runeReader) (lexFn, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return nil, err
	}
	rr.consume(sz)
	switch {
	case r == '`':
		e.emit("string")
		return initialState, nil
	}
	e.push(r)
	return multilineStringState, nil
}

