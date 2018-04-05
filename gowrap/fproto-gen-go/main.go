package main

import (
	"flag"
	"log"
	"os"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto-wrap/gowrap"
)

// Array flags type
type arrayFlags []string

func (i *arrayFlags) String() string {
	return "array flags"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// command line flags
var (
	incPaths   = arrayFlags{}
	protoPaths = arrayFlags{}
	outputPath = flag.String("output_path", "", "Output root path")
)

// Usage:
// fproto-gen-go -inc_path="/protoc-3.5.1/include" -inc_path="/otherproto/include" -proto_path="/mysource/proto" -output_path="/mysource/proto_wrappers"
func main() {
	// parse command line flags
	flag.Var(&incPaths, "inc_path", "Include paths (can be set multiple times)")
	flag.Var(&protoPaths, "proto_path", "Application protocol buffers paths (can be set multiple times)")
	flag.Parse()

	// check parameters
	if len(protoPaths) == 0 {
		log.Fatal("At least one proto path is required")
	}

	if *outputPath == "" {
		log.Fatal("The output path is required")
	}

	// create output path
	if err := os.MkdirAll(*outputPath, os.ModePerm); err != nil {
		log.Fatalf("Error creating output_path '%s': %v", *outputPath, err)
	}

	// creates a new fdep.Dep to parse the proto files
	parsedep := fdep.NewDep()

	// add include directories
	parsedep.IncludeDirs = append(parsedep.IncludeDirs, incPaths...)

	// add own source code
	for _, protoPath := range protoPaths {
		if s, err := os.Stat(protoPath); err != nil {
			log.Fatalf("Error reading proto_path: %v", err)
		} else if !s.IsDir() {
			log.Fatalf("proto_path isn't a directory: %s", protoPath)
		}

		err := parsedep.AddPath(protoPath, fdep.DepType_Own)
		if err != nil {
			log.Fatal(err)
		}
	}

	// check for missing dependencies
	err := parsedep.CheckDependencies()
	if err != nil {
		log.Fatal(err)
	}

	// creates the wrapper generator
	w := fproto_gowrap.NewWrapper(parsedep)

	// Add type converters
	/*
		w.TypeConverters = append(w.TypeConverters,
			&fprotostd_gowrap_uuid.TypeConverterPlugin_UUID{},
			&fprotostd_gowrap_time.TypeConverterPlugin_Time{},
			&fprotostd_gowrap_duration.TypeConverterPlugin_Duration{},
			&fprotostd_gowrap_jsonobject.TypeConverterPlugin_JSONObject{},
		)
	*/

	// Add Customizers
	/*
		w.Customizers = append(w.Customizers,
			&fprotostd_gowrap_sqltag.Customizer_SQLTag{},
			&fprotostd_gowrap_jsontag.Customizer_JSONTag{},
		)
		w.ServiceGen = fproto_gowrap.NewServiceGen_gRPC()
	*/

	// generate the wrapper files
	err = w.GenerateFiles(*outputPath)
	if err != nil {
		log.Fatal(err)
	}
}
