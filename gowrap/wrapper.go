package fproto_gowrap

import (
	"errors"

	"github.com/RangelReale/fdep"
)

type WrapperFile struct {
	FileId    string
	Suffix    string
	FileAlias string // if blank creates a new file, else alias to existing one
}

// Root wrapper struct
type Wrapper struct {
	dep *fdep.Dep

	PkgSource      PkgSource
	TypeConverters []TypeConverterPlugin
	ServiceGen     ServiceGen
	Customizers    []Customizer
	Files          []*WrapperFile
}

// Creates a new wrapper
func NewWrapper(dep *fdep.Dep) *Wrapper {
	return &Wrapper{
		dep: dep,
	}
}

// Generates one file
func (wp *Wrapper) GenerateFile(filename string, output FileOutput) error {
	df, ok := wp.dep.Files[filename]
	if !ok {
		return errors.New("File not found")
	}

	g, err := NewGenerator(wp.dep, df)
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

	// write all files
	for _, gf := range g.Files {
		if gf != nil && gf.HaveData() {
			err = output.Output(gf)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Generates all owned files.
func (wp *Wrapper) Generate(output FileOutput) error {
	output.Initialize()
	defer output.Finalize()

	for _, df := range wp.dep.Files {
		if df.DepType == fdep.DepType_Own {
			g, err := NewGenerator(wp.dep, df)
			if err != nil {
				return err
			}

			if !g.IsFileWrap(df) {
				continue
			}

			g.PkgSource = wp.PkgSource
			g.TypeConverters = wp.TypeConverters
			g.ServiceGen = wp.ServiceGen
			g.Customizers = wp.Customizers
			for _, f := range wp.Files {
				if f.FileAlias != "" {
					g.SetFileAlias(f.FileId, f.FileAlias)
				} else {
					g.SetFile(f.FileId, f.Suffix)
				}
			}

			err = g.Generate()
			if err != nil {
				return err
			}

			// write all files
			for _, gf := range g.Files {
				if gf != nil && gf.HaveData() {
					err = output.Output(gf)
					if err != nil {
						return err
					}
				}
			}

		}
	}

	// check for Customizer_Global customizers
	for _, cz := range wp.Customizers {
		if czglobal, isczglobal := cz.(Customizer_Global); isczglobal {
			g, err := NewGenerator(wp.dep, nil)
			if err != nil {
				return err
			}

			g.PkgSource = wp.PkgSource
			g.TypeConverters = wp.TypeConverters
			g.ServiceGen = wp.ServiceGen
			/*
				g.Customizers = wp.Customizers
				for _, f := range wp.Files {
					if f.FileAlias != "" {
						g.SetFileAlias(f.FileId, f.FileAlias)
					} else {
						g.SetFile(f.FileId, f.Suffix)
					}
				}
			*/

			err = czglobal.GenerateGlobalCode(g)
			if err != nil {
				return err
			}

			// write all files
			for _, gf := range g.Files {
				if gf != nil && gf.HaveData() {
					err = output.Output(gf)
					if err != nil {
						return err
					}
				}
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
