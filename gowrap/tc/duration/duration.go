package fproto_gowrap_duration

import (
	"fmt"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto-wrap/gowrap"
)

const (
	TCID_DURATION string = "9be62862-ed8b-4469-9c2c-d8fc6bc0054d"
)

// Converts between google.protobuf.Duration and time.Duration
type TypeConverterPlugin_Duration struct {
}

func (t *TypeConverterPlugin_Duration) GetTypeConverter(tp *fdep.DepType) fproto_gowrap.TypeConverter {
	if tp.FileDep.FilePath == "google/protobuf/duration.proto" &&
		tp.FileDep.ProtoFile.PackageName == "google.protobuf" &&
		tp.Name == "Duration" {
		return &TypeConverter_Duration{}
	}
	return nil
}

// Converter
type TypeConverter_Duration struct {
}

func (t *TypeConverter_Duration) TCID() string {
	return TCID_DURATION
}

func (t *TypeConverter_Duration) TypeName(g *fproto_gowrap.GeneratorFile, tntype fproto_gowrap.TypeConverterTypeNameType) string {
	alias := g.Dep("time", "time")
	return fmt.Sprintf("%s.%s", alias, "Duration")
}

func (t *TypeConverter_Duration) IsPointer() bool {
	return false
}

func (t *TypeConverter_Duration) GenerateImport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	pb_alias := g.Dep("github.com/golang/protobuf/ptypes", "pb_types")

	g.P("if ", varSrc, " != nil {")
	g.In()
	g.P(varDest, ", err = ", pb_alias, ".Duration(", varSrc, ")")
	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_Duration) GenerateExport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	pb_alias := g.Dep("github.com/golang/protobuf/ptypes", "pb_types")

	g.P(varDest, " = ", pb_alias, ".DurationProto(", varSrc, ")")

	return false, nil
}
