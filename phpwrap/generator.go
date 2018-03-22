package fproto_phpwrap

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto-wrap"
	"github.com/RangelReale/fproto/fdep"
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
	filedep    *fdep.FileDep
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
	filedep, ok := dep.Files[filepath]
	if !ok {
		return nil, fmt.Errorf("File %s not found", filepath)
	}

	ret := &Generator{
		dep:     dep,
		filedep: filedep,
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
	if g.filedep.ProtoFile.Syntax == "proto3" {
		return GeneratorSyntax_Proto3
	}
	return GeneratorSyntax_Proto2
}

func (g *Generator) GetDep() *fdep.Dep {
	return g.dep
}

func (g *Generator) GetFileDep() *fdep.FileDep {
	return g.filedep
}

// Check if the file should be wrapped (the file option fproto_wrap.wrap=false disables it)
func (g *Generator) IsFileWrap(filedep *fdep.FileDep) bool {
	if filedep.DepType != fdep.DepType_Own {
		return false
	}

	if o := filedep.ProtoFile.FindOption("fproto_wrap.wrap"); o != nil {
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
	err := g.GenerateEnums()
	if err != nil {
		return err
	}

	err = g.GenerateMessages()
	if err != nil {
		return err
	}

	// CUSTOMIZER
	/*
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
	*/

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
	sourceNS, wrapNS, _ := g.PhpWrapNS(tp.FileDep)

	// Camel-cased name, with "." replaced by "_"
	phpName := fproto_wrap.CamelCaseProto(tp.Name)

	sourceName = fmt.Sprintf("\\%s\\%s", sourceNS, phpName)
	wrapName = fmt.Sprintf("\\%s\\%s", wrapNS, phpName)
	return
}

func (g *Generator) GenerateEnums() error {
	for _, s := range g.filedep.ProtoFile.CollectEnums() {
		err := g.GenerateEnum(s.(*fproto.EnumElement))
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) GenerateEnum(enum *fproto.EnumElement) error {
	sourceNS, _, wrapPath := g.PhpWrapNS(g.filedep)
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
	for _, s := range g.filedep.ProtoFile.CollectMessages() {
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
	sourceNS, wrapNS, _ := g.PhpWrapNS(tp.FileDep)

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
	tp_msg := g.dep.DepTypeFromElement(message)
	if tp_msg == nil {
		return errors.New("message type not found")
	}

	sourceNS, _, wrapPath := g.PhpWrapNS(g.filedep)
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
			tp_fld, err := tp_msg.MustGetType(xfld.Type)
			if err != nil {
				return err
			}

			_, wrapFieldTypeName := g.BuildTypeNSName(tp_fld)
			if xfld.Repeated {
				wrapFieldTypeName += "[]"
			}

			gf.GenerateFieldComment(fld, []string{
				fmt.Sprintf("@var %s", wrapFieldTypeName),
			})

			gf.P("private $", fldname, " = null;")
		case *fproto.MapFieldElement:
			// Get field type
			tp_fld, err := tp_msg.MustGetType(xfld.Type)
			if err != nil {
				return err
			}
			tp_keyfld, err := tp_msg.MustGetType(xfld.KeyType)
			if err != nil {
				return err
			}

			_, wrapFieldTypeName := g.BuildTypeNSName(tp_fld)
			_, wrapKeyFieldTypeName := g.BuildTypeNSName(tp_keyfld)

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
					tp_oofld, err := tp_msg.MustGetType(xoofld.Type)
					if err != nil {
						return err
					}

					_, wrapOOFieldTypeName := g.BuildTypeNSName(tp_oofld)
					if xoofld.Repeated {
						wrapOOFieldTypeName += "[]"
					}

					gf.GenerateFieldComment(oofld, []string{
						fmt.Sprintf("@var %s", wrapOOFieldTypeName),
					})

					gf.P("private $", oofldname, " = null;")
				case *fproto.MapFieldElement:
					// Get field type
					tp_oofld, err := tp_msg.MustGetType(xoofld.Type)
					if err != nil {
						return err
					}
					tp_ookeyfld, err := tp_msg.MustGetType(xoofld.KeyType)
					if err != nil {
						return err
					}

					_, wrapOOFieldTypeName := g.BuildTypeNSName(tp_oofld)
					_, wrapOOKeyFieldTypeName := g.BuildTypeNSName(tp_ookeyfld)

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
		fldname, fldgetter, fldsetter := g.BuildFieldName(fld)

		// public function getField() {
		// 		return $this->field;
		// }
		gf.P("public function ", fldgetter, "() {")
		gf.In()
		gf.P("return $this->", fldname, ";")
		gf.Out()
		gf.P("}")

		gf.P()

		switch xfld := fld.(type) {
		case *fproto.FieldElement, *fproto.MapFieldElement:
			// public function setField($var) {
			// 		$this->field = $var;
			//		return $this;
			// }
			gf.P("public function ", fldsetter, "($var) {")
			gf.In()
			gf.P("$this->", fldname, " = $var;")
			gf.P("return $this;")
			gf.Out()
			gf.P("}")

			gf.P()
		case *fproto.OneOfFieldElement:
			// each oneof field have a getter and a setter
			for _, oofld := range xfld.Fields {
				oofldname, oofldgetter, oofldsetter := g.BuildFieldName(oofld)

				// public function getField() {
				// 		return $this->field;
				// }
				gf.P("public function ", oofldgetter, "() {")
				gf.In()
				gf.P("return $this->", oofldname, ";")
				gf.Out()
				gf.P("}")

				gf.P()

				// public function setField($var) {
				// 		$this->field = $var;
				//		$this->oneoffield = 'field';
				//		return $this;
				// }
				gf.P("public function ", oofldsetter, "($var) {")
				gf.In()
				gf.P("$this->", oofldname, " = $var;")
				gf.P("$this->", fldname, " = '", oofldname, "';")
				gf.P("return $this;")
				gf.Out()
				gf.P("}")

				gf.P()
			}

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
		fldname, fldgetter, _ := g.BuildFieldName(fld)

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// Get field type
			tp_fld, err := tp_msg.MustGetType(xfld.Type)
			if err != nil {
				return err
			}

			if tp_fld.IsScalar() || !tp_fld.IsPointer() || tp_fld.FileDep.DepType != fdep.DepType_Own {
				// scalar or not pointer field
				gf.P("$this->", fldname, " = $source->", fldgetter, "();")
			} else {
				// convert field value
				_, wrapFieldTypeName := g.BuildTypeNSName(tp_fld)

				gf.P("if ($source->", fldgetter, "() !== null) {")
				gf.In()

				if !xfld.Repeated {
					gf.P("$this->", fldname, " = new ", wrapFieldTypeName, "();")
					gf.P("$this->", fldname, "->import($source->", fldgetter, "());")
				} else {
					// loop into array
					gf.P("$this->", fldname, " = [];")

					gf.P("foreach ($source->", fldgetter, "() as $mi => $mv) {")
					gf.In()

					varName := "$" + fldname + "__import"

					gf.P(varName, " = new ", wrapFieldTypeName, "();")
					gf.P(varName, "->import($mv);")

					gf.P("$this->", fldname, "[] = ", varName, ";")

					gf.Out()
					gf.P("}")
				}

				gf.Out()
				gf.P("}")
			}
		case *fproto.MapFieldElement:
			// Get map field type
			tp_fld, err := tp_msg.MustGetType(xfld.Type)
			if err != nil {
				return err
			}

			// convert field value
			gf.P("if ($source->", fldgetter, "() !== null) {")
			gf.In()

			gf.P("$this->", fldname, " = [];")

			gf.P("foreach ($source->", fldgetter, "() as $mi => $mv) {")
			gf.In()

			if tp_fld.IsScalar() {
				gf.P("$this->", fldname, "[$mi] = mv;")
			} else {
				_, wrapFieldTypeName := g.BuildTypeNSName(tp_fld)

				varName := "$" + fldname + "__import"

				gf.P(varName, " = new ", wrapFieldTypeName, "();")
				gf.P(varName, "->import($mv);")

				gf.P("$this->", fldname, "[$mi] = ", varName, ";")
			}

			gf.Out()
			gf.P("}")

			gf.Out()
			gf.P("}")
		case *fproto.OneOfFieldElement:
			gf.P("switch ($source->", fldgetter, "()) {")

			for _, oofld := range xfld.Fields {
				oofldname, oofldgetter, _ := g.BuildFieldName(oofld)

				gf.P("case '", oofldname, "':")
				gf.In()

				gf.P("$this->", oofldname, " = $source->", oofldgetter, "();")
				gf.P("$this->", fldname, " = '", oofldname, "';")

				gf.P("break;")
				gf.Out()
			}

			gf.P("}")
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
		fldname, _, fldsetter := g.BuildFieldName(fld)

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			tp_fld, err := tp_msg.MustGetType(xfld.Type)
			if err != nil {
				return err
			}

			if tp_fld.IsScalar() || !tp_fld.IsPointer() || tp_fld.FileDep.DepType != fdep.DepType_Own {
				gf.P("$ret->", fldsetter, "($this->", fldname, ");")
			} else {
				gf.P("if (isset($this->", fldname, ")) {")
				gf.In()

				if !xfld.Repeated {
					gf.P("$ret->", fldsetter, "($this->", fldname, "->export());")
				} else {
					varName := "$" + fldname + "__array"

					gf.P(varName, " = [];")
					gf.P("foreach ($this->", fldname, " as $mi => $mv) {")
					gf.In()

					gf.P(varName, "[] = $mv->export();")

					gf.Out()
					gf.P("}")

					gf.P("$ret->", fldsetter, "(", varName, ");")
				}

				gf.Out()
				gf.P("}")
			}
		case *fproto.MapFieldElement:
			tp_fld, err := tp_msg.MustGetType(xfld.Type)
			if err != nil {
				return err
			}

			gf.P("if (isset($this->", fldname, ")) {")
			gf.In()

			varName := "$" + fldname + "__map"

			gf.P(varName, " = [];")
			gf.P("foreach ($this->", fldname, " as $mi => $mv) {")
			gf.In()

			if tp_fld.IsScalar() || !tp_fld.IsPointer() || tp_fld.FileDep.DepType != fdep.DepType_Own {
				gf.P(varName, "[$mi] = $mv;")
			} else {
				gf.P(varName, "[$mi] = $mv->export();")
			}

			gf.Out()
			gf.P("}")

			gf.P("$ret->", fldsetter, "(", varName, ");")

			gf.Out()
			gf.P("}")
		case *fproto.OneOfFieldElement:
			gf.P("switch ($this->", fldname, ") {")

			for _, oofld := range xfld.Fields {
				oofldname, _, oofldsetter := g.BuildFieldName(oofld)

				gf.P("case '", oofldname, "':")
				gf.In()
				gf.P("$ret->", oofldsetter, "($this->", fldname, ");")
				gf.P("break;")
				gf.Out()
			}

			gf.P("}")

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
			pprefix = "else "
		}

		switch xfld := fld.(type) {
		case *fproto.FieldElement, *fproto.MapFieldElement:
			gf.P(pprefix, "if ($vname == '", fldname, "') {")
			gf.In()
			gf.P("$this->", fldsetter, "($vvalue);")
			gf.Out()
			gf.P("}")
		case *fproto.OneOfFieldElement:
			for _, oofld := range xfld.Fields {
				oofldname, _, oofldsetter := g.BuildFieldName(oofld)

				gf.P(pprefix, "if ($vname == '", oofldname, "') {")
				gf.In()
				gf.P("$this->", oofldsetter, "($vvalue);")
				gf.Out()
				gf.P("}")
			}
		}

	}

	if len(message.Fields) > 0 {
		gf.P("else {")
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

// Get type converter for type
func (g *Generator) getTypeConv(tp *fdep.DepType) TypeConverter {
	for _, tcp := range g.TypeConverters {
		tc := tcp.GetTypeConverter(tp)
		if tc != nil {
			return tc
		}
	}
	return nil
}

// Returns the source and wrapped namespace.
func (g *Generator) PhpWrapNS(filedep *fdep.FileDep) (sourceNS string, wrapNS string, wrapPath string) {
	if filedep == nil {
		return "", "", ""
	}

	sourceNS = g.BuildPHPNamespacedName(filedep.ProtoFile.PackageName)
	wrapNS = ""

	if g.NSSource != nil {
		if p, ok := g.NSSource.GetNS(filedep); ok {
			wrapNS = p
		}
	}

	if wrapNS == "" {
		for _, o := range filedep.ProtoFile.Options {
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
