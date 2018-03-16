package fproto_gowrap

import (
	"fmt"
	"path"
	"strings"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto/fdep"
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
	filedep    *fdep.FileDep
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
func NewGenerator(dep *fdep.Dep, filepath string) (*Generator, error) {
	filedep, ok := dep.Files[filepath]
	if !ok {
		return nil, fmt.Errorf("File %s not found", filepath)
	}

	ret := &Generator{
		dep:        dep,
		filedep:    filedep,
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
func (g *Generator) IsFileGowrap(filedep *fdep.FileDep) bool {
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
	for _, enum := range g.filedep.ProtoFile.Enums {
		err := g.generateEnum(enum)
		if err != nil {
			return err
		}
	}
	return nil
}

// Generates the protobuf messages
func (g *Generator) GenerateMessages() error {
	for _, message := range g.filedep.ProtoFile.Messages {
		err := g.generateMessage(message)
		if err != nil {
			return err
		}
	}
	return nil
}

// Generates the protobuf services
func (g *Generator) GenerateServices() error {
	if g.ServiceGen == nil || len(g.filedep.ProtoFile.Services) == 0 {
		return nil
	}

	for _, svc := range g.filedep.ProtoFile.Services {
		err := g.ServiceGen.GenerateService(g, svc)
		if err != nil {
			return err
		}
	}
	return nil
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
		case *fproto.OneofFieldElement:
			if !isfirst {
				ret = append(ret, el.Name)
			}
			cur = el.Parent
		case *fproto.EnumConstantElement:
			if !isfirst {
				ret = append(ret, el.Name)
			}
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

// Builds the message name.
// Given the proto:
// 		message A_msg { message B_test { string field; } }
// The returns value for the message "B" would be:
// 		goName: AMsg_BTest
// 		protoName: A_msg.B_test
// 		protoScope: A_msg
func (g *Generator) BuildMessageName(message *fproto.MessageElement) (goName string, protoName string, protoScope string) {
	// Get the message scope on the current file as an array
	scope := g.GetScope(message)

	goName = CamelCaseSlice(append(scope, CamelCase(message.Name)))
	protoName = strings.Join(append(scope, message.Name), ".")
	protoScope = strings.Join(scope, ".")

	return
}

// Builds the field name.
func (g *Generator) BuildFieldName(field fproto.FieldElementTag) (goName string, protoName string) {
	goName = CamelCase(field.FieldName())
	protoName = field.FieldName()

	return
}

// Helper as this is frequently used
func (g *Generator) BuildFieldGoName(field fproto.FieldElementTag) string {
	ret, _ := g.BuildFieldName(field)
	return ret
}

// Generates a message
func (g *Generator) generateMessage(message *fproto.MessageElement) error {
	if message.IsExtend {
		return nil
	}

	// build aliases to the original type
	go_alias_ie := g.FImpExp().FileDep(nil, "", false)

	// get the type names
	msgGoName, msgProtoName, _ := g.BuildMessageName(message)

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

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname fieldtype
			g.FMain().GenerateComment(xfld.Comment)

			tc_gowrap, err := g.GetGowrapType(msgProtoName, xfld.Type)
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

			g.FMain().P(g.BuildFieldGoName(xfld), " ", type_prefix, tc_gowrap.TypeName(g.FMain(), tctn), field_tag.OutputWithSpace())
		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype
			g.FMain().GenerateComment(xfld.Comment)

			tc_gowrap, err := g.GetGowrapType(msgProtoName, xfld.Type)
			if err != nil {
				return err
			}
			keytc_gowrap, err := g.GetGowrapType(msgProtoName, xfld.KeyType)
			if err != nil {
				return err
			}

			g.FMain().P(g.BuildFieldGoName(xfld), " map[", keytc_gowrap.TypeName(g.FMain(), TNT_TYPENAME), "]", tc_gowrap.TypeName(g.FMain(), TNT_TYPENAME), field_tag.OutputWithSpace())
		case *fproto.OneofFieldElement:
			// fieldname isSTRUCT_ONEOF
			g.FMain().GenerateComment(xfld.Comment)

			oneofGoName, _, _ := g.BuildOneOfName(xfld)

			g.FMain().P(g.BuildFieldGoName(xfld), " ", oneofGoName, field_tag.OutputWithSpace())
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

		g.FImpExp().P("// ", fldProtoName)

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname = go_package.fieldname
			tc_gowrap, err := g.GetGowrapType(msgProtoName, xfld.Type)
			if err != nil {
				return err
			}

			source_field := "s." + fldGoName
			dest_field := "ret." + fldGoName
			if xfld.Repeated {
				g.FImpExp().P("for _, ms := range s.", fldGoName, " {")
				g.FImpExp().In()
				g.FImpExp().P("var msi ", tc_gowrap.TypeName(g.FImpExp(), TNT_TYPENAME))

				source_field = "ms"
				dest_field = "msi"
			}

			check_error, err := tc_gowrap.GenerateImport(g.FImpExp(), source_field, dest_field, "err")
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

			tc_gowrap, err := g.GetGowrapType(msgProtoName, xfld.Type)
			if err != nil {
				return err
			}

			g.FImpExp().P("for msidx, ms := range s.", fldGoName, " {")
			g.FImpExp().In()
			g.FImpExp().P("var msi ", tc_gowrap.TypeName(g.FImpExp(), TNT_TYPENAME))

			check_error, err := tc_gowrap.GenerateImport(g.FImpExp(), "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + msgGoName + "{}")
			}

			g.FImpExp().P("ret.", fldGoName, "[msidx] = msi")

			g.FImpExp().Out()
			g.FImpExp().P("}")
		case *fproto.OneofFieldElement:
			g.FImpExp().P("switch en := s.", fldGoName, ".(type) {")

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oneofFieldGoName, _, _ := g.BuildOneOfFieldName(xoofld)

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
		fieldGoName, fieldProtoName := g.BuildFieldName(fld)

		g.FImpExp().P("// ", fieldProtoName)
		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname = go_package.fieldname

			tc_gowrap, tc_go, err := g.GetBothTypes(msgProtoName, xfld.Type)
			if err != nil {
				return err
			}

			source_field := "m." + fieldGoName
			dest_field := "ret." + fieldGoName
			if xfld.Repeated {
				g.FImpExp().P("for _, ms := range m.", fieldGoName, " {")
				g.FImpExp().In()
				g.FImpExp().P("var msi ", tc_go.TypeName(g.FImpExp(), TNT_TYPENAME))

				source_field = "ms"
				dest_field = "msi"
			}

			check_error, err := tc_gowrap.GenerateExport(g.FImpExp(), source_field, dest_field, "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + msgGoName + "{}")
			}

			if xfld.Repeated {
				g.FImpExp().P("ret.", fieldGoName, " = append(ret.", fieldGoName, ", msi)")

				g.FImpExp().Out()
				g.FImpExp().P("}")
			}

		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype

			tc_gowrap, tc_go, err := g.GetBothTypes(msgProtoName, xfld.Type)
			if err != nil {
				return err
			}

			g.FImpExp().P("for msidx, ms := range m.", fieldGoName, " {")
			g.FImpExp().In()
			g.FImpExp().P("var msi ", tc_go.TypeName(g.FImpExp(), TNT_TYPENAME))

			check_error, err := tc_gowrap.GenerateExport(g.FImpExp(), "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + msgGoName + "{}")
			}

			g.FImpExp().P("ret.", fieldGoName, "[msidx] = msi")

			g.FImpExp().Out()
			g.FImpExp().P("}")
		case *fproto.OneofFieldElement:
			g.FImpExp().P("switch en := m.", fieldGoName, ".(type) {")

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oneofFieldGoName, _, _ := g.BuildOneOfFieldName(xoofld)

					g.FImpExp().P("case *", oneofFieldGoName, ":")
					g.FImpExp().In()

					g.FImpExp().P("ret.", fieldGoName, ", err = ", "en.Export()")

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

	// Enums
	for _, enum := range message.Enums {
		err := g.generateEnum(enum)
		if err != nil {
			return err
		}
	}

	// Oneofs
	for _, fld := range message.Fields {
		switch xfld := fld.(type) {
		case *fproto.OneofFieldElement:
			err := g.generateOneOf(xfld)
			if err != nil {
				return err
			}
		}
	}

	// Submessages
	for _, submsg := range message.Messages {
		err := g.generateMessage(submsg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) BuildEnumName(enum *fproto.EnumElement) (goName string, protoName string, protoScope string) {
	// Get the enum scope on the current file as an array
	scope := g.GetScope(enum)

	// enums don't have "_" at last part
	goName = CamelCaseSlice(scope) + CamelCase(enum.Name)
	protoName = strings.Join(append(scope, enum.Name), ".")
	protoScope = strings.Join(scope, ".")

	return
}

func (g *Generator) BuildEnumConstantName(ec *fproto.EnumConstantElement) (goName string, protoName string, protoScope string) {
	// Get the enum constant scope on the current file as an array
	scope := g.GetScope(ec)

	// constants from root enums are named differently
	var ecbasename string
	if len(scope) <= 1 {
		ecbasename = CamelCaseSlice(scope)
	} else {
		// ignore the last scope
		ecbasename = CamelCaseSlice(scope[:len(scope)-1])
	}

	// enum constant name isn't camel-cased
	goName = ecbasename + "_" + ec.Name
	protoName = strings.Join(append(scope, ec.Name), ".")
	protoScope = strings.Join(scope, ".")

	return
}

func (g *Generator) generateEnum(enum *fproto.EnumElement) error {
	goName, protoName, _ := g.BuildEnumName(enum)

	// build aliases to the original type
	go_alias := g.FMain().FileDep(nil, "", false)

	//
	// type MyEnum = go_package.Enum
	//
	if !g.FMain().GenerateComment(enum.Comment) {
		g.FMain().GenerateCommentLine("ENUM: ", protoName)
	}

	g.FMain().P("type ", goName, " = ", go_alias, ".", goName)
	g.FMain().P()
	g.FMain().P("const (")
	g.FMain().In()

	for _, ec := range enum.EnumConstants {
		// MyEnumConstant MyEnum = go_package.MyEnumConstant
		ecGoName, _, _ := g.BuildEnumConstantName(ec)

		g.FMain().GenerateComment(ec.Comment)

		g.FMain().P(ecGoName, " ", goName, " = ", go_alias, ".", ecGoName)
	}

	g.FMain().Out()
	g.FMain().P(")")
	g.FMain().P()

	// var MyEnum_name = go_package.MyEnum_name
	g.FMain().P("var ", goName, "_name = ", go_alias, ".", goName, "_name")

	// var MyEnum_value = go_package.MyEnum_value
	g.FMain().P("var ", goName, "_value = ", go_alias, ".", goName, "_value")

	g.FMain().P()

	return nil
}

func (g *Generator) BuildOneOfName(oneof *fproto.OneofFieldElement) (goName string, protoName string, protoScope string) {
	// Get the oneof scope on the current file as an array
	scope := g.GetScope(oneof)

	// oneof have an "is" prefix
	goName = "is" + CamelCaseSlice(append(scope, CamelCase(oneof.Name)))
	protoName = strings.Join(append(scope, oneof.Name), ".")
	protoScope = strings.Join(scope, ".")

	return
}

func (g *Generator) BuildOneOfFieldName(oneoffield fproto.FieldElementTag) (goName string, protoName string, protoScope string) {
	// Get the message scope on the current file as an array
	scope := g.GetScope(oneoffield)
	parent_scope := g.GetScope(oneoffield.ParentElement())

	// the Go name uses the parent as scope
	goName = CamelCaseSlice(append(parent_scope, CamelCase(oneoffield.FieldName())))
	protoName = strings.Join(append(scope, oneoffield.FieldName()), ".")
	protoScope = strings.Join(scope, ".")

	return
}

func (g *Generator) generateOneOf(oneof *fproto.OneofFieldElement) error {
	// CUSTOMIZER
	cz := &wrapCustomizers{g.Customizers}

	// build aliases to the original type
	go_alias_ie := g.FImpExp().FileDep(nil, "", false)

	goName, protoName, _ := g.BuildOneOfName(oneof)

	// type isSTRUCT_ONEOF interface {
	//		isSTRUCT_ONEOF()
	// }

	if !g.FMain().GenerateComment(oneof.Comment) {
		g.FMain().GenerateCommentLine("ONEOF: ", protoName)
	}

	g.FMain().P("type ", goName, " interface {")
	g.FMain().In()
	g.FMain().P(goName, "()")
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

		fieldGoName, _ := g.BuildFieldName(oofld)

		switch xoofld := oofld.(type) {
		case *fproto.FieldElement:
			// type STRUCT_ONEOFFIELD struct {
			// 		ONEOFFIELD fieldtype
			// }

			oneofFieldGoName, oneofFieldProtoName, _ := g.BuildOneOfFieldName(xoofld)

			g.FMain().P("type ", oneofFieldGoName, " struct {")
			g.FMain().In()

			// fieldname fieldtype
			tc_gowrap, err := g.GetGowrapType(oneofFieldProtoName, xoofld.Type)
			if err != nil {
				return err
			}

			g.FMain().P(fieldGoName, " ", tc_gowrap.TypeName(g.FMain(), TNT_TYPENAME), field_tag.OutputWithSpace())

			g.FMain().Out()
			g.FMain().P("}")
			g.FMain().P()

			// func (*STRUCT_ONEOFFIELD) isSTRUCT_ONEOF()  {}

			g.FMain().P("func (*", oneofFieldGoName, ") ", goName, "() {}")
			g.FMain().P()

			//
			// func (*STRUCT_ONEOFFIELD) Import()  {}
			//
			g.FImpExp().GenerateCommentLine("IMPORT: ", oneofFieldGoName)

			g.FImpExp().P("func ", oneofFieldGoName, "_Import(s *", go_alias_ie, ".", oneofFieldGoName, ") (*", oneofFieldGoName, ", error) {")
			g.FImpExp().In()

			g.FImpExp().P("var err error")
			g.FImpExp().P("ret := &", oneofFieldGoName, "{}")

			tcoo_gowrap, err := g.GetGowrapType(oneofFieldProtoName, xoofld.Type)
			if err != nil {
				return err
			}

			check_error, err := tcoo_gowrap.GenerateImport(g.FImpExp(), "s."+fieldGoName, "ret."+fieldGoName, "err")
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

			check_error, err = tcoo_gowrap.GenerateExport(g.FImpExp(), "o."+fieldGoName, "ret."+fieldGoName, "err")
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
func (g *Generator) getTypeConv(tp *fdep.DepType) TypeConverter {
	for _, tcp := range g.TypeConverters {
		tc := tcp.GetTypeConverter(tp)
		if tc != nil {
			return tc
		}
	}
	return nil
}

func (g *Generator) BuildTypeName(dt *fdep.DepType) (goName string, protoName string, protoScope string) {
	if dt.IsScalar() {
		return dt.ScalarType.GoType(), dt.ScalarType.GoType(), dt.ScalarType.ProtoType()
	}

	if dt.Item != nil {
		switch item := dt.Item.(type) {
		case *fproto.MessageElement:
			goName, protoName, protoScope = g.BuildMessageName(item)
			return
		case *fproto.EnumElement:
			goName, protoName, protoScope = g.BuildEnumName(item)
			return
		case *fproto.OneofFieldElement:
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
			return
		}
	}

	// Fallback
	return strings.Replace(dt.Name, ".", "_", -1), dt.Name, dt.Name
}

// Get gowrap type
// The parameters MUST be protobuf names
func (g *Generator) GetGowrapType(scope, fldtype string) (TypeConverter, error) {
	tp, err := g.GetDepType(scope, fldtype)
	if err != nil {
		return nil, err
	}
	if tp.IsScalar() {
		return &TypeConverter_Scalar{tp, fldtype}, nil
	} else {
		if tc := g.getTypeConv(tp); tc != nil {
			return tc, nil
		}
		return &TypeConverter_Default{g, tp, g.filedep, true}, nil
	}
}

// Get go type
// The parameters MUST be protobuf names
func (g *Generator) GetGoType(scope, fldtype string) (TypeConverter, error) {
	tp, err := g.GetDepType(scope, fldtype)
	if err != nil {
		return nil, err
	}
	if tp.IsScalar() {
		return &TypeConverter_Scalar{tp, fldtype}, nil
	} else {
		return &TypeConverter_Default{g, tp, g.filedep, false}, nil
	}
}

// Get both types
func (g *Generator) GetBothTypes(scope, fldtype string) (tc_gowrap TypeConverter, tc_go TypeConverter, err error) {
	tc_gowrap, err = g.GetGowrapType(scope, fldtype)
	if err != nil {
		return nil, nil, err
	}
	tc_go, err = g.GetGoType(scope, fldtype)
	if err != nil {
		return nil, nil, err
	}

	return
}

// Get dependent type
func (g *Generator) GetDepType(scope, fldtype string) (tp *fdep.DepType, err error) {
	// search scope recursivelly, starting from the name itself
	// example: GetDepType("google.protobuf", "Timestamp")
	//		search: Timestamp
	//		search: google.protobuf.Timestamp
	//		search: google.Timestamp
	sclist := []string{""} // first item is blank, so the name itself is searched first
	if len(scope) > 0 {
		sclist = append(sclist, strings.Split(scope, ".")...)
	}

	for sci := 0; sci < len(sclist); sci++ {
		var ffname string
		if sci == 0 {
			ffname = fldtype
		} else {
			ffname = strings.Join(sclist[1:sci+1], ".") + "." + fldtype
		}

		tp, err = g.filedep.GetType(ffname)
		if err != nil {
			return nil, err
		}
		if tp != nil {
			break
		}
	}

	if tp == nil {
		return nil, fmt.Errorf("Unable to find dependent type '%s' on scope '%s' in file '%s'", fldtype, scope, g.filedep.FilePath)
	}

	return
}

// Returns the wrapped package name.
func (g *Generator) GoWrapPackage(filedep *fdep.FileDep) string {
	if g.PkgSource != nil {
		if p, ok := g.PkgSource.GetPkg(filedep); ok {
			return p
		}
	}

	for _, o := range filedep.ProtoFile.Options {
		if o.Name == "gowrap_package" {
			return o.Value.String()
		}
	}
	for _, o := range filedep.ProtoFile.Options {
		if o.Name == "go_package" {
			return o.Value.String()
		}
	}
	return path.Dir(filedep.FilePath)
}
