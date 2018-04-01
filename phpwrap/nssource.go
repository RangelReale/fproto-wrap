package fproto_phpwrap

import "github.com/RangelReale/fdep"

// Interface to customize the PHP namespace name for a depfile
type NSSource interface {
	// Gets a PHP namespace name for a depfile. If supported, must return true on the second result.
	GetNS(depfile *fdep.DepFile) (string, bool)
}
