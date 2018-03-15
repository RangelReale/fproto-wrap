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
		if o.Value != "true" {
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
	err = cz.GenerateCode(g, g.dep, g.filedep)
	if err != nil {
		return err
	}

	err = g.GenerateServices()
	if err != nil {
		return err
	}

	// CUSTOMIZER
	err = cz.GenerateServiceCode(g, g.dep, g.filedep)
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

func (g *Generator) generateMessage(message *fproto.MessageElement) error {
	if message.IsExtend {
		return nil
	}

	// build aliases to the original type
	//go_alias := g.FMain().FileDep(nil, "", false)
	go_alias_ie := g.FImpExp().FileDep(nil, "", false)

	// Get the message scope on the current file as an array
	scope := g.GetScope(message)

	// append the message name to the scope dot separated
	msgscopedname := append(scope, CamelCase(message.Name))
	msgscopednamestr := strings.Join(msgscopedname, ".")
	msgscopedpbnamestr := strings.Join(append(scope, message.Name), ".")

	structName := CamelCaseSlice(msgscopedname)

	// CUSTOMIZER
	cz := &wrapCustomizers{g.Customizers}

	//
	// type MyMessage struct
	//
	if !g.FMain().GenerateComment(message.Comment) {
		g.FMain().GenerateCommentLine("MESSAGE: ", msgscopedpbnamestr)
	}

	g.FMain().P("type ", structName, " struct {")
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

			tc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			var type_prefix string
			if xfld.Repeated {
				type_prefix = "[]"
			}

			g.FMain().P(CamelCase(xfld.Name), " ", type_prefix, tc_gowrap.TypeName(g.FMain(), TNT_FIELD_DEFINITION), field_tag.OutputWithSpace())
		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype
			g.FMain().GenerateComment(xfld.Comment)

			tc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}
			keytc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.KeyType)
			if err != nil {
				return err
			}

			g.FMain().P(CamelCase(xfld.Name), " map[", keytc_gowrap.TypeName(g.FMain(), TNT_TYPENAME), "]", tc_gowrap.TypeName(g.FMain(), TNT_TYPENAME), field_tag.OutputWithSpace())
		case *fproto.OneofFieldElement:
			// fieldname isSTRUCT_ONEOF
			g.FMain().GenerateComment(xfld.Comment)

			ooscopedname := append(msgscopedname, CamelCase(xfld.Name))
			ooscopednamestr := CamelCaseSlice(ooscopedname)

			g.FMain().P(CamelCase(xfld.Name), " is", ooscopednamestr, field_tag.OutputWithSpace())
		}
	}

	g.FMain().Out()
	g.FMain().P("}")
	g.FMain().P()

	//
	// func MyMessage_Import(s *go_package.MyMessage) (*MyMessage, error)
	//
	g.FImpExp().GenerateCommentLine("IMPORT: ", msgscopedpbnamestr)

	g.FImpExp().P("func ", structName, "_Import(s *", go_alias_ie, ".", structName, ") (*", structName, ", error) {")
	g.FImpExp().In()

	g.FImpExp().P("if s == nil {")
	g.FImpExp().In()
	g.FImpExp().P("return nil, nil")
	g.FImpExp().Out()
	g.FImpExp().P("}")
	g.FImpExp().P()

	g.FImpExp().P("var err error")
	g.FImpExp().P("ret := &", structName, "{}")

	for _, fld := range message.Fields {
		g.FImpExp().P("// ", msgscopedpbnamestr, ".", fld.FieldName())

		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname = go_package.fieldname
			tc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			source_field := "s." + CamelCase(xfld.Name)
			dest_field := "ret." + CamelCase(xfld.Name)
			if xfld.Repeated {
				g.FImpExp().P("for _, ms := range s.", CamelCase(xfld.Name), " {")
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
				g.FImpExp().GenerateErrorCheck("&" + structName + "{}")
			}

			if xfld.Repeated {
				g.FImpExp().P("ret.", CamelCase(xfld.Name), " = append(ret.", CamelCase(xfld.Name), ", msi)")

				g.FImpExp().Out()
				g.FImpExp().P("}")
			}
		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype

			tc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			g.FImpExp().P("for msidx, ms := range s.", CamelCase(xfld.Name), " {")
			g.FImpExp().In()
			g.FImpExp().P("var msi ", tc_gowrap.TypeName(g.FImpExp(), TNT_TYPENAME))

			check_error, err := tc_gowrap.GenerateImport(g.FImpExp(), "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + structName + "{}")
			}

			g.FImpExp().P("ret.", CamelCase(xfld.Name), "[msidx] = msi")

			g.FImpExp().Out()
			g.FImpExp().P("}")
		case *fproto.OneofFieldElement:
			g.FImpExp().P("switch en := s.", CamelCase(xfld.Name), ".(type) {")

			//ooscopedname := append(msgscopedname, CamelCase(xfld.Name))
			//ooscopednamestr := CamelCaseSlice(ooscopedname)

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oofldscopedname := append(msgscopedname, CamelCase(xoofld.Name))
					oofldscopednamestr := CamelCaseSlice(oofldscopedname)

					g.FImpExp().P("case *", go_alias_ie, ".", oofldscopednamestr, ":")
					g.FImpExp().In()

					g.FImpExp().P("ret.", CamelCase(xfld.Name), ", err = ", oofldscopednamestr, "_Import(en)")

					g.FImpExp().Out()
				}
			}

			g.FImpExp().P("}")

			g.FImpExp().GenerateErrorCheck("&" + structName + "{}")
		}
	}

	g.FImpExp().P("return ret, err")

	g.FImpExp().Out()
	g.FImpExp().P("}")

	g.FImpExp().P()

	//
	// func (m *MyMessage) Export() (*go_package.MyMessage, error)
	//
	g.FImpExp().GenerateCommentLine("EXPORT: ", msgscopedpbnamestr)

	g.FImpExp().P("func (m *", structName, ") Export() (*", go_alias_ie, ".", structName, ", error) {")
	g.FImpExp().In()

	g.FImpExp().P("if m == nil {")
	g.FImpExp().In()
	g.FImpExp().P("return nil, nil")
	g.FImpExp().Out()
	g.FImpExp().P("}")
	g.FImpExp().P()

	g.FImpExp().P("var err error")
	g.FImpExp().P("ret := &", go_alias_ie, ".", structName, "{}")

	for _, fld := range message.Fields {
		g.FImpExp().P("// ", msgscopedpbnamestr, ".", fld.FieldName())
		switch xfld := fld.(type) {
		case *fproto.FieldElement:
			// fieldname = go_package.fieldname

			tc_gowrap, tc_go, err := g.GetBothTypes(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			source_field := "m." + CamelCase(xfld.Name)
			dest_field := "ret." + CamelCase(xfld.Name)
			if xfld.Repeated {
				g.FImpExp().P("for _, ms := range m.", CamelCase(xfld.Name), " {")
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
				g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + structName + "{}")
			}

			if xfld.Repeated {
				g.FImpExp().P("ret.", CamelCase(xfld.Name), " = append(ret.", CamelCase(xfld.Name), ", msi)")

				g.FImpExp().Out()
				g.FImpExp().P("}")
			}

		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype

			tc_gowrap, tc_go, err := g.GetBothTypes(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			g.FImpExp().P("for msidx, ms := range m.", CamelCase(xfld.Name), " {")
			g.FImpExp().In()
			g.FImpExp().P("var msi ", tc_go.TypeName(g.FImpExp(), TNT_TYPENAME))

			check_error, err := tc_gowrap.GenerateExport(g.FImpExp(), "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + structName + "{}")
			}

			g.FImpExp().P("ret.", CamelCase(xfld.Name), "[msidx] = msi")

			g.FImpExp().Out()
			g.FImpExp().P("}")
		case *fproto.OneofFieldElement:
			g.FImpExp().P("switch en := m.", CamelCase(xfld.Name), ".(type) {")

			//ooscopedname := append(msgscopedname, CamelCase(xfld.Name))
			//ooscopednamestr := CamelCaseSlice(ooscopedname)

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oofldscopedname := append(msgscopedname, CamelCase(xoofld.Name))
					oofldscopednamestr := CamelCaseSlice(oofldscopedname)

					g.FImpExp().P("case *", oofldscopednamestr, ":")
					g.FImpExp().In()

					g.FImpExp().P("ret.", CamelCase(xfld.Name), ", err = ", "en.Export()")

					g.FImpExp().Out()
				}
			}

			g.FImpExp().P("}")

			g.FImpExp().GenerateErrorCheck("&" + go_alias_ie + "." + structName + "{}")
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

func (g *Generator) generateEnum(enum *fproto.EnumElement) error {
	// Get the enum scope on the current file as an array
	scope := g.GetScope(enum)

	// append the enum name to the scope dot separated
	enumscopedname := append(scope, CamelCase(enum.Name))
	//enumscopednamestr := strings.Join(enumscopedname, ".")
	enumscopedpbnamestr := strings.Join(append(scope, enum.Name), ".")

	enumName := CamelCaseSlice(enumscopedname)

	// build aliases to the original type
	go_alias := g.FMain().FileDep(nil, "", false)
	//go_alias_ie := g.FImpExp().FileDep(nil, "", false)

	//
	// type MyEnum = go_package.Enum
	//
	if !g.FMain().GenerateComment(enum.Comment) {
		g.FMain().GenerateCommentLine("ENUM: ", enumscopedpbnamestr)
	}

	g.FMain().P("type ", enumName, " = ", go_alias, ".", enumName)
	g.FMain().P()
	g.FMain().P("const (")
	g.FMain().In()

	for _, ec := range enum.EnumConstants {
		// MyEnumConstant MyEnum = go_package.MyEnumConstant
		var encscopedname []string
		if len(scope) == 0 {
			encscopedname = append(enumscopedname, CamelCase(ec.Name))
		} else {
			encscopedname = append(scope, CamelCase(ec.Name))
		}

		encscopednamestr := CamelCaseSlice(encscopedname)

		g.FMain().GenerateComment(ec.Comment)

		g.FMain().P(encscopednamestr, " ", enumName, " = ", go_alias, ".", encscopednamestr)
	}

	g.FMain().Out()
	g.FMain().P(")")
	g.FMain().P()

	// var MyEnum_name = go_package.MyEnum_name
	g.FMain().P("var ", enumName, "_name = ", go_alias, ".", enumName, "_name")

	// var MyEnum_value = go_package.MyEnum_value
	g.FMain().P("var ", enumName, "_value = ", go_alias, ".", enumName, "_value")

	g.FMain().P()

	return nil
}

func (g *Generator) generateOneOf(oneof *fproto.OneofFieldElement) error {
	// CUSTOMIZER
	cz := &wrapCustomizers{g.Customizers}

	// Get the oneof scope on the current file as an array
	scope := g.GetScope(oneof)
	scopestr := CamelCaseSlice(scope)

	// build aliases to the original type
	//go_alias := g.FMain().FileDep(nil, "", false)
	go_alias_ie := g.FImpExp().FileDep(nil, "", false)

	ooscopedname := append(scope, CamelCase(oneof.Name))
	//ooscopednamestr := strings.Join(ooscopedname, ".")
	ooscopedpbnamestr := strings.Join(append(scope, oneof.Name), ".")

	oneofName := CamelCaseSlice(ooscopedname)

	// type isSTRUCT_ONEOF interface {
	//		isSTRUCT_ONEOF()
	// }

	if !g.FMain().GenerateComment(oneof.Comment) {
		g.FMain().GenerateCommentLine("ONEOF: ", ooscopedpbnamestr)
	}

	g.FMain().P("type is", oneofName, " interface {")
	g.FMain().In()
	g.FMain().P("is", oneofName, "()")
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

		switch xoofld := oofld.(type) {
		case *fproto.FieldElement:
			// type STRUCT_ONEOFFIELD struct {
			// 		ONEOFFIELD fieldtype
			// }

			// WARNING: the field name uses the parent struct name, not the oneof name

			oofldscopedname := append(scope, CamelCase(xoofld.Name))
			//oofldscopednamestr := strings.Join(oofldscopedname, ".")
			oofldscopedpbnamestr := strings.Join(append(scope, oneof.Name, xoofld.Name), ".")

			ooFldName := CamelCaseSlice(oofldscopedname)

			if !g.FMain().GenerateComment(xoofld.Comment) {
				g.FMain().GenerateCommentLine("ONEOF Field: ", oofldscopedpbnamestr)
			}

			g.FMain().P("type ", ooFldName, " struct {")
			g.FMain().In()

			// fieldname fieldtype
			tc_gowrap, err := g.GetGowrapType(scopestr, xoofld.Type)
			if err != nil {
				return err
			}

			g.FMain().P(CamelCase(xoofld.Name), " ", tc_gowrap.TypeName(g.FMain(), TNT_TYPENAME), field_tag.OutputWithSpace())

			g.FMain().Out()
			g.FMain().P("}")
			g.FMain().P()

			// func (*STRUCT_ONEOFFIELD) isSTRUCT_ONEOF()  {}

			g.FMain().P("func (*", ooFldName, ") is", oneofName, "() {}")
			g.FMain().P()

			//
			// func (*STRUCT_ONEOFFIELD) Import()  {}
			//
			g.FImpExp().GenerateCommentLine("IMPORT: ", oofldscopedpbnamestr)

			g.FImpExp().P("func ", ooFldName, "_Import(s *", go_alias_ie, ".", ooFldName, ") (*", ooFldName, ", error) {")
			g.FImpExp().In()

			g.FImpExp().P("var err error")
			g.FImpExp().P("ret := &", ooFldName, "{}")

			tcoo_gowrap, err := g.GetGowrapType(scopestr, xoofld.Type)
			if err != nil {
				return err
			}

			check_error, err := tcoo_gowrap.GenerateImport(g.FImpExp(), "s."+CamelCase(xoofld.Name), "ret."+CamelCase(xoofld.Name), "err")
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
			g.FImpExp().GenerateCommentLine("EXPORT: ", oofldscopedpbnamestr)

			g.FImpExp().P("func (o *", ooFldName, ") Export() (*", go_alias_ie, ".", ooFldName, ", error) {")
			g.FImpExp().In()

			g.FImpExp().P("var err error")
			g.FImpExp().P("ret := &", go_alias_ie, ".", ooFldName, "{}")

			check_error, err = tcoo_gowrap.GenerateExport(g.FImpExp(), "o."+CamelCase(xoofld.Name), "ret."+CamelCase(xoofld.Name), "err")
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

// Get gowrap type
func (g *Generator) GetGowrapType(scope, fldtype string) (TypeConverter, error) {
	tp, isscalar, err := g.GetDepType(scope, fldtype)
	if err != nil {
		return nil, err
	}
	if isscalar {
		return &TypeConverter_Scalar{fldtype}, nil
	} else {
		if tc := g.getTypeConv(tp); tc != nil {
			return tc, nil
		}
		return &TypeConverter_Default{g, tp, g.filedep, true}, nil
	}
}

// Get go type
func (g *Generator) GetGoType(scope, fldtype string) (TypeConverter, error) {
	tp, isscalar, err := g.GetDepType(scope, fldtype)
	if err != nil {
		return nil, err
	}
	if isscalar {
		return &TypeConverter_Scalar{fldtype}, nil
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
func (g *Generator) GetDepType(scope, fldtype string) (tp *fdep.DepType, isscalar bool, err error) {
	// check if if scalar
	if _, ok := fproto.ParseScalarType(fldtype); ok {
		isscalar = true
	} else {
		isscalar = false
		var err error

		// search scope recursivelly, starting from the name itself
		// example: GetDepType("google.protobuf", "Timestamp")
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
				return nil, false, err
			}
			if tp != nil {
				break
			}
		}
	}

	if !isscalar && tp == nil {
		return nil, false, fmt.Errorf("Unable to find dependent type '%s' on scope '%s' in file '%s'", fldtype, scope, g.filedep.FilePath)
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
			return o.Value
		}
	}
	for _, o := range filedep.ProtoFile.Options {
		if o.Name == "go_package" {
			return o.Value
		}
	}
	return path.Dir(filedep.FilePath)
}
