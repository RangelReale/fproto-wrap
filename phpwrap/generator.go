package fproto_phpwrap

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto-wrap"
)

type GeneratorSyntax int

const (
	GeneratorSyntax_Proto2 GeneratorSyntax = iota
	GeneratorSyntax_Proto3
)

// Generators generates a wrapper for a single source file.
// There can be more than one output files.
type Generator struct {
	dep        *fdep.Dep
	depfile    *fdep.DepFile
	tc_default TypeConverter

	// Files to output
	Files map[string]*GeneratorFile
	//FilesAlias map[string]string

	// Interface to do namespace name generation.
	NSSource NSSource

	// List of type conversions
	TypeConverters []TypeConverterPlugin

	// Service generator
	ServiceGen ServiceGen

	// Customizers
	Customizers []Customizer
}

// Creates a new generator for the file path.
func NewGenerator(dep *fdep.Dep, filepath string) (*Generator, error) {
	depfile, ok := dep.Files[filepath]
	if !ok {
		return nil, fmt.Errorf("File %s not found", filepath)
	}

	ret := &Generator{
		dep:     dep,
		depfile: depfile,
		Files:   make(map[string]*GeneratorFile),
		//FilesAlias: make(map[string]string),
	}

	return ret, nil
}

// Creates a new file
func (g *Generator) SetFile(fileId string) {
	g.Files[fileId] = NewGeneratorFile(g, fileId)
}

// Gets a file by id
func (g *Generator) F(fileId string) *GeneratorFile {
	if gf, ok := g.Files[fileId]; ok {
		return gf
	}

	panic(fmt.Sprintf("Generator file id %s not found", fileId))
	return nil
}

// Gets the syntax
func (g *Generator) Syntax() GeneratorSyntax {
	if g.depfile.ProtoFile.Syntax == "proto3" {
		return GeneratorSyntax_Proto3
	}
	return GeneratorSyntax_Proto2
}

func (g *Generator) GetDep() *fdep.Dep {
	return g.dep
}

func (g *Generator) GetDepFile() *fdep.DepFile {
	return g.depfile
}

// Check if the file should be wrapped (the file option fproto_wrap.wrap=false disables it)
func (g *Generator) IsFileWrap(depfile *fdep.DepFile) bool {
	if depfile.DepType != fdep.DepType_Own {
		return false
	}

	if o := depfile.ProtoFile.FindOption("fproto_wrap.wrap"); o != nil {
		if o.Value.String() != "true" {
			return false
		}
	}

	return true
}

// Builds a PHP namespaced name array (split by ".", and each item with first character uppercased)
func (g *Generator) BuildPHPNamespacedNameArray(name string) []string {
	paths := strings.Split(name, ".")

	var retpaths []string
	for _, p := range paths {
		retpaths = append(retpaths, fproto_wrap.UCFirst(p))
	}
	return retpaths
}

// Builds a PHP namespaced name (split by ".", each item with first character uppercased, separated by \)
func (g *Generator) BuildPHPNamespacedName(name string) string {
	return strings.Join(g.BuildPHPNamespacedNameArray(name), "\\")
}

// Return an array of scopes of the element, NOT including the element itself
func (g *Generator) GetScope(element fproto.FProtoElement) []string {
	var ret []string
	isfirst := true
	cur := element
	for cur != nil {
		switch el := cur.(type) {
		case *fproto.MessageElement:
			if !isfirst {
				ret = append(ret, el.Name)
			}
			cur = el.Parent
		case *fproto.EnumElement:
			if !isfirst {
				ret = append(ret, el.Name)
			}
			cur = el.Parent
		case *fproto.OneOfFieldElement:
			if !isfirst {
				ret = append(ret, el.Name)
			}
			cur = el.Parent
		case *fproto.EnumConstantElement:
			if !isfirst {
				ret = append(ret, el.Name)
			}
			cur = el.Parent
		case *fproto.FieldElement:
			// don't add to list
			cur = el.Parent
		case *fproto.MapFieldElement:
			// don't add to list
			cur = el.Parent
		default:
			cur = nil
		}
		isfirst = false
	}

	// reverse order
	if ret != nil {
		return fproto.ReverseStr(ret)
	}

	return ret
}

