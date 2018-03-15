package fproto_gowrap

import "github.com/RangelReale/fproto/fdep"

// Interface to customize the Go package name for a filedep
type PkgSource interface {
	// Gets a Go package name for a filedep. If supported, must return true on the second result.
	GetPkg(filedep *fdep.FileDep) (string, bool)
}
