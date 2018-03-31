package fproto_gowrap_time

import (
	"fmt"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto-wrap/gowrap"
)

const (
	TCID_TIME     string = "de426778-4912-450e-b531-8e83f8056ce3"
	TCID_NULLTIME string = "11ed386a-0cee-4590-9090-b03c0325e13a"
)

// Converts between google.protobuf.Timestamp and time.Time
type TypeConverterPlugin_Time struct {
}

func (t *TypeConverterPlugin_Time) GetTypeConverter(tp *fdep.DepType) fproto_gowrap.TypeConverter {
	if tp.FileDep.FilePath == "google/protobuf/timestamp.proto" &&
		tp.FileDep.ProtoFile.PackageName == "google.protobuf" &&
		tp.Name == "Timestamp" {
		return &TypeConverter_Time{}
	}
	if tp.FileDep.FilePath == "github.com/RangelReale/fproto-wrap/time.proto" &&
		tp.FileDep.ProtoFile.PackageName == "fproto_wrap" &&
		tp.Name == "NullTime" {
		return &TypeConverter_NullTime{}
	}
	return nil
}

// Converter
type TypeConverter_Time struct {
}

func (t *TypeConverter_Time) TCID() string {
	return TCID_TIME
}

func (t *TypeConverter_Time) TypeName(g *fproto_gowrap.GeneratorFile, tntype fproto_gowrap.TypeNameType) string {
	alias := g.Dep("time", "time")
	return fmt.Sprintf("%s.%s", alias, "Time")
}

func (t *TypeConverter_Time) IsPointer() bool {
	return false
}

func (t *TypeConverter_Time) GenerateImport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	pb_alias := g.Dep("github.com/golang/protobuf/ptypes", "pb_types")

	g.P("if ", varSrc, " != nil {")
	g.In()
	g.P(varDest, ", err = ", pb_alias, ".Timestamp(", varSrc, ")")
	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_Time) GenerateExport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	pb_alias := g.Dep("github.com/golang/protobuf/ptypes", "pb_types")

	g.P(varDest, ", err = ", pb_alias, ".TimestampProto(", varSrc, ")")

	return true, nil
}

// Converter Null
type TypeConverter_NullTime struct {
}

func (t *TypeConverter_NullTime) TCID() string {
	return TCID_NULLTIME
}

func (t *TypeConverter_NullTime) TypeName(g *fproto_gowrap.GeneratorFile, tntype fproto_gowrap.TypeNameType) string {
	alias := g.Dep("github.com/RangelReale/fproto-wrap/gowrap/tc/time/ptypes", "time_ptypes")
	return fmt.Sprintf("%s.%s", alias, "NullTime")
}

func (t *TypeConverter_NullTime) IsPointer() bool {
	return false
}

func (t *TypeConverter_NullTime) GenerateImport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	pb_alias := g.Dep("github.com/golang/protobuf/ptypes", "pb_types")
	alias := g.Dep("github.com/RangelReale/fproto-wrap/gowrap/tc/time/ptypes", "time_ptypes")

	g.P("if ", varSrc, " != nil {")
	g.In()

	g.P(varDest, " = ", alias, ".NullTime{}")
	g.P("if ", varSrc, ".Valid {")
	g.In()

	g.P(varDest, ".Valid = true")
	g.P(varDest, ".Time, err = ", pb_alias, ".Timestamp(", varSrc, ".Value)")

	g.Out()
	g.P("}")

	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_NullTime) GenerateExport(g *fproto_gowrap.GeneratorFile, varSrc string, varDest string, varError string) (checkError bool, err error) {
	pb_alias := g.Dep("github.com/golang/protobuf/ptypes", "pb_types")
	alias := g.Dep("github.com/RangelReale/fproto-wrap/gowrap/gwproto", "time")

	g.P("if ", varSrc, ".Valid {")
	g.In()

	g.P(varDest, " = &", alias, ".NullTime{}")
	g.P(varDest, ".Valid = true")
	g.P(varDest, ".Value, err = ", pb_alias, ".TimestampProto(", varSrc, ".Time)")

	g.Out()
	g.P("}")

	return true, nil
}
