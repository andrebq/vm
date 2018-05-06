package vm

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type (
	// TypedReader transforms binary encoded data into
	// typed values
	TypedReader struct {
		r   *bufio.Reader
		err error
	}

	// TypedWriter takes typed values and generates a
	// stream of binary encoded entries
	TypedWriter struct {
		w     io.Writer
		err   error
		total int64
	}
)

// NewTypedReader returns a new instance of a typed reader
// reading data from r
func NewTypedReader(r io.Reader) *TypedReader {
	switch r := r.(type) {
	case *bufio.Reader:
		return &TypedReader{r: r, err: nil}
	case *TypedReader:
		return r
	}
	return &TypedReader{r: bufio.NewReader(r), err: nil}
}

// PeekType returns the next type, if any.
func (tr *TypedReader) PeekType() (ValueType, error) {
	var buf []byte
	buf, tr.err = tr.Peek(1)
	if tr.err != nil {
		return Undefined, tr.err
	}
	return ValueType(buf[0]), tr.err
}

// ReadType returns the next type and consumes input
func (tr *TypedReader) ReadType() (ValueType, error) {
	var b byte
	b, tr.err = tr.readByte()
	if tr.err != nil {
		return Undefined, tr.err
	}
	if b < byte(List) || b > byte(Blob) {
		tr.err = errors.New("invalid type")
		return Undefined, tr.err
	}
	return ValueType(b), nil
}

// ReadTypeAs returns true if the type is expected
func (tr *TypedReader) ReadTypeAs(expected ValueType) (bool, error) {
	var tp ValueType
	tp, tr.err = tr.ReadType()
	if tp != expected {
		tr.err = errors.New("unexpected type")
	}
	return tp == expected, tr.err
}

func (tr *TypedReader) readByte() (byte, error) {
	if tr.err != nil {
		return 0, tr.err
	}
	out := make([]byte, 1)
	tr.Read(out)
	return out[0], tr.err
}

// Peek one n bytes without consuming data
func (tr *TypedReader) Peek(n int) ([]byte, error) {
	if tr.err != nil {
		return nil, tr.err
	}
	var buf []byte
	buf, tr.err = tr.r.Peek(n)
	return buf, tr.err
}

// Read implements io.Reader interface
func (tr *TypedReader) Read(out []byte) (int, error) {
	if tr.err != nil {
		return 0, tr.err
	}
	var sz int
	sz, tr.err = tr.r.Read(out)
	return sz, tr.err
}

// ReadSize consumes an int64 value and return it as int
func (tr *TypedReader) ReadSize() (int, error) {
	var sz int64
	sz, tr.err = tr.ReadInt64()
	return int(sz), tr.err
}

// ReadInt64 consumes one int64 value
func (tr *TypedReader) ReadInt64() (int64, error) {
	ok, _ := tr.ReadTypeAs(Integer)
	if !ok {
		return 0, tr.err
	}

	var out int64
	tr.err = binary.Read(tr, binary.BigEndian, &out)
	return out, tr.err
}

// ReadDouble consumes one float64 value
func (tr *TypedReader) ReadDouble() (float64, error) {
	ok, _ := tr.ReadTypeAs(Double)
	if !ok {
		return 0, tr.err
	}
	var out float64
	tr.err = binary.Read(tr, binary.BigEndian, &out)
	return out, tr.err
}

// ReadString consumes one length prefixed utf-8 string
func (tr *TypedReader) ReadString() (string, error) {
	ok, _ := tr.ReadTypeAs(String)
	if !ok {
		return "", tr.err
	}
	sz, _ := tr.ReadSize()
	if sz == 0 {
		return "", tr.err
	}
	buf := make([]byte, sz)
	_, tr.err = tr.Read(buf)
	if tr.err != nil {
		return "", tr.err
	}
	if !utf8.Valid(buf) {
		tr.err = errors.New("invalid utf8")
		return "", tr.err
	}
	return string(buf), tr.err
}

// ReadSymbol consumes one symbol, a symbol is a utf-8 string
// without whitespaces
func (tr *TypedReader) ReadSymbol() (Symbol, error) {
	ok, _ := tr.ReadTypeAs(Sym)
	if !ok {
		return Symbol(""), tr.err
	}
	sz, _ := tr.ReadSize()
	if sz == 0 {
		return "", tr.err
	}
	buf := make([]byte, sz)
	_, tr.err = tr.Read(buf)
	if tr.err != nil {
		return "", tr.err
	}
	if !utf8.Valid(buf) {
		tr.err = errors.New("invalid utf8")
		return "", tr.err
	}
	// TODO(andre): add validation to check for whitespace
	return Symbol(string(buf)), tr.err
}

