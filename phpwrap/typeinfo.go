package fproto_phpwrap

type TypeInfo interface {
	Source() TypeNamer
	Converter() TypeConverter
}
