package fproto_phpwrap_uuid

import (
	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto-wrap/phpwrap"
)

const (
	TCID_UUID     string = "5f381deb-11a8-4ab7-ae80-7501c1dabd95"
	TCID_NULLUUID string = "d0b80892-1684-45a2-a30a-0e794c51a42a"
)

//
// UUID
// Converts between fproto_wrap.UUID and \Ramsey\Uuid\Uuid
//

type TypeConverterPlugin_UUID struct {
}

func (t *TypeConverterPlugin_UUID) GetTypeConverter(tp *fdep.DepType) fproto_phpwrap.TypeConverter {
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
// Converts between fproto_wrap.UUID and \Ramsey\Uuid\Uuid
//

type TypeConverter_UUID struct {
}

func (t *TypeConverter_UUID) TCID() string {
	return TCID_UUID
}

func (t *TypeConverter_UUID) TypeName(g *fproto_phpwrap.GeneratorFile, tntype fproto_phpwrap.TypeConverterTypeNameType) string {

	switch tntype {
	case fproto_phpwrap.TNT_NS_WRAPNAME:
		return "\\Ramsey\\Uuid\\Uuid"
	case fproto_phpwrap.TNT_NS_SOURCENAME:
		return "\\Fproto_wrap\\UUID"
	}

	return "Uuid"
}

func (t *TypeConverter_UUID) IsScalar() bool {
	return false
}

func (t *TypeConverter_UUID) GenerateImport(g *fproto_phpwrap.GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	g.P(varDest, " = \\Ramsey\\Uuid\\Uuid::fromString(", varSrc, "->getValue());")

	return true, nil
}

func (t *TypeConverter_UUID) GenerateExport(g *fproto_phpwrap.GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	g.P(varDest, " = new \\Fproto_wrap\\UUID();")
	g.P(varDest, "->setValue(", varSrc, "->toString());")

	return true, nil
}

//
// NullUUID
// Converts between fproto_wrap.NullUUID and \Ramsey\Uuid\Uuid
//

type TypeConverter_NullUUID struct {
}

func (t *TypeConverter_NullUUID) TCID() string {
	return TCID_UUID
}

func (t *TypeConverter_NullUUID) TypeName(g *fproto_phpwrap.GeneratorFile, tntype fproto_phpwrap.TypeConverterTypeNameType) string {

	switch tntype {
	case fproto_phpwrap.TNT_NS_WRAPNAME:
		return "\\Ramsey\\Uuid\\Uuid"
	case fproto_phpwrap.TNT_NS_SOURCENAME:
		return "\\Fproto_wrap\\UUID"
	}

	return "Uuid"
}

func (t *TypeConverter_NullUUID) IsScalar() bool {
	return false
}

func (t *TypeConverter_NullUUID) GenerateImport(g *fproto_phpwrap.GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	g.P(varDest, " = null;")
	g.P("if (", varSrc, "->getValue() != '') {")
	g.In()
	g.P(varDest, " = \\Ramsey\\Uuid\\Uuid::fromString(", varSrc, "->getValue());")
	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_NullUUID) GenerateExport(g *fproto_phpwrap.GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	g.P(varDest, " = new \\Fproto_wrap\\UUID();")
	g.P(varDest, "->setValue(", varSrc, "->toString());")

	return true, nil
}
