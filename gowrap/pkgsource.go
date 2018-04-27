package fproto_gowrap

import "github.com/RangelReale/fdep"

// Interface to customize the Go package name for a depfile
type PkgSource interface {
	// Gets a Go package name for a depfile. If supported, must return true on the second result.
	GetPkg(g *Generator, depfile *fdep.DepFile) (string, bool)

	// Gets a Go file package name to use as the "package" directive for the file
	// The default is "fw"+BaseName(GetPkg())
	// If supported, must return true on the second result.
	GetFilePkg(g *Generator, depfile *fdep.DepFile) (string, bool)
}