// ReadBlob consumes a length prefixed block of uninterpreted bytes
func (tr *TypedReader) ReadBlob() ([]byte, error) {
	ok, _ := tr.ReadTypeAs(Blob)
	if !ok {
		return nil, tr.err
	}
	sz, _ := tr.ReadSize()
	if sz == 0 {
		return nil, tr.err
	}
	buf := make([]byte, sz)
	_, tr.err = tr.Read(buf)
	return buf, tr.err
}

// ReadList consumes n items from the stream with the correct types
func (tr *TypedReader) ReadList() ([]interface{}, error) {
	ok, _ := tr.ReadTypeAs(List)
	if !ok {
		return nil, tr.err
	}

	sz, _ := tr.ReadSize()
	lst := make([]interface{}, 0)
	for i := 0; i < sz; i++ {
		var val interface{}
		val, tr.err = tr.readNextItem()
		if tr.err != nil {
			return nil, tr.err
		}
		lst = append(lst, val)
	}
	return lst, tr.err
}

func (tr *TypedReader) readNextItem() (interface{}, error) {
	var vt ValueType
	vt, tr.err = tr.PeekType()
	if tr.err != nil {
		return nil, tr.err
	}
	var val interface{}
	switch vt {
	case Integer:
		val, _ = tr.ReadInt64()
	case Double:
		val, _ = tr.ReadDouble()
	case String:
		val, _ = tr.ReadString()
	case Sym:
		val, _ = tr.ReadSymbol()
	case Blob:
		val, _ = tr.ReadBlob()
	case List:
		val, _ = tr.ReadList()
	default:
		tr.err = errors.New("invalid type")
	}
	return val, tr.err
}

// Err returns the first error found during scanning
func (tr *TypedReader) Err() error {
	return tr.err
}

// NewTypedWriter returns a typed writer backed by the given writer
func NewTypedWriter(w io.Writer) *TypedWriter {
	if tw, ok := w.(*TypedWriter); ok {
		return tw
	}
	return &TypedWriter{w: w, err: nil, total: 0}
}

// Write implements io.Writer
func (tw *TypedWriter) Write(out []byte) (int, error) {
	if tw.err != nil {
		return 0, tw.err
	}
	var sz int
	sz, tw.err = tw.w.Write(out)
	tw.total += int64(sz)
	return sz, tw.err
}

// WriteType writes the given type
func (tw *TypedWriter) WriteType(t ValueType) error {
	tw.Write([]byte{byte(t)})
	return tw.err
}

// WriteSize takes the int value and writes it as int64
func (tw *TypedWriter) WriteSize(sz int) error {
	tw.WriteInt64(int64(sz))
	return tw.err
}

// WriteInt64 self explanatory
func (tw *TypedWriter) WriteInt64(v int64) error {
	tw.WriteType(Integer)
	binary.Write(tw, binary.BigEndian, v)
	return tw.err
}

// WriteDouble self explanatory
func (tw *TypedWriter) WriteDouble(v float64) error {
	tw.WriteType(Double)
	binary.Write(tw, binary.BigEndian, v)
	return tw.err
}

// WriteString self explanatory
func (tw *TypedWriter) WriteString(v string) error {
	if !utf8.ValidString(v) {
		tw.err = errors.New("invalid utf8 string")
		return tw.err
	}
	tw.WriteType(String)
	tw.WriteSize(len(v))
	io.WriteString(tw, v)
	return tw.err
}

// WriteSymbol self explanatory
func (tw *TypedWriter) WriteSymbol(v Symbol) error {
	if !utf8.ValidString(string(v)) {
		tw.err = errors.New("invalid utf8 string")
		return tw.err
	}
	tw.WriteType(Sym)
	tw.WriteSize(len(v))
	io.WriteString(tw, string(v))
	return tw.err
}

// WriteBlob self explanatory
func (tw *TypedWriter) WriteBlob(v []byte) error {
	tw.WriteType(Blob)
	tw.WriteSize(len(v))
	tw.Write(v)
	return tw.err
}

// WriteList self explanatory
func (tw *TypedWriter) WriteList(val []interface{}) error {
	tw.WriteType(List)
	tw.WriteSize(len(val))
	for _, v := range val {
		tw.err = tw.writeNextValue(v)
		if tw.err != nil {
			return tw.err
		}
	}
	return tw.err
}

func (tw *TypedWriter) writeNextValue(v interface{}) error {
	switch v := v.(type) {
	case string:
		tw.WriteString(v)
	case []byte:
		tw.WriteBlob(v)
	case Symbol:
		tw.WriteSymbol(v)
	case float64:
		tw.WriteDouble(v)
	case int64:
		tw.WriteInt64(v)
	case []interface{}:
		tw.WriteList(v)
	default:
		tw.err = fmt.Errorf("cannot encode value %#v", v)
	}
	return tw.err
}
