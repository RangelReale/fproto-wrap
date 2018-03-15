package jsonobject

import (
	"github.com/RangelReale/fproto-gowrap"
	"github.com/RangelReale/fproto/fdep"
)

//
// JSONObject
// Converts between fproto_gowrap.tc.json.JSON and map[string]interface{}
//

type TypeConverterPlugin_JSONObject struct {
}

func (t *TypeConverterPlugin_JSONObject) GetTypeConverter(tp *fdep.DepType) fproto_gowrap.TypeConverter {
	if tp.FileDep.FilePath == "github.com/RangelReale/fproto-gowrap/tc/jsonobject/jsonobject.proto" &&
		tp.FileDep.ProtoFile.PackageName == "fproto_gowrap.tc.jsonobject" &&
		tp.Name == "JSONObject" {
		return &TypeConverter_JSONObject{}
	}
	return nil
}

//
// JSONObject
// Converts between fproto_gowrap.tc.jsonobject.JSONObject and interface{}
//

type TypeConverter_JSONObject struct {
}

func (t *TypeConverter_JSONObject) TypeName(g *fproto_gowrap.Generator, tntype fproto_gowrap.TypeConverterTypeNameType) string {
	return "interface{}"
}

func (t *TypeConverter_JSONObject) IsPointer() bool {
	return true
}

func (t *TypeConverter_JSONObject) GenerateImport(g *fproto_gowrap.Generator, varSrc string, varDest string, varError string) (checkError bool, err error) {
	alias := g.Dep("encoding/json", "json")

	g.P("if ", varSrc, " != nil && ", varSrc, ".Value != \"\" {")
	g.In()

	g.P("jtemp := make(map[string]interface{})")
	g.P("err = ", alias, ".Unmarshal([]byte(", varSrc, ".Value), jtemp)")
	g.P("if err != nil {")
	g.In()
	g.P(varDest, " = jtemp")
	g.Out()
	g.P("}")

	g.Out()
	g.P("}")

	return true, nil
}

func (t *TypeConverter_JSONObject) GenerateExport(g *fproto_gowrap.Generator, varSrc string, varDest string, varError string) (checkError bool, err error) {
	alias := g.Dep("encoding/json", "json")

	tc_go, err := g.GetGoType("", "fproto_gowrap.tc.jsonobject.JSONObject")
	if err != nil {
		return false, err
	}

	g.P("if ", varSrc, " != nil {")
	g.In()

	g.P("var jtemp []byte")
	g.P(varDest, " = ", tc_go.TypeName(g, fproto_gowrap.TNT_EMPTYVALUE))

	g.P("jtemp, err = ", alias, ".Marshal(", varSrc, ")")
	g.P("if err != nil {")
	g.In()
	g.P(varDest, ".Value = string(jtemp)")
	g.Out()
	g.P("}")
	g.Out()

	g.Out()
	g.P("}")

	return true, nil
}
