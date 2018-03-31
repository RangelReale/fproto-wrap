package fproto_phpwrap

type TypeNameType int

const (
	TNT_TYPENAME TypeNameType = iota
	TNT_NS_TYPENAME

	//TNT_NS_SOURCENAME
	//TNT_NS_WRAPNAME
)

type TypeNamer interface {
	// Gets the type name in relation to the current file
	TypeName(g *GeneratorFile, tntype TypeNameType) string

	// Returns if the underlining type is scalar
	IsScalar() bool
}
