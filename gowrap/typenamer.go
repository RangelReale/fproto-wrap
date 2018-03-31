package fproto_gowrap

type TypeNamer interface {
	// Gets the type name in relation to the current file
	TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string

	// Returns if the underlining type is a pointer
	IsPointer() bool
}