// Executes the generator
func (g *Generator) Generate() error {
	// CUSTOMIZER
	cz := &wrapCustomizers{g.Customizers}

	err := g.GenerateEnums()
	if err != nil {
		return err
	}

	err = g.GenerateMessages()
	if err != nil {
		return err
	}

	// CUSTOMIZER
	err = cz.GenerateCode(g)
	if err != nil {
		return err
	}

	err = g.GenerateServices()
	if err != nil {
		return err
	}

	// CUSTOMIZER
	err = cz.GenerateServiceCode(g)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) BuildTypeName(dt *fdep.DepType) (phpName string, protoName string) {
	if dt.IsScalar() {
		return ScalarToPhp(*dt.ScalarType), ScalarToPhp(*dt.ScalarType)
	}

	if dt.Item != nil {
		switch item := dt.Item.(type) {
		case *fproto.MessageElement:
			phpName, protoName = g.BuildMessageName(item)
			return
		case *fproto.EnumElement:
			phpName, protoName = g.BuildEnumName(item)
			return
			/*
				case *fproto.OneOfFieldElement:
					goName, protoName, protoScope = g.BuildOneOfName(item)
					return
				case fproto.FieldElementTag:
					// if the parent is a oneof, call a different function
					switch item.ParentElement().(type) {
					case *fproto.EnumElement:
						goName, protoName, protoScope = g.BuildOneOfFieldName(item)
					default:
						goName, protoName = g.BuildFieldName(item)
						protoScope = ""
					}
			*/
			return
		}
	}

	// Fallback
	return fproto_wrap.CamelCaseProto(dt.Name), dt.Name
}

func (g *Generator) BuildTypeNSName(dt *fdep.DepType) (sourceName string, wrapName string) {
	if dt.IsScalar() {
		return ScalarToPhp(*dt.ScalarType), ScalarToPhp(*dt.ScalarType)
	}

	if dt.Item != nil {
		switch dt.Item.(type) {
		case *fproto.MessageElement:
			sourceName, wrapName = g.BuildMessageNSName(dt)
			return
		case *fproto.EnumElement:
			sourceName, wrapName = g.BuildEnumNSName(dt)
			return
			/*
				case *fproto.OneOfFieldElement:
					goName, protoName, protoScope = g.BuildOneOfName(item)
					return
				case fproto.FieldElementTag:
					// if the parent is a oneof, call a different function
					switch item.ParentElement().(type) {
					case *fproto.EnumElement:
						goName, protoName, protoScope = g.BuildOneOfFieldName(item)
					default:
						goName, protoName = g.BuildFieldName(item)
						protoScope = ""
					}
			*/
			return
		}
	}

	// Fallback
	return fproto_wrap.CamelCaseProto(dt.Name), fproto_wrap.CamelCaseProto(dt.Name)
}

func (g *Generator) BuildEnumName(enum *fproto.EnumElement) (phpName string, protoName string) {
	// get the dep type
	tp_enum := g.dep.DepTypeFromElement(enum)
	if tp_enum == nil {
		panic("enum type not found")
	}

	// Camel-cased name, with "." replaced by "_"
	phpName = fproto_wrap.CamelCaseProto(tp_enum.Name)

	protoName = tp_enum.Name

	return
}

// Build message namespaced name
func (g *Generator) BuildEnumNSName(tp *fdep.DepType) (sourceName string, wrapName string) {
	sourceNS, wrapNS, _ := g.PhpWrapNS(tp.DepFile)

	// Camel-cased name, with "." replaced by "_"
	phpName := fproto_wrap.CamelCaseProto(tp.Name)

	sourceName = fmt.Sprintf("\\%s\\%s", sourceNS, phpName)
	wrapName = fmt.Sprintf("\\%s\\%s", wrapNS, phpName)
	return
}

