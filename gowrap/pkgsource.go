package fproto_gowrap

import "github.com/RangelReale/fdep"

// Interface to customize the Go package name for a depfile
type PkgSource interface {
	// Gets a Go package name for a depfile. If supported, must return true on the second result.
	GetPkg(depfile *fdep.DepFile) (string, bool)
}
