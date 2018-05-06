package vm

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/pkg/errors"
)

type runeReader struct {
	buf     *bufio.Reader
	curLine int
	curCol  int
}

func (rr *runeReader) errf(msg string, args ...interface{}) error {
	for i, v := range args {
		switch v := v.(type) {
		case rune:
			args[i] = fmt.Sprintf("%v(rune:%v)", string(v), v)
		}
	}
	return errors.WithStack(
		errors.Errorf("%v [%v:%v]",
			fmt.Sprintf(msg, args...), rr.curLine, rr.curCol))
}

func (rr *runeReader) consume(n int) (int, error) {
	return rr.buf.Discard(n)
}

func (rr *runeReader) peekRune() (rune, int, error) {
	// try to peek the next rune, expanding on the size one byte at a time
	// up to utf8.UTFMax bytes (which is the biggest encoded utf8 byte sequence)
	for i := 1; i < utf8.UTFMax; i++ {
		b, err := rr.buf.Peek(i)
		if err != nil {
			return utf8.RuneError, 0, err
		}
		if len(b) == 0 {
			return utf8.RuneError, 0, io.EOF
		}
		r, sz := utf8.DecodeRune(b)
		if r != utf8.RuneError {
			return r, sz, nil
		}
	}
	return utf8.RuneError, 0, errors.Errorf("not a valid utf8 text stream")
}

func (rr *runeReader) readRune() (rune, int, error) {
	r, sz, err := rr.peekRune()
	if err != nil {
		return utf8.RuneError, 0, err
	}
	sz, err = rr.buf.Discard(sz)
	return r, sz, err
}