func (g *Generator) GenerateEnums() error {
	for _, s := range g.depfile.ProtoFile.CollectEnums() {
		err := g.GenerateEnum(s.(*fproto.EnumElement))
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) GenerateEnum(enum *fproto.EnumElement) error {
	sourceNS, _, wrapPath := g.PhpWrapNS(g.depfile)
	enPhpName, enProtoName := g.BuildEnumName(enum)
	fileId := path.Join(wrapPath, enPhpName)

	g.SetFile(fileId)

	gf := g.F(fileId)

	// class Enum extends \SourceNs\Enum

	if !gf.GenerateComment(enum.Comment, nil) {
		gf.GenerateCommentLine("ENUM: ", enProtoName)
	}

	// only inherit from the source class
	gf.P("class ", enPhpName, " extends \\", sourceNS, "\\", enPhpName)
	gf.P("{")
	gf.In()

	gf.Out()
	gf.P("}")

	return nil
}

func (g *Generator) GenerateMessages() error {
	for _, s := range g.depfile.ProtoFile.CollectMessages() {
		err := g.GenerateMessage(s.(*fproto.MessageElement))
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) BuildMessageName(message *fproto.MessageElement) (phpName string, protoName string) {
	// get the dep type
	tp_message := g.dep.DepTypeFromElement(message)
	if tp_message == nil {
		panic("message type not found")
	}

	// Camel-cased name, with "." replaced by "_"
	phpName = fproto_wrap.CamelCaseProto(tp_message.Name)

	protoName = tp_message.Name

	return
}

// Build message namespaced name
func (g *Generator) BuildMessageNSName(tp *fdep.DepType) (sourceName string, wrapName string) {
	sourceNS, wrapNS, _ := g.PhpWrapNS(tp.DepFile)

	// Camel-cased name, with "." replaced by "_"
	phpName := fproto_wrap.CamelCaseProto(tp.Name)

	sourceName = fmt.Sprintf("\\%s\\%s", sourceNS, phpName)
	wrapName = fmt.Sprintf("\\%s\\%s", wrapNS, phpName)
	return
}

func (g *Generator) BuildFieldName(field fproto.FieldElementTag) (fieldname string, fieldGetter string, fieldSetter string) {
	fieldname = fproto_wrap.SnakeCase(field.ElementName())
	fieldGetter = "get" + fproto_wrap.CamelCase(field.ElementName())
	fieldSetter = "set" + fproto_wrap.CamelCase(field.ElementName())
	return
}

func (g *Generator) GenerateMessage(message *fproto.MessageElement) error {
	if message.IsExtend {
		return nil
	}

	tp_msg := g.dep.DepTypeFromElement(message)
	if tp_msg == nil {
		return errors.New("message type not found")
	}

	sourceNS, _, wrapPath := g.PhpWrapNS(g.depfile)
	msPhpName, msProtoName := g.BuildMessageName(message)
	fileId := path.Join(wrapPath, msPhpName)

	g.SetFile(fileId)

	gf := g.F(fileId)

	// class Message

	if !gf.GenerateComment(message.Comment, nil) {
		gf.GenerateCommentLine("MESSAGE: ", msProtoName)
	}

	gf.P("class ", msPhpName)
	gf.P("{")
	gf.In()

	// private fields
	for _, fld := range message.Fields {
		fldname, _, _ := g.BuildFieldName(fld)

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// Get field type
			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}

			wrapFieldTypeName := tinfo.Converter().TypeName(gf, TNT_NS_TYPENAME)

			gf.GenerateFieldComment(fld, []string{
				fmt.Sprintf("@var %s", wrapFieldTypeName),
			})

			gf.P("private $", fldname, " = null;")
		case *fproto.MapFieldElement:
			// Get field type
			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}

			tinfokey, err := g.GetTypeInfoFromParent(tp_msg, xfld.KeyType)
			if err != nil {
				return err
			}

			wrapFieldTypeName := tinfo.Converter().TypeName(gf, TNT_NS_TYPENAME)
			wrapKeyFieldTypeName := tinfokey.Converter().TypeName(gf, TNT_NS_TYPENAME)

			gf.GenerateFieldComment(fld, []string{
				fmt.Sprintf("@var %s[] key => %s", wrapFieldTypeName, wrapKeyFieldTypeName),
			})

			gf.P("private $", fldname, " = null;")
		case *fproto.OneOfFieldElement:
			gf.GenerateFieldComment(fld, []string{
				"@var mixed oneof",
			})

			gf.P("private $", fldname, " = null;")

			// each oneof field have a variable
			for _, oofld := range xfld.Fields {
				oofldname, _, _ := g.BuildFieldName(oofld)

				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					// Get field type
					ootinfo, err := g.GetTypeInfoFromParent(tp_msg, xoofld.Type)
					if err != nil {
						return err
					}

					wrapOOFieldTypeName := ootinfo.Converter().TypeName(gf, TNT_NS_TYPENAME)

					if xoofld.Repeated {
						wrapOOFieldTypeName += "[]"
					}

					gf.GenerateFieldComment(oofld, []string{
						fmt.Sprintf("@var %s", wrapOOFieldTypeName),
					})

					gf.P("private $", oofldname, " = null;")
				case *fproto.MapFieldElement:
					// Get field type
					ootinfo, err := g.GetTypeInfoFromParent(tp_msg, xoofld.Type)
					if err != nil {
						return err
					}
					ootinfokey, err := g.GetTypeInfoFromParent(tp_msg, xoofld.KeyType)
					if err != nil {
						return err
					}

					wrapOOFieldTypeName := ootinfo.Converter().TypeName(gf, TNT_NS_TYPENAME)
					wrapOOKeyFieldTypeName := ootinfokey.Converter().TypeName(gf, TNT_NS_TYPENAME)

					gf.GenerateFieldComment(oofld, []string{
						fmt.Sprintf("@var %s[] key => %s", wrapOOFieldTypeName, wrapOOKeyFieldTypeName),
					})
				default:
					gf.GenerateFieldComment(oofld, nil)

					gf.P("private $", oofldname, " = null;")
				}
			}
		}
	}

	gf.P()

	// constructor
	gf.P("public function __construct($values = null)")
	gf.P("{")
	gf.In()

	gf.P("$this->importValues($values);")

	gf.Out()
	gf.P("}")

	gf.P()

	// field getters and setters
	for _, fld := range message.Fields {
		err := g.generateFieldGetterSetter(gf, tp_msg, fld)
		if err != nil {
			return err
		}
	}

	//
	// IMPORTER
	//

	gf.GenerateComment(nil, []string{
		fmt.Sprintf("@param \\%s\\%s %s", sourceNS, msPhpName, "$source"),
	})

	// public function import(\SourceNamespace\Message $source)
	gf.P("public function import(\\", sourceNS, "\\", msPhpName, " $source)")
	gf.P("{")
	gf.In()

	for _, fld := range message.Fields {
		err := g.generateFieldImport(gf, tp_msg, fld)
		if err != nil {
			return err
		}
	}

	gf.Out()
	gf.P("}")

	gf.P()

	//
	// EXPORTER
	//

	gf.GenerateComment(nil, []string{
		fmt.Sprintf("@return \\%s\\%s", sourceNS, msPhpName),
	})

	// public function export()
	gf.P("public function export()")
	gf.P("{")
	gf.In()

	gf.P("$ret = new \\", sourceNS, "\\", msPhpName, "();")

	for _, fld := range message.Fields {
		err := g.generateFieldExport(gf, tp_msg, fld)
		if err != nil {
			return err
		}
	}

	gf.P("return $ret;")

	gf.Out()
	gf.P("}")

	gf.P()

	//
	// IMPORT VALUES
	//

	// public function importValues($values)
	gf.P("public function importValues($values)")
	gf.P("{")
	gf.In()

	gf.P("if ($values === null) return;")
	gf.P()

	gf.P("foreach ($values as $vname => $vvalue) {")
	gf.In()

	for fidx, fld := range message.Fields {
		fldname, _, fldsetter := g.BuildFieldName(fld)

		pprefix := ""
		if fidx > 0 {
			pprefix = "} else "
		}

		switch xfld := fld.(type) {
		case *fproto.FieldElement, *fproto.MapFieldElement:
			gf.P(pprefix, "if ($vname == '", fldname, "') {")
			gf.In()
			gf.P("$this->", fldsetter, "($vvalue);")
			gf.Out()
		case *fproto.OneOfFieldElement:
			for _, oofld := range xfld.Fields {
				oofldname, _, oofldsetter := g.BuildFieldName(oofld)

				gf.P(pprefix, "if ($vname == '", oofldname, "') {")
				gf.In()
				gf.P("$this->", oofldsetter, "($vvalue);")
				gf.Out()
			}
		}

	}

	if len(message.Fields) > 0 {
		gf.P("} else {")
		gf.In()
	}
	gf.P("throw new \\Exception(\"Param '.$vname.' doesn't exists\");")
	if len(message.Fields) > 0 {
		gf.Out()
		gf.P("}")
	}

	gf.Out()
	gf.P("}")

	gf.Out()
	gf.P("}")

	gf.P()

	//
	// EXPORT VALUES
	//

	// public function exportValues()

	gf.P("public function exportValues()")
	gf.P("{")
	gf.In()

	gf.P("$ret = [];")

	for _, fld := range message.Fields {
		fldname, fldgetter, _ := g.BuildFieldName(fld)

		switch xfld := fld.(type) {
		case *fproto.FieldElement, *fproto.MapFieldElement:
			gf.P("$ret['", fldname, "'] = $this->", fldgetter, "();")
		case *fproto.OneOfFieldElement:
			gf.P("switch ($this->", fldname, ") {")

			for _, oofld := range xfld.Fields {
				oofldname, oofldgetter, _ := g.BuildFieldName(oofld)

				gf.P("case '", oofldname, "':")
				gf.In()
				gf.P("$ret['", oofldname, "'] = $this->", oofldgetter, "();")
				gf.Out()
			}

			gf.P("}")
		}
	}

	gf.P("return $ret;")

	gf.Out()
	gf.P("}")

	gf.P()

	// end class
	gf.Out()
	gf.P("}")

	return nil
}

