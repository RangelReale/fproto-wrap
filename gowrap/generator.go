package fproto_gowrap

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

// Output file id
const (
	FILEID_MAIN          = "main"
	FILEID_IMPORT_EXPORT = "import_export"
	FILEID_SERVICE       = "service"
)

// Generators generates a wrapper for a single source file.
// There can be more than one output files.
type Generator struct {
	dep        *fdep.Dep
	depfile    *fdep.DepFile
	tc_default TypeConverter

	// Files to output
	Files      map[string]*GeneratorFile
	FilesAlias map[string]string

	// Interface to do package name generation.
	PkgSource PkgSource

	// List of type conversions
	TypeConverters []TypeConverterPlugin

	// Service generator
	ServiceGen ServiceGen

	// Customizers
	Customizers []Customizer
}

// Creates a new generator for the file path.
func NewGenerator(dep *fdep.Dep, depfile *fdep.DepFile) (*Generator, error) {
	ret := &Generator{
		dep:        dep,
		depfile:    depfile,
		Files:      make(map[string]*GeneratorFile),
		FilesAlias: make(map[string]string),
	}

	// Creates the main file
	ret.Files[FILEID_MAIN] = NewGeneratorFile(ret, "main", "")

	// Alias import_export to main
	ret.FilesAlias[FILEID_IMPORT_EXPORT] = FILEID_MAIN
	// Alias service to main
	ret.FilesAlias[FILEID_SERVICE] = FILEID_MAIN

	return ret, nil
}

// Creates a new file
func (g *Generator) SetFile(fileId string, suffix string) {
	g.Files[fileId] = NewGeneratorFile(g, fileId, suffix)
	delete(g.FilesAlias, fileId)
}

// Creates a new file with fixed filename
func (g *Generator) SetFileFixed(fileId string, filename string) {
	g.Files[fileId] = NewGeneratorFileFixed(g, fileId, filename)
	delete(g.FilesAlias, fileId)
}

// Sets one file as alias of another
func (g *Generator) SetFileAlias(fileId string, sourceFileId string) {
	g.FilesAlias[fileId] = sourceFileId
	delete(g.Files, fileId)
}

// Gets a file by id
func (g *Generator) F(fileId string) *GeneratorFile {
	if gf, ok := g.Files[fileId]; ok {
		return gf
	}

	// Search in alias
	if gfa, ok := g.FilesAlias[fileId]; ok {
		if fileId == gfa {
			panic("Infinite loop")
		}
		return g.F(gfa)
	}

	panic(fmt.Sprintf("Generator file id %s not found", fileId))
	return nil
}

// Helper to get the MAIN file
func (g *Generator) FMain() *GeneratorFile {
	return g.F(FILEID_MAIN)
}

// Helper to get the IMPORT_EXPORT file
func (g *Generator) FImpExp() *GeneratorFile {
	return g.F(FILEID_IMPORT_EXPORT)
}

