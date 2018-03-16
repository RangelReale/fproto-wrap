package fproto_gowrap

import (
	"github.com/RangelReale/fproto"
)

// Interface to allow customizing various aspects of the output
type Customizer interface {
	// Allows adding tags for a generated struct field. Use the currentTag field to read and edit the tags.
	GetTag(g *Generator, currentTag *StructTag, parentItem fproto.FProtoElement, item fproto.FProtoElement) error

	// Allows code generation after all the protofile data was generated.
	GenerateCode(g *Generator) error

	// Allows service code generation after all the protofile services were generated.
	GenerateServiceCode(g *Generator) error
}

// Wraps a list of customizers
type wrapCustomizers struct {
	customizers []Customizer
}

func (c *wrapCustomizers) GetTag(g *Generator, currentTag *StructTag, parentItem fproto.FProtoElement, item fproto.FProtoElement) error {
	for _, cz := range c.customizers {
		err := cz.GetTag(g, currentTag, parentItem, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *wrapCustomizers) GenerateCode(g *Generator) error {
	for _, cz := range c.customizers {
		err := cz.GenerateCode(g)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *wrapCustomizers) GenerateServiceCode(g *Generator) error {
	for _, cz := range c.customizers {
		err := cz.GenerateServiceCode(g)
		if err != nil {
			return err
		}
	}
	return nil
}
