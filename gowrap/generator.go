package fproto_gowrap

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto/fdep"
)

type GeneratorSyntax int

const (
	GeneratorSyntax_Proto2 GeneratorSyntax = iota
	GeneratorSyntax_Proto3
)

// Generators generates a wrapper for a single file.
type Generator struct {
	*bytes.Buffer
	indent string

	dep        *fdep.Dep
	filedep    *fdep.FileDep
	tc_default TypeConverter

	imports map[string]string

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

	return &Generator{
		Buffer:  new(bytes.Buffer),
		dep:     dep,
		filedep: filedep,
		imports: make(map[string]string),
	}, nil
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

func (g *Generator) IsFileGowrap(filedep *fdep.FileDep) bool {
	if filedep.DepType != fdep.DepType_Own {
		return false
	}

	if o := filedep.ProtoFile.FindOption("fproto_gowrap.wrap"); o != nil {
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
	go_alias := g.FileDep(nil, "", false)

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
	if !g.GenerateComment(message.Comment) {
		g.GenerateCommentLine("MESSAGE: ", msgscopedpbnamestr)
	}

	g.P("type ", structName, " struct {")
	g.In()

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
			g.GenerateComment(xfld.Comment)

			tc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			var type_prefix string
			if xfld.Repeated {
				type_prefix = "[]"
			}

			g.P(CamelCase(xfld.Name), " ", type_prefix, tc_gowrap.TypeName(g, TNT_FIELD_DEFINITION), field_tag.OutputWithSpace())
		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype
			g.GenerateComment(xfld.Comment)

			tc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}
			keytc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.KeyType)
			if err != nil {
				return err
			}

			g.P(CamelCase(xfld.Name), " map[", keytc_gowrap.TypeName(g, TNT_TYPENAME), "]", tc_gowrap.TypeName(g, TNT_TYPENAME), field_tag.OutputWithSpace())
		case *fproto.OneofFieldElement:
			// fieldname isSTRUCT_ONEOF
			g.GenerateComment(xfld.Comment)

			ooscopedname := append(msgscopedname, CamelCase(xfld.Name))
			ooscopednamestr := CamelCaseSlice(ooscopedname)

			g.P(CamelCase(xfld.Name), " is", ooscopednamestr, field_tag.OutputWithSpace())
		}
	}

	g.Out()
	g.P("}")
	g.P()

	//
	// func MyMessage_Import(s *go_package.MyMessage) (*MyMessage, error)
	//
	g.GenerateCommentLine("IMPORT: ", msgscopedpbnamestr)

	g.P("func ", structName, "_Import(s *", go_alias, ".", structName, ") (*", structName, ", error) {")
	g.In()

	g.P("if s == nil {")
	g.In()
	g.P("return nil, nil")
	g.Out()
	g.P("}")
	g.P()

	g.P("var err error")
	g.P("ret := &", structName, "{}")

	for _, fld := range message.Fields {
		g.P("// ", msgscopedpbnamestr, ".", fld.FieldName())

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
				g.P("for _, ms := range s.", CamelCase(xfld.Name), " {")
				g.In()
				g.P("var msi ", tc_gowrap.TypeName(g, TNT_TYPENAME))

				source_field = "ms"
				dest_field = "msi"
			}

			check_error, err := tc_gowrap.GenerateImport(g, source_field, dest_field, "err")
			if err != nil {
				return err
			}
			if check_error {
				g.GenerateErrorCheck("&" + structName + "{}")
			}

			if xfld.Repeated {
				g.P("ret.", CamelCase(xfld.Name), " = append(ret.", CamelCase(xfld.Name), ", msi)")

				g.Out()
				g.P("}")
			}
		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype

			tc_gowrap, err := g.GetGowrapType(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			g.P("for msidx, ms := range s.", CamelCase(xfld.Name), " {")
			g.In()
			g.P("var msi ", tc_gowrap.TypeName(g, TNT_TYPENAME))

			check_error, err := tc_gowrap.GenerateImport(g, "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.GenerateErrorCheck("&" + structName + "{}")
			}

			g.P("ret.", CamelCase(xfld.Name), "[msidx] = msi")

			g.Out()
			g.P("}")
		case *fproto.OneofFieldElement:
			g.P("switch en := s.", CamelCase(xfld.Name), ".(type) {")

			//ooscopedname := append(msgscopedname, CamelCase(xfld.Name))
			//ooscopednamestr := CamelCaseSlice(ooscopedname)

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oofldscopedname := append(msgscopedname, CamelCase(xoofld.Name))
					oofldscopednamestr := CamelCaseSlice(oofldscopedname)

					g.P("case *", go_alias, ".", oofldscopednamestr, ":")
					g.In()

					g.P("ret.", CamelCase(xfld.Name), ", err = ", oofldscopednamestr, "_Import(en)")

					g.Out()
				}
			}

			g.P("}")

			g.GenerateErrorCheck("&" + structName + "{}")
		}
	}

	g.P("return ret, err")

	g.Out()
	g.P("}")

	g.P()

	//
	// func (m *MyMessage) Export() (*go_package.MyMessage, error)
	//
	g.GenerateCommentLine("EXPORT: ", msgscopedpbnamestr)

	g.P("func (m *", structName, ") Export() (*", go_alias, ".", structName, ", error) {")
	g.In()

	g.P("if m == nil {")
	g.In()
	g.P("return nil, nil")
	g.Out()
	g.P("}")
	g.P()

	g.P("var err error")
	g.P("ret := &", go_alias, ".", structName, "{}")

	for _, fld := range message.Fields {
		g.P("// ", msgscopedpbnamestr, ".", fld.FieldName())
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
				g.P("for _, ms := range m.", CamelCase(xfld.Name), " {")
				g.In()
				g.P("var msi ", tc_go.TypeName(g, TNT_TYPENAME))

				source_field = "ms"
				dest_field = "msi"
			}

			check_error, err := tc_gowrap.GenerateExport(g, source_field, dest_field, "err")
			if err != nil {
				return err
			}
			if check_error {
				g.GenerateErrorCheck("&" + go_alias + "." + structName + "{}")
			}

			if xfld.Repeated {
				g.P("ret.", CamelCase(xfld.Name), " = append(ret.", CamelCase(xfld.Name), ", msi)")

				g.Out()
				g.P("}")
			}

		case *fproto.MapFieldElement:
			// fieldname map[keytype]fieldtype

			tc_gowrap, tc_go, err := g.GetBothTypes(msgscopednamestr, xfld.Type)
			if err != nil {
				return err
			}

			g.P("for msidx, ms := range m.", CamelCase(xfld.Name), " {")
			g.In()
			g.P("var msi ", tc_go.TypeName(g, TNT_TYPENAME))

			check_error, err := tc_gowrap.GenerateExport(g, "ms", "msi", "err")
			if err != nil {
				return err
			}
			if check_error {
				g.GenerateErrorCheck("&" + go_alias + "." + structName + "{}")
			}

			g.P("ret.", CamelCase(xfld.Name), "[msidx] = msi")

			g.Out()
			g.P("}")
		case *fproto.OneofFieldElement:
			g.P("switch en := m.", CamelCase(xfld.Name), ".(type) {")

			//ooscopedname := append(msgscopedname, CamelCase(xfld.Name))
			//ooscopednamestr := CamelCaseSlice(ooscopedname)

			for _, oofld := range xfld.Fields {
				switch xoofld := oofld.(type) {
				case *fproto.FieldElement:
					oofldscopedname := append(msgscopedname, CamelCase(xoofld.Name))
					oofldscopednamestr := CamelCaseSlice(oofldscopedname)

					g.P("case *", oofldscopednamestr, ":")
					g.In()

					g.P("ret.", CamelCase(xfld.Name), ", err = ", "en.Export()")

					g.Out()
				}
			}

			g.P("}")

			g.GenerateErrorCheck("&" + go_alias + "." + structName + "{}")
		}
	}

	g.P("return ret, err")

	g.Out()
	g.P("}")

	g.P()

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
	go_alias := g.FileDep(nil, "", false)

	//
	// type MyEnum = go_package.Enum
	//
	if !g.GenerateComment(enum.Comment) {
		g.GenerateCommentLine("ENUM: ", enumscopedpbnamestr)
	}

	g.P("type ", enumName, " = ", go_alias, ".", enumName)
	g.P()
	g.P("const (")
	g.In()

	for _, ec := range enum.EnumConstants {
		// MyEnumConstant MyEnum = go_package.MyEnumConstant
		var encscopedname []string
		if len(scope) == 0 {
			encscopedname = append(enumscopedname, CamelCase(ec.Name))
		} else {
			encscopedname = append(scope, CamelCase(ec.Name))
		}

		encscopednamestr := CamelCaseSlice(encscopedname)

		g.GenerateComment(ec.Comment)

		g.P(encscopednamestr, " ", enumName, " = ", go_alias, ".", encscopednamestr)
	}

	g.Out()
	g.P(")")
	g.P()

	// var MyEnum_name = go_package.MyEnum_name
	g.P("var ", enumName, "_name = ", go_alias, ".", enumName, "_name")

	// var MyEnum_value = go_package.MyEnum_value
	g.P("var ", enumName, "_value = ", go_alias, ".", enumName, "_value")

	g.P()

	return nil
}

