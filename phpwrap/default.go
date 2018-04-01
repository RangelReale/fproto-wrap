package fproto_phpwrap

import (
	"os"
	"path/filepath"

	"github.com/RangelReale/fdep"
)

//
// TypeNamer: Source
//

type TypeNamer_Source struct {
	g *Generator
	//  The type of the source data
	tp *fdep.DepType
	// The file where the type is in relation to
	depfile *fdep.DepFile
}

func (t *TypeNamer_Source) TypeName(g *GeneratorFile, tntype TypeNameType) string {
	switch tntype {
	case TNT_NS_TYPENAME:
		sourceFieldTypeName, _ := g.G().BuildTypeNSName(t.tp)
		return sourceFieldTypeName
	}

	typeName, _ := g.G().BuildTypeName(t.tp)
	return typeName
}

func (t *TypeNamer_Source) IsScalar() bool {
	return false
}

//
// TypeNamer: Scalar
//

type TypeNamer_Scalar struct {
	tp *fdep.DepType
}

func (t *TypeNamer_Scalar) TypeName(g *GeneratorFile, tntype TypeNameType) string {
	return ScalarToPhp(*t.tp.ScalarType)
}

func (t *TypeNamer_Scalar) IsScalar() bool {
	return true
}

//
// TypeConverter: Default
//

const (
	TCID_DEFAULT string = "d7ac6dec-bb7c-48eb-8515-626b94ef8ad3"
)

// Default type converter
type TypeConverter_Default struct {
	g *Generator
	//  The type of the source data
	tp *fdep.DepType
	// The file where the type is in relation to
	depfile *fdep.DepFile
}

func (t *TypeConverter_Default) TCID() string {
	return TCID_DEFAULT
}

func (t *TypeConverter_Default) TypeName(g *GeneratorFile, tntype TypeNameType) string {
	switch tntype {
	case TNT_NS_TYPENAME:
		sourceFieldTypeName, wrapFieldTypeName := g.G().BuildTypeNSName(t.tp)
		if !g.G().IsWrap(t.tp) {
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

func (t *TypeConverter_Default) GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	if !g.G().IsWrap(t.tp) {
		return false, nil
	}

	// convert field value
	_, wrapFieldTypeName := g.G().BuildTypeNSName(t.tp)

	g.P(varDest, " = new ", wrapFieldTypeName, "();")
	g.P(varDest, "->import(", varSrc, ");")

	return true, nil
}
func (t *TypeConverter_Default) GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	if !g.G().IsWrap(t.tp) {
		return false, nil
	}

	g.P(varDest, " = ", varSrc, "->export();")

	return true, nil
}

//
// TypeConverter: Scalar
//

const (
	TCID_SCALAR string = "10cddb9d-e263-4074-afdf-3505b57fc4c8"
)

// Type converter for scalar fields
type TypeConverter_Scalar struct {
	tp *fdep.DepType
}

func (t *TypeConverter_Scalar) TCID() string {
	return TCID_SCALAR
}

func (t *TypeConverter_Scalar) TypeName(g *GeneratorFile, tntype TypeNameType) string {
	return ScalarToPhp(*t.tp.ScalarType)
}

func (t *TypeConverter_Scalar) IsScalar() bool {
	return true
}

func (t *TypeConverter_Scalar) GenerateImport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	return false, nil
}

func (t *TypeConverter_Scalar) GenerateExport(g *GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
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
// FileOutput: Default
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
