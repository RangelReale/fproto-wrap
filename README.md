# fproto-wrap

Repository for protobuf wrapper generators.

This repository has generators for various programming languages.

The generators should be made in Go with the help of my other library [fdep](https://github.com/RangelReale/fdep), but the
target of the generation can be any programming language.

Currently there are Golang and PHP wrapper generators.

The generators support type converters, which allows to replace a protobuf-generated type with an easier-to-use one.
For example, in Go, instead of using "Timestamp" from "github.com/golang/protobuf/ptypes/timestamp", it converts
automatically to and from "Time" from the "time" package.

This package aims to be *extremelly* customizable, allowing to generate extra code and type converters using interfaces,
without changing this package source code.

### generators

* Golang - [https://github.com/RangelReale/fproto-wrap/gowrap](https://github.com/RangelReale/fproto-wrap/tree/master/gowrap)
* PHP - [https://github.com/RangelReale/fproto-wrap/phpwrap](https://github.com/RangelReale/fproto-wrap/tree/master/phpwrap)

### type converters

* [Standard](https://github.com/RangelReale/fproto-wrap-std) (time, duration, uuid, json)
* [Headers](https://github.com/RangelReale/fproto-wrap-headers) (wraps an HTTP-like header structure, a map of lists of values - map[string][]string)

### customizers

* [Validator](https://github.com/RangelReale/fproto-wrap-validator) Validator generator the for wrapped code.

### author

Rangel Reale (rangelspam@gmail.com)
