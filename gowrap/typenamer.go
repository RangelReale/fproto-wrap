package fproto_gowrap

import (
	"fmt"

	"github.com/RangelReale/fdep"
)

type TypeNamer interface {
	// Gets the type name in relation to the current file
	TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string

	// Returns if the underlining type is a pointer
	IsPointer() bool
}

type TypeNamer_Source struct {
	g       *Generator
	tp      *fdep.DepType
	filedep *fdep.FileDep
}

// Gets the type name in relation to the current file
func (t *TypeNamer_Source) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
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

	falias := g.FileDep(t.tp.FileDep, t.tp.Alias, false)
	ret += fmt.Sprintf("%s.%s", falias, goTypeName)

	switch tntype {
	case TNT_EMPTYVALUE:
		if t.tp.IsPointer() {
			ret += "{}"
		}
	}

	return ret
}

// Returns if the underlining type is a pointer
func (t *TypeNamer_Source) IsPointer() bool {
	return t.tp.IsPointer()
}

type TypeNamer_Scalar struct {
	tp *fdep.DepType
}

// Gets the type name in relation to the current file
func (t *TypeNamer_Scalar) TypeName(g *GeneratorFile, tntype TypeConverterTypeNameType) string {
	var ret string

	switch tntype {
	case TNT_FIELD_DEFINITION:
		if g.G().Syntax() == GeneratorSyntax_Proto2 && t.tp.CanPointer() {
			ret += "*"
		}
	}

	return ret + t.tp.ScalarType.GoType()
}

// Returns if the underlining type is a pointer
func (t *TypeNamer_Scalar) IsPointer() bool {
	return false
}
