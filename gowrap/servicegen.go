package fproto_gowrap

import "github.com/RangelReale/fproto"

// Interface to generate service specifications
type ServiceGen interface {
	ServiceType() string
	GenerateService(g *Generator, svc *fproto.ServiceElement) error
}
