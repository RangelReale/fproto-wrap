package fproto_gowrap

import (
	"io"

	"github.com/RangelReale/fproto/fdep"
)

// Root wrapper struct
type Wrapper struct {
	dep *fdep.Dep

	PkgSource      PkgSource
	TypeConverters []TypeConverterPlugin
	ServiceGen     ServiceGen
	Customizers    []Customizer
}

// Creates a new wrapper
func NewWrapper(dep *fdep.Dep) *Wrapper {
	return &Wrapper{
		dep: dep,
	}
}

// Generates one file
func (wp *Wrapper) GenerateFile(filename string, w io.Writer) error {
	g, err := NewGenerator(wp.dep, filename)
	g.PkgSource = wp.PkgSource
	g.TypeConverters = wp.TypeConverters
	g.ServiceGen = wp.ServiceGen
	g.Customizers = wp.Customizers
	if err != nil {
		return err
	}

	err = g.Generate()
	if err != nil {
		return err
	}

	err = g.Output(w)
	return err
}

// Generates all owned files.
func (wp *Wrapper) Generate(output FileOutput) error {
	output.Initialize()
	defer output.Finalize()

	for _, df := range wp.dep.Files {
		if df.DepType == fdep.DepType_Own {
			g, err := NewGenerator(wp.dep, df.FilePath)
			if err != nil {
				return err
			}

			if !g.IsFileGowrap(df) {
				continue
			}

			g.PkgSource = wp.PkgSource
			g.TypeConverters = wp.TypeConverters
			g.ServiceGen = wp.ServiceGen
			g.Customizers = wp.Customizers

			err = g.Generate()
			if err != nil {
				return err
			}

			err = output.Output(g)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Generates all owned files.
func (wp *Wrapper) GenerateFiles(outputpath string) error {
	output := NewFileOutput_Default(outputpath)
	return wp.Generate(output)
}
