package fproto_gowrap_uuid

import (
	"fmt"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto-wrap/gowrap"
)

const (
	TCID_UUID     string = "5f381deb-11a8-4ab7-ae80-7501c1dabd95"
	TCID_NULLUUID string = "d0b80892-1684-45a2-a30a-0e794c51a42a"
)

//
// UUID
// Converts between fproto_wrap.UUID and github.com/RangelReale/go.uuid UUID
//

type TypeConverterPlugin_UUID struct {
}

func (t *TypeConverterPlugin_UUID) GetTypeConverter(tp *fdep.DepType) fproto_gowrap.TypeConverter {
	if tp.FileDep.FilePath == "github.com/RangelReale/fproto-wrap/uuid.proto" &&
		tp.FileDep.ProtoFile.PackageName == "fproto_wrap" &&
		tp.Name == "UUID" {
		return &TypeConverter_UUID{}
	}
	if tp.FileDep.FilePath == "github.com/RangelReale/fproto-wrap/uuid.proto" &&
		tp.FileDep.ProtoFile.PackageName == "fproto_wrap" &&
		tp.Name == "NullUUID" {
		return &TypeConverter_NullUUID{}
	}
	return nil
}

//
// UUID
// Converts between fproto_wrap.UUID and github.com/RangelReale/go.uuid UUID
//

type TypeConverter_UUID struct {
}

func (t *TypeConverter_UUID) TCID() string {
	return TCID_UUID
}

func (t *TypeConverter_UUID) TypeName(g *fproto_gowrap.GeneratorFile, tntype fproto_gowrap.TypeNameType) string {
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

func (t *TypeConverter_UUID) GenerateImport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	alias := g.Dep("github.com/RangelReale/go.uuid", "uuid")

	g.P("if ", varSrc, " != nil {")
	g.In()
	g.P(varDest, ", err = ", alias, ".FromString(", varSrc, ".Value)")
	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_UUID) GenerateExport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	tp, err := g.G().GetDep().FindType("fproto_wrap.UUID")
	if err != nil {
		return false, err
	}
	tsource := g.G().GetTypeSource(tp)

	g.P(varDest, " = ", tsource.TypeName(g, fproto_gowrap.TNT_EMPTYVALUE))
	g.P(varDest, ".Value = ", varSrc, ".String()")
	return false, nil
}

//
// NullUUID
// Converts between fproto_gowrap.tc.uuid.UUID and github.com/RangelReale/go.uuid UUID
//

type TypeConverter_NullUUID struct {
}

func (t *TypeConverter_NullUUID) TCID() string {
	return TCID_NULLUUID
}

func (t *TypeConverter_NullUUID) TypeName(g *fproto_gowrap.GeneratorFile, tntype fproto_gowrap.TypeNameType) string {
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

func (t *TypeConverter_NullUUID) GenerateImport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
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

func (t *TypeConverter_NullUUID) GenerateExport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	tp, err := g.G().GetDep().FindType("fproto_wrap.NullUUID")
	if err != nil {
		return false, err
	}
	tsource := g.G().GetTypeSource(tp)

	g.P(varDest, " = ", tsource.TypeName(g, fproto_gowrap.TNT_EMPTYVALUE))
	g.P("if ", varSrc, ".Valid {")
	g.In()
	g.P(varDest, ".Value = ", varSrc, ".UUID.String()")
	g.P(varDest, ".Valid = ", varSrc, ".Valid")
	g.Out()
	g.P("}")
	return false, nil
}
