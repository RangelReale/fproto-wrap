package fproto_gowrap

import (
	"fmt"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto/fdep"
)

const (
	TCID_DEFAULT string = "d7365856-bd04-413e-976d-350998cc1e7d"
	TCID_SCALAR  string = "cb67c193-7b51-4392-baa2-3c92ba6015e6"
)

// Default type converter
type TypeConverter_Default struct {
	g         *Generator
	tp        *fdep.DepType
	filedep   *fdep.FileDep
	is_gowrap bool
}

func (t *TypeConverter_Default) TCID() string {
	return TCID_DEFAULT
}

func (t *TypeConverter_Default) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
	ret := ""

	switch tntype {
	case TNT_TYPENAME, TNT_POINTER:
		if t.tp.IsPointer() {
			ret += "*"
		}
	case TNT_FIELD_DEFINITION:
		if (g.G().Syntax() == GeneratorSyntax_Proto2 && t.tp.CanPointer()) || t.tp.IsPointer() {
			ret += "*"
		}
	case TNT_EMPTYVALUE:
		if t.tp.IsPointer() {
			ret += "&"
		}
	case TNT_EMPTYORNILVALUE:
		if t.tp.IsPointer() {
			return "nil"
		}
	}

	// get Go type name
	goTypeName, _, _ := g.G().BuildTypeName(t.tp)

	if t.is_gowrap && t.tp.FileDep.IsSamePackage(t.filedep) {
		ret += fmt.Sprintf("%s", goTypeName)
	} else {
		falias := g.FileDep(t.tp.FileDep, t.tp.Alias, t.is_gowrap)
		ret += fmt.Sprintf("%s.%s", falias, goTypeName)
	}

	switch tntype {
	case TNT_EMPTYVALUE:
		if t.tp.IsPointer() {
			ret += "{}"
		}
	}

	return ret
}

func (t *TypeConverter_Default) IsPointer() bool {
	return t.tp.IsPointer()
}

func (t *TypeConverter_Default) GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	if !g.G().IsFileGowrap(t.tp.FileDep) {
		g.P(varDest, " = ", varSrc)
		return false, nil
	}

	var falias string
	if !t.is_gowrap || !t.tp.FileDep.IsSamePackage(t.filedep) {
		falias = g.FileDep(t.tp.FileDep, t.tp.Alias, t.is_gowrap) + "."
	}

	switch t.tp.Item.(type) {
	case *fproto.EnumElement:
		g.P(varDest, " = ", varSrc)
		return false, nil
	}

	// get Go type name
	goTypeName, _, _ := g.G().BuildTypeName(t.tp)

	// varDest, err = goalias.MyStruct_Import(varSrc)
	g.P(varDest, ", err = ", falias, goTypeName, "_Import(", varSrc, ")")

	return true, nil
}

func (t *TypeConverter_Default) GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	if !g.G().IsFileGowrap(t.tp.FileDep) {
		g.P(varDest, " = ", varSrc)
		return false, nil
	}

	switch t.tp.Item.(type) {
	case *fproto.EnumElement:
		g.P(varDest, " = ", varSrc)
		return false, nil
	}

	// varDest, err = MyStruct.Export()
	g.P(varDest, ", err = ", varSrc, ".Export()")
	return true, nil
}

// Type converter for scalar fields
type TypeConverter_Scalar struct {
	tp      *fdep.DepType
	fldtype string
}

func (t *TypeConverter_Scalar) TCID() string {
	return TCID_SCALAR
}

func (t *TypeConverter_Scalar) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
	var ret string

	switch tntype {
	case TNT_FIELD_DEFINITION:
		if g.G().Syntax() == GeneratorSyntax_Proto2 && t.tp.CanPointer() {
			ret += "*"
		}
	}

	if ft, ok := fproto.ParseScalarType(t.fldtype); ok {
		return ret + ft.GoType()
	}

	return ret + t.fldtype
}

func (t *TypeConverter_Scalar) IsPointer() bool {
	return false
}

func (t *TypeConverter_Scalar) GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	// just assign
	g.P(varDest, " = ", varSrc)
	return false, nil
}

func (t *TypeConverter_Scalar) GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	// just assign
	g.P(varDest, " = ", varSrc)
	return false, nil
}