// Helper to get the SERVICE file
func (g *Generator) FService() *GeneratorFile {
	return g.F(FILEID_SERVICE)
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

// Generates the protobuf enums
func (g *Generator) GenerateEnums() error {
	for _, enum := range g.depfile.ProtoFile.CollectEnums() {
		err := g.generateEnum(enum.(*fproto.EnumElement))
		if err != nil {
			return err
		}
	}
	return nil
}

// Generates the protobuf messages
func (g *Generator) GenerateMessages() error {
	for _, message := range g.depfile.ProtoFile.CollectMessages() {
		err := g.generateMessage(message.(*fproto.MessageElement))
		if err != nil {
			return err
		}
	}
	return nil
}

// Generates the protobuf services
func (g *Generator) GenerateServices() error {
	if g.ServiceGen == nil || len(g.depfile.ProtoFile.Services) == 0 {
		return nil
	}

	for _, svc := range g.depfile.ProtoFile.CollectServices() {
		err := g.ServiceGen.GenerateService(g, svc.(*fproto.ServiceElement))
		if err != nil {
			return err
		}
	}
	return nil
}

// Builds the message name.
// Given the proto:
// 		message A_msg { message B_test { string field; } }
// The returns value for the message "B" would be:
// 		goName: AMsg_BTest
// 		protoName: A_msg.B_test
func (g *Generator) BuildMessageName(message *fproto.MessageElement) (goName string, protoName string) {
	// get the dep type
	tp_message := g.dep.DepTypeFromElement(message)
	if tp_message == nil {
		panic("message type not found")
	}

	goName = fproto_wrap.CamelCaseProto(tp_message.Name)
	protoName = tp_message.Name

	return
}

// Builds the field name.
func (g *Generator) BuildFieldName(field fproto.FieldElementTag) (goName string, protoName string) {
	goName = fproto_wrap.CamelCase(field.FieldName())
	protoName = field.FieldName()

	return
}

// Generates a message
func (g *Generator) generateMessage(message *fproto.MessageElement) error {
	if message.IsExtend {
		return nil
	}

	// build aliases to the original type
	go_alias_ie := g.FImpExp().DeclFileDep(nil, "", false)

	// get the message DepType
	tp_msg := g.dep.DepTypeFromElement(message)
	if tp_msg == nil {
		return errors.New("message type not found")
	}

	// get the type names
	msgGoName, msgProtoName := g.BuildMessageName(message)

	// CUSTOMIZER
	cz := &wrapCustomizers{g.Customizers}

	//
	// type MyMessage struct
	//
	if !g.FMain().GenerateComment(message.Comment) {
		g.FMain().GenerateCommentLine("MESSAGE: ", msgProtoName)
	}

	g.FMain().P("type ", msgGoName, " struct {")
	g.FMain().In()

	for _, fld := range message.Fields {
		// CUSTOMIZER
		field_tag := NewStructTag()

		err := cz.GetTag(g, field_tag, message, fld)
		if err != nil {
			return err
		}

		fldGoName, _ := g.BuildFieldName(fld)

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname fieldtype
			g.FMain().GenerateComment(xfld.Comment)

			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}

			var type_prefix string
			tctn := TNT_FIELD_DEFINITION
			if xfld.Repeated {
				type_prefix = "[]"
				// when array, never add pointer to scalar type
				tctn = TNT_TYPENAME
			}

			g.FMain().P(fldGoName, " ", type_prefix, tinfo.Converter().TypeName(g.FMain(), tctn, 0), field_tag.OutputWithSpace())
		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype
			g.FMain().GenerateComment(xfld.Comment)

			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}
			tinfokey, err := g.GetTypeInfoFromParent(tp_msg, xfld.KeyType)
			if err != nil {
				return err
			}

			g.FMain().P(fldGoName, " map[", tinfokey.Converter().TypeName(g.FMain(), TNT_TYPENAME, 0), "]", tinfo.Converter().TypeName(g.FMain(), TNT_TYPENAME, 0), field_tag.OutputWithSpace())
		case *fproto.OneOfFieldElement:
			// fieldname isSTRUCT_ONEOF
			g.FMain().GenerateComment(xfld.Comment)

			oneofGoName, _ := g.BuildOneOfName(xfld)

			g.FMain().P(fldGoName, " ", oneofGoName, field_tag.OutputWithSpace())
		}
	}

	g.FMain().Out()
	g.FMain().P("}")
	g.FMain().P()

	//
	// func MyMessage_Import(s *go_package.MyMessage) (*MyMessage, error)
	//
	g.FImpExp().GenerateCommentLine("IMPORT: ", msgProtoName)

	g.FImpExp().P("func ", msgGoName, "_Import(s *", go_alias_ie, ".", msgGoName, ") (*", msgGoName, ", error) {")
	g.FImpExp().In()

	g.FImpExp().P("if s == nil {")
	g.FImpExp().In()
	g.FImpExp().P("return nil, nil")
	g.FImpExp().Out()
	g.FImpExp().P("}")
	g.FImpExp().P()

	g.FImpExp().P("var err error")
	g.FImpExp().P("ret := &", msgGoName, "{}")

	for _, fld := range message.Fields {
		fldGoName, fldProtoName := g.BuildFieldName(fld)

		g.FImpExp().P("// ", msgProtoName, ".", fldProtoName)

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname = go_package.fieldname
			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}

			source_field := "s." + fldGoName
			dest_field := "ret." + fldGoName
			if xfld.Repeated {
				g.FImpExp().P("for _, ms := range s.", fldGoName, " {")
				g.FImpExp().In()
				g.FImpExp().P("var msi ", tinfo.Converter().TypeName(g.FImpExp(), TNT_TYPENAME, 0))

				source_field = "ms"
				dest_field = "msi"
			}

			check_error, err := tinfo.Converter().GenerateImport(g.FImpExp(), source_field, dest_field, "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + msgGoName + "{}")
			}

			if xfld.Repeated {
				g.FImpExp().P("ret.", fldGoName, " = append(ret.", fldGoName, ", msi)")

				g.FImpExp().Out()
				g.FImpExp().P("}")
			}
		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype
			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}
			tinfokey, err := g.GetTypeInfoFromParent(tp_msg, xfld.KeyType)
			if err != nil {
				return err
			}

			g.FImpExp().P("if len(s.", fldGoName, ") > 0 {")
			g.FImpExp().In()

			g.FImpExp().P("ret.", fldGoName, "= make(map[", tinfokey.Converter().TypeName(g.FMain(), TNT_TYPENAME, 0), "]", tinfo.Converter().TypeName(g.FMain(), TNT_TYPENAME, 0), ")")

			g.FImpExp().P("for msidx, ms := range s.", fldGoName, " {")
			g.FImpExp().In()
			g.FImpExp().P("var msi ", tinfo.Converter().TypeName(g.FImpExp(), TNT_TYPENAME, 0))

			check_error, err := tinfo.Converter().GenerateImport(g.FImpExp(), "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + msgGoName + "{}")
			}

			g.FImpExp().P("ret.", fldGoName, "[msidx] = msi")

			g.FImpExp().Out()
			g.FImpExp().P("}")

			g.FImpExp().Out()
			g.FImpExp().P("}")
		case *fproto.OneOfFieldElement:
			g.FImpExp().P("switch en := s.", fldGoName, ".(type) {")

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oneofFieldGoName, _ := g.BuildOneOfFieldName(xoofld)

					g.FImpExp().P("case *", go_alias_ie, ".", oneofFieldGoName, ":")
					g.FImpExp().In()

					g.FImpExp().P("ret.", fldGoName, ", err = ", oneofFieldGoName, "_Import(en)")

					g.FImpExp().Out()
				}
			}

			g.FImpExp().P("}")

			g.FImpExp().GenerateErrorCheck("&" + msgGoName + "{}")
		}
	}

	g.FImpExp().P("return ret, err")

	g.FImpExp().Out()
	g.FImpExp().P("}")

	g.FImpExp().P()

	//
	// func (m *MyMessage) Export() (*go_package.MyMessage, error)
	//
	g.FImpExp().GenerateCommentLine("EXPORT: ", msgProtoName)

	g.FImpExp().P("func (m *", msgGoName, ") Export() (*", go_alias_ie, ".", msgGoName, ", error) {")
	g.FImpExp().In()

	g.FImpExp().P("if m == nil {")
	g.FImpExp().In()
	g.FImpExp().P("return nil, nil")
	g.FImpExp().Out()
	g.FImpExp().P("}")
	g.FImpExp().P()

	g.FImpExp().P("var err error")
	g.FImpExp().P("ret := &", go_alias_ie, ".", msgGoName, "{}")

	for _, fld := range message.Fields {
		fldGoName, fldProtoName := g.BuildFieldName(fld)

		g.FImpExp().P("// ", msgProtoName, ".", fldProtoName)

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname = go_package.fieldname
			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}

			source_field := "m." + fldGoName
			dest_field := "ret." + fldGoName
			if xfld.Repeated {
				g.FImpExp().P("for _, ms := range m.", fldGoName, " {")
				g.FImpExp().In()
				g.FImpExp().P("var msi ", tinfo.Source().TypeName(g.FImpExp(), TNT_TYPENAME, 0))

				source_field = "ms"
				dest_field = "msi"
			}

			check_error, err := tinfo.Converter().GenerateExport(g.FImpExp(), source_field, dest_field, "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + msgGoName + "{}")
			}

			if xfld.Repeated {
				g.FImpExp().P("ret.", fldGoName, " = append(ret.", fldGoName, ", msi)")

				g.FImpExp().Out()
				g.FImpExp().P("}")
			}

		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype
			tinfo, err := g.GetTypeInfoFromParent(tp_msg, xfld.Type)
			if err != nil {
				return err
			}

			tinfokey, err := g.GetTypeInfoFromParent(tp_msg, xfld.KeyType)
			if err != nil {
				return err
			}

			g.FImpExp().P("if len(m.", fldGoName, ") > 0 {")
			g.FImpExp().In()

			g.FImpExp().P("ret.", fldGoName, "= make(map[", tinfokey.Source().TypeName(g.FImpExp(), TNT_TYPENAME, 0), "]", tinfo.Source().TypeName(g.FImpExp(), TNT_TYPENAME, 0), ")")

			g.FImpExp().P("for msidx, ms := range m.", fldGoName, " {")
			g.FImpExp().In()
			g.FImpExp().P("var msi ", tinfo.Source().TypeName(g.FImpExp(), TNT_TYPENAME, 0))

			check_error, err := tinfo.Converter().GenerateExport(g.FImpExp(), "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + msgGoName + "{}")
			}

			g.FImpExp().P("ret.", fldGoName, "[msidx] = msi")

			g.FImpExp().Out()
			g.FImpExp().P("}")

			g.FImpExp().Out()
			g.FImpExp().P("}")
		case *fproto.OneOfFieldElement:
			g.FImpExp().P("switch en := m.", fldGoName, ".(type) {")

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oneofFieldGoName, _ := g.BuildOneOfFieldName(xoofld)

					g.FImpExp().P("case *", oneofFieldGoName, ":")
					g.FImpExp().In()

					g.FImpExp().P("ret.", fldGoName, ", err = ", "en.Export()")

					g.FImpExp().Out()
				}
			}

			g.FImpExp().P("}")

			g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + msgGoName + "{}")
		}
	}

	g.FImpExp().P("return ret, err")

	g.FImpExp().Out()
	g.FImpExp().P("}")

	g.FImpExp().P()

	// Oneofs
	for _, fld := range message.Fields {
		switch xfld := fld.(type) {
		case *fproto.OneOfFieldElement:
			err := g.generateOneOf(xfld)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Generator) BuildEnumName(enum *fproto.EnumElement) (goName string, protoName string) {
	// get the dep type
	tp_enum := g.dep.DepTypeFromElement(enum)
	if tp_enum == nil {
		panic("enum type not found")
	}

	// Camel-cased name, with "." replaced by "_"
	goName = fproto_wrap.CamelCaseProto(tp_enum.Name)

	protoName = tp_enum.Name

	return
}

func (g *Generator) BuildEnumConstantName(ec *fproto.EnumConstantElement) (goName string, protoName string) {
	// get the dep type
	tp_ec := g.dep.DepTypeFromElement(ec)
	if tp_ec == nil {
		panic("enum constant type not found")
	}

	// Skip up to 2 parents if available (constants from root enums are named differently).
	tp_parent, _ := tp_ec.SkipParents(2)
	if tp_parent == nil {
		panic("enum constant parent type not found")
	}

	// enum constant name isn't camel-cased
	goName = fproto_wrap.CamelCaseProto(tp_parent.Name) + "_" + ec.Name

	protoName = tp_ec.Name

	return
}

func (g *Generator) generateEnum(enum *fproto.EnumElement) error {
	enGoName, enProtoName := g.BuildEnumName(enum)

	// build aliases to the original type
	go_alias := g.FMain().DeclFileDep(nil, "", false)

	//
	// type MyEnum = go_package.Enum
	//
	if !g.FMain().GenerateComment(enum.Comment) {
		g.FMain().GenerateCommentLine("ENUM: ", enProtoName)
	}

	g.FMain().P("type ", enGoName, " = ", go_alias, ".", enGoName)
	g.FMain().P()
	g.FMain().P("const (")
	g.FMain().In()

	for _, ec := range enum.EnumConstants {
		// MyEnumConstant MyEnum = go_package.MyEnumConstant
		ecGoName, _ := g.BuildEnumConstantName(ec)

		g.FMain().GenerateComment(ec.Comment)

		g.FMain().P(ecGoName, " ", enGoName, " = ", go_alias, ".", ecGoName)
	}

	g.FMain().Out()
	g.FMain().P(")")
	g.FMain().P()

	// var MyEnum_name = go_package.MyEnum_name
	g.FMain().P("var ", enGoName, "_name = ", go_alias, ".", enGoName, "_name")

	// var MyEnum_value = go_package.MyEnum_value
	g.FMain().P("var ", enGoName, "_value = ", go_alias, ".", enGoName, "_value")

	g.FMain().P()

	return nil
}

func (g *Generator) BuildOneOfName(oneof *fproto.OneOfFieldElement) (goName string, protoName string) {
	// get the dep type
	tp_oneof := g.dep.DepTypeFromElement(oneof)
	if tp_oneof == nil {
		panic("oneof type not found")
	}

	// oneof have an "is" prefix
	goName = "is" + fproto_wrap.CamelCaseProtoElement(tp_oneof.Name)

	protoName = tp_oneof.Name

	return
}

func (g *Generator) BuildOneOfFieldName(oneoffield fproto.FieldElementTag) (goName string, protoName string) {
	// get the dep type
	tp_fld := g.dep.DepTypeFromElement(oneoffield)
	if tp_fld == nil {
		panic("oneof field type not found")
	}

	// skip 2 parents to the message
	tp_msg, _ := tp_fld.SkipParents(2)
	if tp_msg == nil {
		panic("oneof field message type not found")
	}

	// the Go name uses the message as the scope
	goName = fproto_wrap.CamelCaseProtoElement(tp_msg.Name) + "_" + fproto_wrap.CamelCase(oneoffield.FieldName())

	protoName = tp_fld.Name

	return
}

func (g *Generator) generateOneOf(oneof *fproto.OneOfFieldElement) error {
	// CUSTOMIZER
	cz := &wrapCustomizers{g.Customizers}

	// get DepType from element
	tp_oneof := g.dep.DepTypeFromElement(oneof)
	if tp_oneof == nil {
		return errors.New("oneof type not found")
	}

	// build aliases to the original type
	go_alias_ie := g.FImpExp().DeclFileDep(nil, "", false)

	ooGoName, ooProtoName := g.BuildOneOfName(oneof)

	// type isSTRUCT_ONEOF interface {
	//		isSTRUCT_ONEOF()
	// }

	if !g.FMain().GenerateComment(oneof.Comment) {
		g.FMain().GenerateCommentLine("ONEOF: ", ooProtoName)
	}

	g.FMain().P("type ", ooGoName, " interface {")
	g.FMain().In()
	g.FMain().P(ooGoName, "()")
	g.FMain().Out()
	g.FMain().P("}")
	g.FMain().P()

	for _, oofld := range oneof.Fields {
		// CUSTOMIZER
		field_tag := NewStructTag()

		err := cz.GetTag(g, field_tag, oneof, oofld)
		if err != nil {
			return err
		}

		fldGoName, _ := g.BuildFieldName(oofld)

		switch xoofld := oofld.(type) {
		case *fproto.FieldElement:
			// type STRUCT_ONEOFFIELD struct {
			// 		ONEOFFIELD fieldtype
			// }
			tinfo, err := g.GetTypeInfoFromParent(tp_oneof, xoofld.Type)
			if err != nil {
				return err
			}

			oneofFieldGoName, oneofFieldProtoName := g.BuildOneOfFieldName(xoofld)

			g.FMain().P("type ", oneofFieldGoName, " struct {")
			g.FMain().In()

			// fieldname fieldtype
			g.FMain().P(fldGoName, " ", tinfo.Converter().TypeName(g.FMain(), TNT_TYPENAME, 0), field_tag.OutputWithSpace())

			g.FMain().Out()
			g.FMain().P("}")
			g.FMain().P()

			// func (*STRUCT_ONEOFFIELD) isSTRUCT_ONEOF()  {}

			g.FMain().P("func (*", oneofFieldGoName, ") ", ooGoName, "() {}")
			g.FMain().P()

			//
			// func (*STRUCT_ONEOFFIELD) Import()  {}
			//
			g.FImpExp().GenerateCommentLine("IMPORT: ", oneofFieldProtoName)

			g.FImpExp().P("func ", oneofFieldGoName, "_Import(s *", go_alias_ie, ".", oneofFieldGoName, ") (*", oneofFieldGoName, ", error) {")
			g.FImpExp().In()

			g.FImpExp().P("var err error")
			g.FImpExp().P("ret := &", oneofFieldGoName, "{}")

			check_error, err := tinfo.Converter().GenerateImport(g.FImpExp(), "s."+fldGoName, "ret."+fldGoName, "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("nil")
			}

			g.FImpExp().P("return ret, err")
			g.FImpExp().Out()
			g.FImpExp().P("}")
			g.FImpExp().P()

			//
			// func (*STRUCT_ONEOFFIELD) Export()  {}
			//
			g.FImpExp().GenerateCommentLine("EXPORT: ", oneofFieldProtoName)

			g.FImpExp().P("func (o *", oneofFieldGoName, ") Export() (*", go_alias_ie, ".", oneofFieldGoName, ", error) {")
			g.FImpExp().In()

			g.FImpExp().P("var err error")
			g.FImpExp().P("ret := &", go_alias_ie, ".", oneofFieldGoName, "{}")

			check_error, err = tinfo.Converter().GenerateExport(g.FImpExp(), "o."+fldGoName, "ret."+fldGoName, "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("nil")
			}

			g.FImpExp().P("return ret, err")
			g.FImpExp().Out()
			g.FImpExp().P("}")
			g.FImpExp().P()
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

func (g *Generator) BuildTypeName(dt *fdep.DepType) (goName string, protoName string) {
	if dt.IsScalar() {
		return dt.ScalarType.GoType(), dt.ScalarType.GoType()
	}

	if dt.Item != nil {
		switch item := dt.Item.(type) {
		case *fproto.MessageElement:
			goName, protoName = g.BuildMessageName(item)
			return
		case *fproto.EnumElement:
			goName, protoName = g.BuildEnumName(item)
			return
		case *fproto.OneOfFieldElement:
			goName, protoName = g.BuildOneOfName(item)
			return
		case fproto.FieldElementTag:
			// if the parent is a oneof, call a different function
			switch item.ParentElement().(type) {
			case *fproto.EnumElement:
				goName, protoName = g.BuildOneOfFieldName(item)
			default:
				goName, protoName = g.BuildFieldName(item)
			}
			return
		}
	}

	// Fallback
	return strings.Replace(dt.Name, ".", "_", -1), dt.Name
}

// Gets the type for the source protoc-gen-go generated names
func (g *Generator) GetTypeSource(tp *fdep.DepType) TypeNamer {
	if tp.IsScalar() {
		return &TypeNamer_Scalar{tp: tp}
	} else {
		return &TypeNamer_Source{tp: tp}
	}
}

// Gets the type for the gowrap converter
func (g *Generator) GetTypeConverter(tp *fdep.DepType) TypeConverter {
	if tp.IsScalar() {
		return &TypeConverter_Scalar{tp: tp}
	} else {
		if tc := g.findTypeConv(tp); tc != nil {
			return tc
		} else {
			return &TypeConverter_Default{g: g, tp: tp, depfile: g.depfile}
		}
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

// Returns the source package name.
func (g *Generator) GoPackage(depfile *fdep.DepFile) string {
	for _, o := range depfile.ProtoFile.Options {
		if o.Name == "go_package" {
			return o.Value.String()
		}
	}
	return path.Dir(depfile.FilePath)
}

// Returns the wrapped package name.
func (g *Generator) GoWrapPackage(depfile *fdep.DepFile) string {
	if g.PkgSource != nil {
		if p, ok := g.PkgSource.GetPkg(depfile); ok {
			return p
		}
	}

	for _, o := range depfile.ProtoFile.Options {
		if o.Name == "gowrap_package" {
			return o.Value.String()
		}
	}

	// prepend "fpwrap"
	for _, o := range depfile.ProtoFile.Options {
		if o.Name == "go_package" {
			return path.Join("fpwrap", o.Value.String())
		}
	}
	return path.Join("fpwrap", path.Dir(depfile.FilePath))
}

// Returns the source file package name.
func (g *Generator) GoFilePackage(depfile *fdep.DepFile) string {
	return fproto_wrap.BaseName(g.GoWrapPackage(depfile))
}

// Returns the wrapped file package name.
func (g *Generator) GoWrapFilePackage(depfile *fdep.DepFile) string {
	if g.PkgSource != nil {
		if p, ok := g.PkgSource.GetFilePkg(depfile); ok {
			return p
		}
	}

	return "fw" + fproto_wrap.BaseName(g.GoWrapPackage(depfile))
}
