package fproto_phpwrap

// Interface to allow customizing various aspects of the output
type Customizer interface {
	// Allows code generation after all the protofile data was generated.
	GenerateCode(g *Generator) error

	// Allows service code generation after all the protofile services were generated.
	GenerateServiceCode(g *Generator) error
}
