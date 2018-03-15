package uuid

import (
	"fmt"

	"github.com/RangelReale/fproto-gowrap"
	"github.com/RangelReale/fproto/fdep"
)

//
// UUID
// Converts between fproto_gowrap.tc.uuid.UUID and github.com/RangelReale/go.uuid UUID
//

type TypeConverterPlugin_UUID struct {
}

func (t *TypeConverterPlugin_UUID) GetTypeConverter(tp *fdep.DepType) fproto_gowrap.TypeConverter {
	if tp.FileDep.FilePath == "github.com/RangelReale/fproto-gowrap/tc/uuid/uuid.proto" &&
		tp.FileDep.ProtoFile.PackageName == "fproto_gowrap.tc.uuid" &&
		tp.Name == "UUID" {
		return &TypeConverter_UUID{}
	}
	if tp.FileDep.FilePath == "github.com/RangelReale/fproto-gowrap/tc/uuid/uuid.proto" &&
		tp.FileDep.ProtoFile.PackageName == "fproto_gowrap.tc.uuid" &&
		tp.Name == "NullUUID" {
		return &TypeConverter_NullUUID{}
	}
	return nil
}

//
// UUID
// Converts between fproto_gowrap.tc.uuid.UUID and github.com/RangelReale/go.uuid UUID
//

type TypeConverter_UUID struct {
}

func (t *TypeConverter_UUID) TypeName(g *fproto_gowrap.Generator, tntype fproto_gowrap.TypeConverterTypeNameType) string {
	alias := g.Dep("github.com/RangelReale/go.uuid", "uuid")

	switch tntype {
	case fproto_gowrap.TNT_EMPTYVALUE, fproto_gowrap.TNT_EMPTYORNILVALUE:
		return fmt.Sprintf("%s.%s{}", alias, "UUID")
	}

	return fmt.Sprintf("%s.%s", alias, "UUID")
}

func (t *TypeConverter_UUID) IsPointer() bool {
	return false
}

func (t *TypeConverter_UUID) GenerateImport(g *fproto_gowrap.Generator, varSrc string, varDest string, varError string) (checkError bool, err error) {
	alias := g.Dep("github.com/RangelReale/go.uuid", "uuid")

	g.P("if ", varSrc, " != nil {")
	g.In()
	g.P(varDest, ", err = ", alias, ".FromString(", varSrc, ".Value)")
	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_UUID) GenerateExport(g *fproto_gowrap.Generator, varSrc string, varDest string, varError string) (checkError bool, err error) {
	tc_go, err := g.GetGoType("", "fproto_gowrap.tc.uuid.UUID")
	if err != nil {
		return false, err
	}

	g.P(varDest, " = ", tc_go.TypeName(g, fproto_gowrap.TNT_EMPTYVALUE))
	g.P(varDest, ".Value = ", varSrc, ".String()")
	return false, nil
}

//
// NullUUID
// Converts between fproto_gowrap.tc.uuid.UUID and github.com/RangelReale/go.uuid UUID
//

type TypeConverter_NullUUID struct {
}

func (t *TypeConverter_NullUUID) TypeName(g *fproto_gowrap.Generator, tntype fproto_gowrap.TypeConverterTypeNameType) string {
	alias := g.Dep("github.com/RangelReale/go.uuid", "uuid")

	switch tntype {
	case fproto_gowrap.TNT_EMPTYVALUE, fproto_gowrap.TNT_EMPTYORNILVALUE:
		return fmt.Sprintf("%s.%s{}", alias, "NullUUID")
	}

	return fmt.Sprintf("%s.%s", alias, "NullUUID")
}

func (t *TypeConverter_NullUUID) IsPointer() bool {
	return false
}

func (t *TypeConverter_NullUUID) GenerateImport(g *fproto_gowrap.Generator, varSrc string, varDest string, varError string) (checkError bool, err error) {
	alias := g.Dep("github.com/RangelReale/go.uuid", "uuid")

	g.P("if ", varSrc, " != nil {")
	g.In()

	g.P(varDest, " = ", alias, ".NullUUID{}")

	g.P("if ", varSrc, ".Valid {")
	g.In()

	g.P(varDest, ".Valid = true")
	g.P(varDest, ".UUID, err = ", alias, ".FromString(", varSrc, ".Value)")

	g.Out()
	g.P("}")

	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_NullUUID) GenerateExport(g *fproto_gowrap.Generator, varSrc string, varDest string, varError string) (checkError bool, err error) {
	tc_go, err := g.GetGoType("", "fproto_gowrap.tc.uuid.NullUUID")
	if err != nil {
		return false, err
	}

	g.P(varDest, " = ", tc_go.TypeName(g, fproto_gowrap.TNT_EMPTYVALUE))
	g.P("if ", varSrc, ".Valid {")
	g.In()
	g.P(varDest, ".Value = ", varSrc, ".UUID.String()")
	g.P(varDest, ".Valid = ", varSrc, ".Valid")
	g.Out()
	g.P("}")
	return false, nil
}
