package fproto_gowrap

type TypeNameType int

const (
	TNT_TYPENAME TypeNameType = iota
	TNT_FIELD_DEFINITION
	TNT_EMPTYVALUE
	TNT_EMPTYORNILVALUE
	TNT_POINTER
)

// TypeName options
const (
	TNO_FORCE_USE_PACKAGE uint32 = 0x01
)

type TypeNamer interface {
	// Gets the type name in relation to the current file
	TypeName(g *GeneratorFile, tntype TypeNameType, options uint32) string

	// Returns if the underlining type is a pointer
	IsPointer() bool
}
