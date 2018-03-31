package fproto_gowrap

type TypeInfo interface {
	Source() TypeNamer
	Converter() TypeConverter
}
