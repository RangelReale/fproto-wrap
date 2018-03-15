package fproto_wrap_jsontag

import (
	"fmt"

	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto-wrap/gowrap"
	"github.com/RangelReale/fproto/fdep"
)

// Adds a json tag to all struct fields, using snake case formatting
type Customizer_JSONTag struct {
}

func (c *Customizer_JSONTag) GetTag(g *fproto_gowrap.Generator, currentTag *fproto_gowrap.StructTag, parentItem fproto.FProtoElement, item fproto.FProtoElement) error {
	switch fitem := item.(type) {
	case fproto.FieldElementTag:
		jsonopt := fitem.FindOption("fproto_wrap.jsontag.tag_disable")
		if jsonopt != nil && jsonopt.Value.String() == "true" {
			currentTag.Set("json", "-")
		} else {
			fieldname := fproto_gowrap.SnakeCase(fitem.FieldName())
			fnopt := fitem.FindOption("fproto_wrap.jsontag.tag_fieldname")
			if fnopt != nil && fnopt.Value.String() != "" {
				fieldname = fnopt.Value.String()
			}
			currentTag.Set("json", fmt.Sprintf("%s,omitempty", fieldname))
		}
	}
	return nil
}

func (c *Customizer_JSONTag) GenerateCode(g *fproto_gowrap.Generator, dep *fdep.Dep, filedep *fdep.FileDep) error {
	return nil
}

func (c *Customizer_JSONTag) GenerateServiceCode(g *fproto_gowrap.Generator, dep *fdep.Dep, filedep *fdep.FileDep) error {
	return nil
}
