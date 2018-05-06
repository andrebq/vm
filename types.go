package vm

type (
	// Symbol is a string without whitspaces
	Symbol string

	// ValueType is the enumeration for values
	ValueType byte
)

const (
	// Undefined type
	Undefined = ValueType(iota)
	// List type
	List
	// Integer Type
	Integer
	// Double Type
	Double
	// String Type
	String
	// Sym type
	Sym
	// Blob type
	Blob
)