func (g *Generator) generateFieldGetterSetter(gf *GeneratorFile, parent_type *fdep.DepType, fld fproto.FieldElementTag) error {
	fldname, fldgetter, fldsetter := g.BuildFieldName(fld)

	switch xfld := fld.(type) {
	case *fproto.FieldElement:
		// Get field type
		tinfo, err := g.GetTypeInfoFromParent(parent_type, xfld.Type)
		if err != nil {
			return err
		}

		wrapFieldTypeName := tinfo.Converter().TypeName(gf, TNT_NS_TYPENAME)
		if xfld.Repeated {
			wrapFieldTypeName += "[]"
		}

		gf.GenerateFieldComment(nil, []string{
			fmt.Sprintf("@return %s", wrapFieldTypeName),
		})

		// public function getField() {
		// 		return $this->field;
		// }
		gf.P("public function ", fldgetter, "() {")
		gf.In()
		gf.P("return $this->", fldname, ";")
		gf.Out()
		gf.P("}")

		gf.P()

		gf.GenerateFieldComment(nil, []string{
			fmt.Sprintf("@param %s $var", wrapFieldTypeName),
		})

		// public function setField($var) {
		// 		$this->field = $var;
		//		return $this;
		// }
		gf.P("public function ", fldsetter, "($var) {")
		gf.In()
		gf.P("$this->", fldname, " = $var;")
		if parent_type.Item != nil && parent_type.Item != nil {
			if oot, isoot := parent_type.Item.(*fproto.OneOfFieldElement); isoot {
				oofldname, _, _ := g.BuildFieldName(oot)

				//		$this->oneoffield = 'field';
				gf.P("$this->", oofldname, " = '", fldname, "';")
			}
		}

		gf.P("return $this;")
		gf.Out()
		gf.P("}")

		gf.P()
	case *fproto.MapFieldElement:
		// Get field type
		tinfo, err := g.GetTypeInfoFromParent(parent_type, xfld.Type)
		if err != nil {
			return err
		}

		tinfokey, err := g.GetTypeInfoFromParent(parent_type, xfld.KeyType)
		if err != nil {
			return err
		}

		wrapFieldTypeName := tinfo.Converter().TypeName(gf, TNT_NS_TYPENAME)
		wrapKeyFieldTypeName := tinfokey.Converter().TypeName(gf, TNT_NS_TYPENAME)

		gf.GenerateFieldComment(fld, []string{
			fmt.Sprintf("@return %s[] key => %s", wrapFieldTypeName, wrapKeyFieldTypeName),
		})

		// public function getField() {
		// 		return $this->field;
		// }
		gf.P("public function ", fldgetter, "() {")
		gf.In()
		gf.P("return $this->", fldname, ";")
		gf.Out()
		gf.P("}")

		gf.P()

		gf.GenerateFieldComment(nil, []string{
			fmt.Sprintf("@param %s[] $var key => %s", wrapFieldTypeName, wrapKeyFieldTypeName),
		})

		// public function setField($var) {
		// 		$this->field = $var;
		//		return $this;
		// }
		gf.P("public function ", fldsetter, "($var) {")
		gf.In()
		gf.P("$this->", fldname, " = $var;")
		if parent_type.Item != nil && parent_type.Item != nil {
			if oot, isoot := parent_type.Item.(*fproto.OneOfFieldElement); isoot {
				oofldname, _, _ := g.BuildFieldName(oot)

				//		$this->oneoffield = 'field';
				gf.P("$this->", oofldname, " = '", fldname, "';")
			}
		}
		gf.P("return $this;")
		gf.Out()
		gf.P("}")

		gf.P()
	case *fproto.OneOfFieldElement:
		// public function getField() {
		// 		return $this->field;
		// }
		gf.P("public function ", fldgetter, "() {")
		gf.In()
		gf.P("return $this->", fldname, ";")
		gf.Out()
		gf.P("}")

		gf.P()

		// each oneof field have a getter and a setter
		for _, oofld := range xfld.Fields {

			tp_oo := g.dep.DepTypeFromElement(xfld)
			if tp_oo == nil {
				return fmt.Errorf("Could not find dep type from oneof %s", xfld.Name)
			}

			err := g.generateFieldGetterSetter(gf, tp_oo, oofld)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (g *Generator) generateFieldImport(gf *GeneratorFile, parent_type *fdep.DepType, fld fproto.FieldElementTag) error {
	fldname, fldgetter, fldsetter := g.BuildFieldName(fld)

	switch xfld := fld.(type) {
	case *fproto.FieldElement:
		// Get field type
		tinfo, err := g.GetTypeInfoFromParent(parent_type, xfld.Type)
		if err != nil {
			return err
		}

		if !tinfo.Converter().IsScalar() {
			gf.P("if ($source->", fldgetter, "() !== null) {")
			gf.In()
		}

		varName := "$" + fldname + "__wrap"

		source_field := "$source->" + fldgetter + "()"
		dest_field := varName

		if xfld.Repeated {
			gf.P(varName, " = [];")

			gf.P("foreach ($source->", fldgetter, "() as $ms) {")
			gf.In()
			source_field = "$ms"
			dest_field = "$msi"
		}

		generated, err := tinfo.Converter().GenerateImport(gf, source_field, dest_field, "error")
		if err != nil {
			return err
		}

		if !generated && !xfld.Repeated {
			// assign directly
			varName = "$source->" + fldgetter + "()"
		}

		if xfld.Repeated {
			if generated {
				gf.P(varName, "[] = $msi;")
			} else {
				gf.P(varName, "[] = $ms;")
			}

			gf.Out()
			gf.P("}")
		}

		gf.P("$this->", fldsetter, "(", varName, ");")

		if !tinfo.Converter().IsScalar() {
			gf.Out()
			gf.P("}")
		}
	case *fproto.MapFieldElement:
		// Get map field type
		tinfo, err := g.GetTypeInfoFromParent(parent_type, xfld.Type)
		if err != nil {
			return err
		}

		if !tinfo.Converter().IsScalar() {
			gf.P("if ($source->", fldgetter, "() !== null) {")
			gf.In()
		}

		varName := "$" + fldname + "__wrapmap"

		gf.P(varName, " = [];")

		gf.P("foreach ($source->", fldgetter, "() as $msidx => $ms) {")
		gf.In()

		generated, err := tinfo.Converter().GenerateImport(gf, "$ms", "$msi", "error")
		if err != nil {
			return err
		}

		if generated {
			gf.P(varName, "[$msidx] = $msi;")
		} else {
			gf.P(varName, "[$msidx] = $ms;")
		}

		gf.Out()
		gf.P("}")

		gf.P("$this->", fldsetter, "(", varName, ");")

		if !tinfo.Converter().IsScalar() {
			gf.Out()
			gf.P("}")
		}

	case *fproto.OneOfFieldElement:
		gf.P("switch ($source->", fldgetter, "()) {")

		tp_oo := g.dep.DepTypeFromElement(xfld)
		if tp_oo == nil {
			return fmt.Errorf("Could not find dep type from oneof %s", xfld.Name)
		}

		for _, oofld := range xfld.Fields {
			oofldname, _, _ := g.BuildFieldName(oofld)

			gf.P("case '", oofldname, "':")
			gf.In()

			err := g.generateFieldImport(gf, tp_oo, oofld)
			if err != nil {
				return err
			}

			gf.P("break;")
			gf.Out()
		}

		gf.P("}")
	}

	return nil
}

func (g *Generator) generateFieldExport(gf *GeneratorFile, parent_type *fdep.DepType, fld fproto.FieldElementTag) error {
	fldname, _, fldsetter := g.BuildFieldName(fld)

	switch xfld := fld.(type) {
	case *fproto.FieldElement:
		tinfo, err := g.GetTypeInfoFromParent(parent_type, xfld.Type)
		if err != nil {
			return err
		}

		if !tinfo.Converter().IsScalar() {
			gf.P("if ($this->", fldname, " !== null) {")
			gf.In()
		}

		varName := "$" + fldname + "__export"

		source_field := "$this->" + fldname
		dest_field := varName

		if xfld.Repeated {
			gf.P(varName, " = [];")

			gf.P("foreach ($this->", fldname, " as $ms) {")
			gf.In()
			source_field = "$ms"
			dest_field = "$msi"
		}

		generated, err := tinfo.Converter().GenerateExport(gf, source_field, dest_field, "error")
		if err != nil {
			return err
		}

		if !generated && !xfld.Repeated {
			// assign directly
			varName = "$this->" + fldname
		}

		if xfld.Repeated {
			if generated {
				gf.P(varName, "[] = $msi;")
			} else {
				gf.P(varName, "[] = $ms;")
			}

			gf.Out()
			gf.P("}")
		}

		gf.P("$ret->", fldsetter, "(", varName, ");")

		if !tinfo.Converter().IsScalar() {
			gf.Out()
			gf.P("}")
		}

	case *fproto.MapFieldElement:
		tinfo, err := g.GetTypeInfoFromParent(parent_type, xfld.Type)
		if err != nil {
			return err
		}

		gf.P("if ($this->", fldname, " !== null) {")
		gf.In()

		varName := "$" + fldname + "__export"

		gf.P(varName, " = [];")

		gf.P("foreach ($this->", fldname, " as $msidx => $ms) {")
		gf.In()

		generated, err := tinfo.Converter().GenerateExport(gf, "$ms", "$msi", "error")
		if err != nil {
			return err
		}

		if generated {
			gf.P(varName, "[$msidx] = $msi;")
		} else {
			gf.P(varName, "[$msidx] = $ms;")
		}

		gf.Out()
		gf.P("}")

		gf.P("$ret->", fldsetter, "(", varName, ");")

		gf.Out()
		gf.P("}")
	case *fproto.OneOfFieldElement:
		gf.P("switch ($this->", fldname, ") {")

		tp_oo := g.dep.DepTypeFromElement(xfld)
		if tp_oo == nil {
			return fmt.Errorf("Could not find dep type from oneof %s", xfld.Name)
		}

		for _, oofld := range xfld.Fields {
			oofldname, _, _ := g.BuildFieldName(oofld)

			gf.P("case '", oofldname, "':")
			gf.In()

			err := g.generateFieldExport(gf, tp_oo, oofld)
			if err != nil {
				return err
			}

			gf.P("break;")
			gf.Out()
		}

		gf.P("}")

	}

	return nil
}

// Generates the protobuf services
func (g *Generator) GenerateServices() error {
	if g.ServiceGen == nil || len(g.depfile.ProtoFile.Services) == 0 {
		return nil
	}

	for _, s := range g.depfile.ProtoFile.CollectServices() {
		err := g.ServiceGen.GenerateService(g, s.(*fproto.ServiceElement))
		if err != nil {
			return err
		}
	}

	return nil
}

// Get type converter for type
func (g *Generator) findTypeConv(tp *fdep.DepType) TypeConverter {
	for _, tcp := range g.TypeConverters {
		tc := tcp.GetTypeConverter(tp)
		if tc != nil {
			return tc
		}
	}
	return nil
}

// Gets the type for the source protoc-gen-go generated names
func (g *Generator) GetTypeSource(tp *fdep.DepType) TypeNamer {
	if tp.IsScalar() {
		return &TypeNamer_Scalar{tp: tp}
	} else {
		return &TypeNamer_Source{tp: tp}
	}
}

// Gets the type for the phpwrap converter
func (g *Generator) GetTypeConverter(tp *fdep.DepType) TypeConverter {
	if tp.IsScalar() {
		return &TypeConverter_Scalar{tp}
	} else {
		if tc := g.findTypeConv(tp); tc != nil {
			return tc
		}
		return &TypeConverter_Default{g, tp, g.depfile}
	}
}

// Get both source and converter types.
func (g *Generator) GetTypeInfo(tp *fdep.DepType) TypeInfo {
	return &TypeInfo_Default{
		source:    g.GetTypeSource(tp),
		converter: g.GetTypeConverter(tp),
	}
}

// Get both source and converter types from a parent and a type name.
func (g *Generator) GetTypeInfoFromParent(parent_tp *fdep.DepType, atype string) (TypeInfo, error) {
	tp, err := parent_tp.GetType(atype)
	if err != nil {
		return nil, err
	}
	return g.GetTypeInfo(tp), nil
}

// Returns the source and wrapped namespace.
func (g *Generator) PhpWrapNS(depfile *fdep.DepFile) (sourceNS string, wrapNS string, wrapPath string) {
	if depfile == nil {
		return "", "", ""
	}

	sourceNS = g.BuildPHPNamespacedName(depfile.ProtoFile.PackageName)
	wrapNS = ""

	if g.NSSource != nil {
		if p, ok := g.NSSource.GetNS(depfile); ok {
			wrapNS = p
		}
	}

	if wrapNS == "" {
		for _, o := range depfile.ProtoFile.Options {
			if o.Name == "phpwrap_ns" {
				wrapNS = o.Value.String()
			}
		}
	}

	if wrapNS == "" {
		wrapNS = "FPWrap\\" + sourceNS
	}

	wrapPath = path.Join(strings.Split(wrapNS, "\\")...)
	return
}

func (g *Generator) IsWrap(tp *fdep.DepType) bool {
	if !g.IsFileWrap(tp.DepFile) || !tp.IsPointer() || tp.DepFile.DepType != fdep.DepType_Own {
		return false
	}
	return true
}
