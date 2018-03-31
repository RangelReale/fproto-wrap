package fproto_gowrap

type TypeInfo interface {
	Source() TypeNamer
	Wrapped() TypeConverter
}

type TypeInfo_Default struct {
	source  TypeNamer
	wrapped TypeConverter
}

func (t *TypeInfo_Default) Source() TypeNamer {
	return t.source
}

func (t *TypeInfo_Default) Wrapped() TypeConverter {
	return t.wrapped
}
