package fproto_phpwrap_time

import (
	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto-wrap/phpwrap"
)

const (
	TCID_TIME     string = "de426778-4912-450e-b531-8e83f8056ce3"
	TCID_NULLTIME string = "11ed386a-0cee-4590-9090-b03c0325e13a"
)

// Converts between google.protobuf.Timestamp and time.Time
type TypeConverterPlugin_Time struct {
}

func (t *TypeConverterPlugin_Time) GetTypeConverter(tp *fdep.DepType) fproto_phpwrap.TypeConverter {
	if tp.FileDep.FilePath == "google/protobuf/timestamp.proto" &&
		tp.FileDep.ProtoFile.PackageName == "google.protobuf" &&
		tp.Name == "Timestamp" {
		return &TypeConverter_Time{}
	}
	/*
		if tp.FileDep.FilePath == "github.com/RangelReale/fproto-wrap/time.proto" &&
			tp.FileDep.ProtoFile.PackageName == "fproto_wrap" &&
			tp.Name == "NullTime" {
			return &TypeConverter_NullTime{}
		}
	*/
	return nil
}

//
// Time
// Converts between google.protobuf.Timestamp and \DateTime
//

type TypeConverter_Time struct {
}

func (t *TypeConverter_Time) TCID() string {
	return TCID_TIME
}

func (t *TypeConverter_Time) TypeName(g *fproto_phpwrap.GeneratorFile, tntype fproto_phpwrap.TypeNameType) string {

	switch tntype {
	case fproto_phpwrap.TNT_NS_TYPENAME:
		return "\\DateTime"
	}

	return "DateTime"
}

func (t *TypeConverter_Time) IsScalar() bool {
	return false
}

func (t *TypeConverter_Time) GenerateImport(g *fproto_phpwrap.GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	g.P(varDest, " = ", varSrc, "->toDateTime();")

	return true, nil
}

func (t *TypeConverter_Time) GenerateExport(g *fproto_phpwrap.GeneratorFile, varSrc string, varDest string, varError string) (generated bool, err error) {
	g.P(varDest, " = new \\Google\\Protobuf\\Timestamp();")
	g.P(varDest, "->fromDateTime(", varSrc, ");")

	return true, nil
}
