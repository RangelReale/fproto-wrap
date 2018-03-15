# fproto-wrap

Repository for protobuf wrapper generators.

This should serve as a repository of useful default types, and also for generators for various programming languages.

The generators should be made in Go using my other library [fdep](https://github.com/RangelReale/fproto/tree/master/fdep), but the
target of the generation can be any programming language.

Currently there is only a Golang wrapper generator.

### generators

* Golang - [https://github.com/RangelReale/fproto-wrap/gowrap](https://github.com/RangelReale/fproto-wrap/tree/master/gowrap)

### default types

* Time / NullTime (wraps google.protobuf.Timestamp)
* JSONObject (wraps a checked JSON string)
* UUID / NullUUID (wraps a checked UUID string)

### external types

* [Headers](https://github.com/RangelReale/fproto-gowrap-headers) (wraps an HTTP-like header structure, a map of lists of values - map[string][]string)

### author

Rangel Reale (rangelspam@gmail.com)
