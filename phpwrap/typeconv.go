package fproto_phpwrap

import "github.com/RangelReale/fproto/fdep"

type TypeConverterTypeNameType int

const (
	TNT_TYPENAME TypeConverterTypeNameType = iota
	TNT_FIELD_DEFINITION
	TNT_EMPTYVALUE
	TNT_EMPTYORNILVALUE
	TNT_POINTER
)

type TypeConverterPlugin interface {
	// Returns a type converter for the type
	GetTypeConverter(tp *fdep.DepType) TypeConverter
}

type TypeConverter interface {
	// Returns an UUID string uniquelly identifying this type converter (without {})
	TCID() string

	// Gets the type name in relation to the current file
	TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string

	// Returns if the underlining type is scalar
	IsScalar() bool

	// Generates code to import the type from the Go protobuf generated type
	GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error)

	// Generates code to export the type to the Go protobuf generated type
	GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error)
}
