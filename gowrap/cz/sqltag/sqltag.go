package sqltag

import (
	"fmt"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto-wrap/gowrap"
	"github.com/RangelReale/fproto/fdep"
)

// Adds a sql tag to all struct fields, using snake case formatting
type Customizer_SQLTag struct {
}

func (c *Customizer_SQLTag) GetTag(g *fproto_gowrap.Generator, currentTag *fproto_gowrap.StructTag, parentItem fproto.FProtoElement, item fproto.FProtoElement) error {
	switch fitem := item.(type) {
	case fproto.FieldElementTag:
		sqlopt := fitem.FindOption("fproto_wrap.sqltag.tag_disable")
		if sqlopt != nil && sqlopt.Value == "true" {
			currentTag.Set("sql", "-")
		} else {
			fieldname := fproto_gowrap.SnakeCase(fitem.FieldName())
			fnopt := fitem.FindOption("fproto_wrap.sqltag.tag_fieldname")
			if fnopt != nil && fnopt.Value != "" {
				fieldname = fnopt.Value
			}
			currentTag.Set("sql", fmt.Sprintf("%s,omitempty", fieldname))
		}
	}
	return nil
}

func (c *Customizer_SQLTag) GenerateCode(g *fproto_gowrap.Generator, dep *fdep.Dep, filedep *fdep.FileDep) error {
	return nil
}

func (c *Customizer_SQLTag) GenerateServiceCode(g *fproto_gowrap.Generator, dep *fdep.Dep, filedep *fdep.FileDep) error {
	return nil
}
