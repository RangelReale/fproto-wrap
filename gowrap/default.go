package fproto_gowrap

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto"
)

//
// TypeNamer: Source
//

type TypeNamer_Source struct {
	g *Generator
	//  The type of the source data
	tp *fdep.DepType
	// The file where the type is in relation to
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
	goTypeName, _ := g.G().BuildTypeName(t.tp)

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

//
// TypeNamer: Scalar
//

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

//
// TypeConverter: Default
//

const (
	TCID_DEFAULT string = "d7365856-bd04-413e-976d-350998cc1e7d"
)

// Default type converter
type TypeConverter_Default struct {
	g *Generator
	//  The type of the source data
	tp *fdep.DepType
	// The file where the type is in relation to
	filedep *fdep.FileDep
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
	goTypeName, _ := g.G().BuildTypeName(t.tp)

	if t.tp.FileDep.IsSamePackage(t.filedep) {
		ret += fmt.Sprintf("%s", goTypeName)
	} else {
		falias := g.FileDep(t.tp.FileDep, t.tp.Alias, true)
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
	if !g.G().IsFileWrap(t.tp.FileDep) {
		g.P(varDest, " = ", varSrc)
		return false, nil
	}

	var falias string
	if !t.tp.FileDep.IsSamePackage(t.filedep) {
		falias = g.FileDep(t.tp.FileDep, t.tp.Alias, true) + "."
	}

	switch t.tp.Item.(type) {
	case *fproto.EnumElement:
		g.P(varDest, " = ", varSrc)
		return false, nil
	}

	// get Go type name
	goTypeName, _ := g.G().BuildTypeName(t.tp)

	// varDest, err = goalias.MyStruct_Import(varSrc)
	g.P(varDest, ", err = ", falias, goTypeName, "_Import(", varSrc, ")")

	return true, nil
}

func (t *TypeConverter_Default) GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	if !g.G().IsFileWrap(t.tp.FileDep) {
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

//
// TypeConverter: Scalar
//

const (
	TCID_SCALAR string = "cb67c193-7b51-4392-baa2-3c92ba6015e6"
)

// Type converter for scalar fields
type TypeConverter_Scalar struct {
	tp *fdep.DepType
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

	return ret + t.tp.ScalarType.GoType()
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

//
// TypeInfo: Default
//

type TypeInfo_Default struct {
	source    TypeNamer
	converter TypeConverter
}

func (t *TypeInfo_Default) Source() TypeNamer {
	return t.source
}

func (t *TypeInfo_Default) Converter() TypeConverter {
	return t.converter
}

//
// Customizer: wrap
//

// Wraps a list of customizers
type wrapCustomizers struct {
	customizers []Customizer
}

func (c *wrapCustomizers) GetTag(g *Generator, currentTag *StructTag, parentItem fproto.FProtoElement, item fproto.FProtoElement) error {
	for _, cz := range c.customizers {
		if ct, ok := cz.(Customizer_Tag); ok {
			err := ct.GetTag(g, currentTag, parentItem, item)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *wrapCustomizers) GenerateCode(g *Generator) error {
	for _, cz := range c.customizers {
		err := cz.GenerateCode(g)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *wrapCustomizers) GenerateServiceCode(g *Generator) error {
	for _, cz := range c.customizers {
		err := cz.GenerateServiceCode(g)
		if err != nil {
			return err
		}
	}
	return nil
}

//
// FileOutput: default
//

type FileOutput_Default struct {
	OutputPath string
}

func NewFileOutput_Default(outputPath string) *FileOutput_Default {
	return &FileOutput_Default{
		OutputPath: outputPath,
	}
}

func (f *FileOutput_Default) Initialize() error {
	return nil
}

func (f *FileOutput_Default) Finalize() error {
	return nil
}

func (f *FileOutput_Default) Output(g *GeneratorFile) error {
	p := filepath.Join(f.OutputPath, g.Filename())

	// create paths
	err := os.MkdirAll(filepath.Dir(p), os.ModePerm)
	if err != nil {
		return err
	}

	// create file
	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer file.Close()

	// output contents
	err = g.Output(file)
	if err != nil {
		return err
	}

	return nil
}
