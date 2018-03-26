package fproto_phpwrap

import "github.com/RangelReale/fdep"

// Interface to customize the PHP namespace name for a filedep
type NSSource interface {
	// Gets a PHP namespace name for a filedep. If supported, must return true on the second result.
	GetNS(filedep *fdep.FileDep) (string, bool)
}
