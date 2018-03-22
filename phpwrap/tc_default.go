package fproto_phpwrap

import (
	"github.com/RangelReale/fproto/fdep"
)

const (
	TCID_DEFAULT string = "d7ac6dec-bb7c-48eb-8515-626b94ef8ad3"
	TCID_SCALAR  string = "10cddb9d-e263-4074-afdf-3505b57fc4c8"
)

// Default type converter
type TypeConverter_Default struct {
	g       *Generator
	tp      *fdep.DepType
	filedep *fdep.FileDep
}

func (t *TypeConverter_Default) TCID() string {
	return TCID_DEFAULT
}

func (t *TypeConverter_Default) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
	switch tntype {
	case TNT_NS_WRAPNAME, TNT_NS_SOURCENAME:
		sourceFieldTypeName, wrapFieldTypeName := g.G().BuildTypeNSName(t.tp)
		if tntype == TNT_NS_SOURCENAME {
			return sourceFieldTypeName
		} else {
			return wrapFieldTypeName
		}
	}

	typeName, _ := g.G().BuildTypeName(t.tp)
	return typeName
}

func (t *TypeConverter_Default) IsScalar() bool {
	return false
}

func (t *TypeConverter_Default) GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	if !g.G().IsFileWrap(t.tp.FileDep) || !t.tp.IsPointer() || t.tp.FileDep.DepType != fdep.DepType_Own {
		g.P(varDest, " = ", varSrc, ";")
		return false, nil
	}

	// convert field value
	_, wrapFieldTypeName := g.G().BuildTypeNSName(t.tp)

	g.P(varDest, " = new ", wrapFieldTypeName, "();")
	g.P(varDest, "->import(", varSrc, ");")

	return true, nil
}
func (t *TypeConverter_Default) GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	if !g.G().IsFileWrap(t.tp.FileDep) || !t.tp.IsPointer() || t.tp.FileDep.DepType != fdep.DepType_Own {
		g.P(varDest, " = ", varSrc, ";")
		return false, nil
	}

	g.P(varDest, " = ", varSrc, "->export();")

	return true, nil
}

// Type converter for scalar fields
type TypeConverter_Scalar struct {
	tp *fdep.DepType
}

func (t *TypeConverter_Scalar) TCID() string {
	return TCID_SCALAR
}

func (t *TypeConverter_Scalar) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
	return ScalarToPhp(*t.tp.ScalarType)
}

func (t *TypeConverter_Scalar) IsScalar() bool {
	return true
}

func (t *TypeConverter_Scalar) GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	return false, nil

	// just assign
	g.P(varDest, " = ", varSrc, ";")
	return false, nil
}

func (t *TypeConverter_Scalar) GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	return false, nil

	// just assign
	g.P(varDest, " = ", varSrc, ";")
	return false, nil
}
