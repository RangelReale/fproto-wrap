package fproto_gowrap

import "github.com/RangelReale/fproto"

// Interface to allow customizing various aspects of the output
type Customizer interface {
	// Allows code generation after all the protofile data was generated.
	GenerateCode(g *Generator) error

	// Allows service code generation after all the protofile services were generated.
	GenerateServiceCode(g *Generator) error
}

type Customizer_Tag interface {
	// Allows adding tags for a generated struct field. Use the currentTag field to read and edit the tags.
	GetTag(g *Generator, currentTag *StructTag, parentItem fproto.FProtoElement, item fproto.FProtoElement) error
}

type Customizer_Global interface {
	// Allows generation of files independent of an specific proto file
	GenerateGlobalCode(g *Generator) error
}
