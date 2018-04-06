package fproto_phpwrap

import "github.com/RangelReale/fdep"

// Interface to allow customizing various aspects of the output
type Customizer interface {
	// Allows code generation after all the protofile data was generated.
	GenerateCode(g *Generator) error

	// Allows service code generation after all the protofile services were generated.
	GenerateServiceCode(g *Generator) error
}

// Interface to allow customization of each generated class
type CustomizerClass interface {
	// Get the base class for the type
	GetBaseClass(g *Generator, tp *fdep.DepType) (string, bool)

	// Generate code on each class
	GenerateClassCode(g *Generator, tp *fdep.DepType) error
}
