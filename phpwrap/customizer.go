package fproto_phpwrap

// Interface to allow customizing various aspects of the output
type Customizer interface {
	// Allows code generation after all the protofile data was generated.
	GenerateCode(g *Generator) error

	// Allows service code generation after all the protofile services were generated.
	GenerateServiceCode(g *Generator) error
}

// Wraps a list of customizers
type wrapCustomizers struct {
	customizers []Customizer
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
