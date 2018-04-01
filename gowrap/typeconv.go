package fproto_gowrap

import "github.com/RangelReale/fdep"

type TCID string

type TypeConverterPlugin interface {
	// Returns a type converter for the type
	GetTypeConverter(tp *fdep.DepType) TypeConverter
}

type TypeConverter interface {
	TypeNamer

	// Returns an UUID string uniquelly identifying this type converter (without {})
	TCID() TCID

	// Generates code to import the type from the Go protobuf generated type
	GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error)

	// Generates code to export the type to the Go protobuf generated type
	GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error)
}
