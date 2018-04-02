# fproto-wrap

Repository for protobuf wrapper generators.

This repository has generators for various programming languages.

The generators should be made in Go using my other library [fdep](https://github.com/RangelReale/fdep), but the
target of the generation can be any programming language.

Currently there are Golang and PHP wrappers.

### generators

* Golang - [https://github.com/RangelReale/fproto-wrap/gowrap](https://github.com/RangelReale/fproto-wrap/tree/master/gowrap)
* PHP - [https://github.com/RangelReale/fproto-wrap/phpwrap](https://github.com/RangelReale/fproto-wrap/tree/master/phpwrap)

### type converters

* [Standard](https://github.com/RangelReale/fproto-wrap-std) (time, duration, uuid, json)
* [Headers](https://github.com/RangelReale/fproto-wrap-headers) (wraps an HTTP-like header structure, a map of lists of values - map[string][]string)

### author

Rangel Reale (rangelspam@gmail.com)
