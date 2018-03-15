package fproto_gowrap

import (
	"fmt"
	"strings"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto/fdep"
)

// Default type converter
type TypeConverter_Default struct {
	g         *Generator
	tp        *fdep.DepType
	filedep   *fdep.FileDep
	is_gowrap bool
}

func (t *TypeConverter_Default) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
	ret := ""

	switch tntype {
	case TNT_TYPENAME, TNT_POINTER:
		if t.tp.IsPointer() {
			ret += "*"
		}
	case TNT_FIELD_DEFINITION:
		if g.G().Syntax() == GeneratorSyntax_Proto2 || t.tp.IsPointer() {
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

	if t.is_gowrap && t.tp.FileDep.IsSamePackage(t.filedep) {
		//_ = g.FileDep(tp.FileDep, tp.Alias)
		ret += fmt.Sprintf("%s", strings.Replace(t.tp.Name, ".", "_", -1))
	} else {
		falias := g.FileDep(t.tp.FileDep, t.tp.Alias, t.is_gowrap)
		ret += fmt.Sprintf("%s.%s", falias, strings.Replace(t.tp.Name, ".", "_", -1))
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

	// varDest, err = MyStruct_Import(varSrc)
	g.P(varDest, ", err = ", falias, strings.Replace(t.tp.Name, ".", "_", -1), "_Import(", varSrc, ")")

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
	fldtype string
}

func (t *TypeConverter_Scalar) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
	var ret string

	switch tntype {
	case TNT_FIELD_DEFINITION:
		if g.G().Syntax() == GeneratorSyntax_Proto2 {
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