func (g *Generator) generateOneOf(oneof *fproto.OneofFieldElement) error {
	// CUSTOMIZER
	cz := &wrapCustomizers{g.Customizers}

	// Get the oneof scope on the current file as an array
	scope := g.GetScope(oneof)
	scopestr := CamelCaseSlice(scope)

	// build aliases to the original type
	go_alias := g.FileDep(nil, "", false)

	ooscopedname := append(scope, CamelCase(oneof.Name))
	//ooscopednamestr := strings.Join(ooscopedname, ".")
	ooscopedpbnamestr := strings.Join(append(scope, oneof.Name), ".")

	oneofName := CamelCaseSlice(ooscopedname)

	// type isSTRUCT_ONEOF interface {
	//		isSTRUCT_ONEOF()
	// }

	if !g.GenerateComment(oneof.Comment) {
		g.GenerateCommentLine("ONEOF: ", ooscopedpbnamestr)
	}

	g.P("type is", oneofName, " interface {")
	g.In()
	g.P("is", oneofName, "()")
	g.Out()
	g.P("}")
	g.P()

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

			if !g.GenerateComment(xoofld.Comment) {
				g.GenerateCommentLine("ONEOF Field: ", oofldscopedpbnamestr)
			}

			g.P("type ", ooFldName, " struct {")
			g.In()

			// fieldname fieldtype
			tc_gowrap, err := g.GetGowrapType(scopestr, xoofld.Type)
			if err != nil {
				return err
			}

			g.P(CamelCase(xoofld.Name), " ", tc_gowrap.TypeName(g, TNT_TYPENAME), field_tag.OutputWithSpace())

			g.Out()
			g.P("}")
			g.P()

			// func (*STRUCT_ONEOFFIELD) isSTRUCT_ONEOF()  {}

			g.P("func (*", ooFldName, ") is", oneofName, "() {}")
			g.P()

			//
			// func (*STRUCT_ONEOFFIELD) Import()  {}
			//
			g.GenerateCommentLine("IMPORT: ", oofldscopedpbnamestr)

			g.P("func ", ooFldName, "_Import(s *", go_alias, ".", ooFldName, ") (*", ooFldName, ", error) {")
			g.In()

			g.P("var err error")
			g.P("ret := &", ooFldName, "{}")

			tcoo_gowrap, err := g.GetGowrapType(scopestr, xoofld.Type)
			if err != nil {
				return err
			}

			check_error, err := tcoo_gowrap.GenerateImport(g, "s."+CamelCase(xoofld.Name), "ret."+CamelCase(xoofld.Name), "err")
			if err != nil {
				return err
			}
			if check_error {
				g.GenerateErrorCheck("nil")
			}

			g.P("return ret, err")
			g.Out()
			g.P("}")
			g.P()

			//
			// func (*STRUCT_ONEOFFIELD) Export()  {}
			//
			g.GenerateCommentLine("EXPORT: ", oofldscopedpbnamestr)

			g.P("func (o *", ooFldName, ") Export() (*", go_alias, ".", ooFldName, ", error) {")
			g.In()

			g.P("var err error")
			g.P("ret := &", go_alias, ".", ooFldName, "{}")

			check_error, err = tcoo_gowrap.GenerateExport(g, "o."+CamelCase(xoofld.Name), "ret."+CamelCase(xoofld.Name), "err")
			if err != nil {
				return err
			}
			if check_error {
				g.GenerateErrorCheck("nil")
			}

			g.P("return ret, err")
			g.Out()
			g.P("}")
			g.P()
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

// Declares a dependency and returns the alias to be used on this file.
func (g *Generator) Dep(imp string, defalias string) string {
	var alias string
	var ok bool
	if alias, ok = g.imports[imp]; ok {
		return alias
	}

	if defalias == "" {
		defalias = path.Base(imp)
	}

	defalias = strings.Replace(defalias, ".", "_", -1)

	alias = defalias
	aliasct := 0
	aliasok := false
	for !aliasok {
		aliasok = true

		for _, a := range g.imports {
			if a == alias {
				aliasct++
				alias = fmt.Sprintf("%s%d", defalias, aliasct)
				aliasok = false
			}
		}

		if aliasok {
			break
		}
	}

	g.imports[imp] = alias
	return alias
}

// Declares a dependency using a FileDep.
func (g *Generator) FileDep(filedep *fdep.FileDep, defalias string, is_gowrap bool) string {
	if filedep == nil {
		filedep = g.filedep
	}
	var p string
	if is_gowrap && !filedep.IsSamePackage(g.filedep) && g.IsFileGowrap(filedep) {
		p = g.GoWrapPackage(filedep)
	} else {
		p = filedep.GoPackage()
	}
	return g.Dep(p, defalias)
}

// Returns the generated file as a string.
func (g *Generator) Output(w io.Writer) error {
	// write in temporary buffer
	tmp := new(bytes.Buffer)

	// Generate header and imports last, though they appear first in the output.
	rem := g.Buffer
	g.Buffer = new(bytes.Buffer)

	g.generateHeader()
	g.generateImports()

	// write headers / imports
	_, err := tmp.Write(g.Bytes())
	if err != nil {
		g.Buffer = rem
		return err
	}

	// write previous content
	_, err = tmp.Write(rem.Bytes())
	if err != nil {
		g.Buffer = rem
		return err
	}

	// restore buffer
	g.Buffer = rem

	// Reformat generated code.
	fset := token.NewFileSet()
	raw := tmp.Bytes()
	ast, err := parser.ParseFile(fset, "", tmp, parser.ParseComments)
	if err != nil {
		// Print out the bad code with line numbers.
		// This should never happen in practice, but it can while changing generated code,
		// so consider this a debugging aid.
		var src bytes.Buffer
		s := bufio.NewScanner(bytes.NewReader(raw))
		for line := 1; s.Scan(); line++ {
			fmt.Fprintf(&src, "%5d\t%s\n", line, s.Bytes())
		}
		return errors.New(fmt.Sprint("bad Go source code was generated:", err.Error(), "\n"+src.String()))
	}

	// write into the requested io.Writer
	err = (&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(w, fset, ast)
	if err != nil {
		return fmt.Errorf("generated Go source code could not be reformatted:", err.Error())
	}

	return nil
}

func (g *Generator) generateHeader() {
	p := baseName(g.GoWrapPackage(g.filedep))

	g.P("// Code generated by fproto-gowrap2. DO NOT EDIT.")
	g.P("// source file: ", g.filedep.FilePath)

	g.P("package ", p)
	g.P()
}

func (g *Generator) generateImports() {
	if len(g.imports) > 0 {
		g.P("import (")
		g.In()

		// loop imports in ascending order
		keys := make([]string, 0)
		for k, _ := range g.imports {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, in := range keys {
			g.P(g.imports[in], ` "`, in, `"`)
		}
		g.Out()
		g.P(")")

		g.P()
	}
}

func (g *Generator) GenerateComment(comment *fproto.Comment) bool {
	if comment != nil && len(comment.Lines) > 0 {
		cstr := "//"
		if comment.ExtraSlash {
			cstr += "/"
		}
		for _, dl := range comment.Lines {
			g.P(cstr, " ", strings.TrimSpace(dl))
		}
		return true
	}
	return false
}

// Generates a multi-line comment starting and ending with an empty line
func (g *Generator) GenerateCommentLine(str ...string) {
	if len(str) > 0 {
		g.P("//")
		p := []interface{}{"// "}
		for _, s := range str {
			p = append(p, s)
		}
		g.P(p...)
		g.P("//")
	}
}

// Returns the expected output file path and name
func (g *Generator) Filename() string {
	p := g.GoWrapPackage(g.filedep)
	return path.Join(p, strings.TrimSuffix(path.Base(g.filedep.FilePath), path.Ext(g.filedep.FilePath))+".gwpb.go")
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

// P prints the arguments to the generated output.  It handles strings and int32s, plus
// handling indirections because they may be *string, etc.
func (g *Generator) P(str ...interface{}) {
	g.WriteString(g.indent)
	for _, v := range str {
		switch s := v.(type) {
		case string:
			g.WriteString(s)
		case *string:
			g.WriteString(*s)
		case bool:
			fmt.Fprintf(g, "%t", s)
		case *bool:
			fmt.Fprintf(g, "%t", *s)
		case int:
			fmt.Fprintf(g, "%d", s)
		case *int32:
			fmt.Fprintf(g, "%d", *s)
		case *int64:
			fmt.Fprintf(g, "%d", *s)
		case float64:
			fmt.Fprintf(g, "%g", s)
		case *float64:
			fmt.Fprintf(g, "%g", *s)
		default:
			panic(fmt.Sprintf("unknown type in printer: %T", v))
		}
	}
	g.WriteByte('\n')
}

// In Indents the output one tab stop.
func (g *Generator) In() { g.indent += "\t" }

// Out unindents the output one tab stop.
func (g *Generator) Out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[1:]
	}
}

func (g *Generator) GenerateErrorCheck(extraRetVal string) {
	g.P("if err != nil {")
	g.In()
	if extraRetVal != "" {
		g.P("return ", extraRetVal, ", err")
	} else {
		g.P("return err")
	}
	g.Out()
	g.P("}")
}
