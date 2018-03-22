package fproto_phpwrap

import "github.com/RangelReale/fproto"

var scalarPhpTypeLookupMap = map[fproto.ScalarType]string{
	fproto.BoolScalar:     "boolean",
	fproto.BytesScalar:    "mixed",
	fproto.DoubleScalar:   "float",
	fproto.FloatScalar:    "float",
	fproto.Fixed32Scalar:  "int",
	fproto.Fixed64Scalar:  "int",
	fproto.Int32Scalar:    "int",
	fproto.Int64Scalar:    "int",
	fproto.Sfixed32Scalar: "int",
	fproto.Sfixed64Scalar: "int",
	fproto.Sint32Scalar:   "int",
	fproto.Sint64Scalar:   "int",
	fproto.StringScalar:   "string",
	fproto.Uint32Scalar:   "int",
	fproto.Uint64Scalar:   "int",
}

func ScalarToPhp(scalar fproto.ScalarType) string {
	return scalarPhpTypeLookupMap[scalar]
}
