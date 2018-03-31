package fproto_gowrap

type TypeNameType int

const (
	TNT_TYPENAME TypeNameType = iota
	TNT_FIELD_DEFINITION
	TNT_EMPTYVALUE
	TNT_EMPTYORNILVALUE
	TNT_POINTER
)

type TypeNamer interface {
	// Gets the type name in relation to the current file
	TypeName(g *GeneratorFile, tntype TypeNameType) string

	// Returns if the underlining type is a pointer
	IsPointer() bool
}
