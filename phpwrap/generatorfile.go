package fproto_phpwrap

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/RangelReale/fproto"
)

// A single generated output file
type GeneratorFile struct {
	generator *Generator
	FileId    string
	FilePath  string

	*bytes.Buffer
	indent string

	imports  map[string]string
	havedata bool
}

// Creates a new generator file
func NewGeneratorFile(generator *Generator, fileId string) *GeneratorFile {
	return &GeneratorFile{
		generator: generator,
		FileId:    fileId,
		FilePath:  fileId,
		Buffer:    new(bytes.Buffer),
		imports:   make(map[string]string),
		havedata:  false,
	}
}

// Returns the parent generator
func (g *GeneratorFile) G() *Generator {
	return g.generator
}

// Checks if any data was written on this file
func (g *GeneratorFile) HaveData() bool {
	return g.havedata
}

// Declares a dependency using a FileDep.
/*
func (g *GeneratorFile) FileDep(filedep *fdep.FileDep, defalias string, is_gowrap bool) string {
	if filedep == nil {
		filedep = g.G().GetFileDep()
	}
	var p string
	if is_gowrap && !filedep.IsSamePackage(g.G().GetFileDep()) && g.G().IsFileGowrap(filedep) {
		p = g.G().GoWrapPackage(filedep)
	} else {
		p = filedep.GoPackage()
	}
	return g.Dep(p, defalias)
}
*/

// Declares a dependency and returns the alias to be used on this file.
/*
func (g *GeneratorFile) Dep(imp string, defalias string) string {
	var alias string
	var ok bool
	if alias, ok = g.imports[imp]; ok {
		return alias
	}

	if defalias == "" {
		defalias = path.Base(imp)
	}

	defalias = strings.Replace(defalias, ".", "_", -1)

	alias = defalias
	aliasct := 0
	aliasok := false
	for !aliasok {
		aliasok = true

		for _, a := range g.imports {
			if a == alias {
				aliasct++
				alias = fmt.Sprintf("%s%d", defalias, aliasct)
				aliasok = false
			}
		}

		if aliasok {
			break
		}
	}

	g.imports[imp] = alias
	return alias
}
*/

// Returns the generated file as a string.
func (g *GeneratorFile) Output(w io.Writer) error {
	// write in temporary buffer
	tmp := new(bytes.Buffer)

	// Generate header and imports last, though they appear first in the output.
	rem := g.Buffer
	g.Buffer = new(bytes.Buffer)

	g.generateHeader()
	g.generateImports()

	// write headers / imports
	_, err := tmp.Write(g.Bytes())
	if err != nil {
		g.Buffer = rem
		return err
	}

	// write previous content
	_, err = tmp.Write(rem.Bytes())
	if err != nil {
		g.Buffer = rem
		return err
	}

	// restore buffer
	g.Buffer = rem

	_, err = w.Write(tmp.Bytes())
	return err
}

// Returns the expected output file path and name
func (g *GeneratorFile) Filename() string {
	return g.FilePath + ".php"
}

func (g *GeneratorFile) generateHeader() {
	_, wrapNS, _ := g.G().PhpWrapNS(g.G().GetFileDep())

	g.P("<?php")
	g.P("namespace ", wrapNS, ";")
	g.P()
}

func (g *GeneratorFile) generateImports() {
	/*
		if len(g.imports) > 0 {
			g.P("import (")
			g.In()

			// loop imports in ascending order
			keys := make([]string, 0)
			for k, _ := range g.imports {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, in := range keys {
				g.P(g.imports[in], ` "`, in, `"`)
			}
			g.Out()
			g.P(")")

			g.P()
		}
	*/
}

func (g *GeneratorFile) GenerateComment(comment *fproto.Comment, extraLines []string) bool {
	if (comment != nil && len(comment.Lines) > 0) || len(extraLines) > 0 {
		g.P("/**")
		if comment != nil {
			for _, dl := range comment.Lines {
				g.P(" * ", strings.TrimSpace(dl))
			}
		}
		if len(extraLines) > 0 {
			for _, dl := range extraLines {
				g.P(" * ", strings.TrimSpace(dl))
			}
		}
		g.P(" */")
		return true
	}
	return false
}

// Generates a multi-line comment starting and ending with an empty line
func (g *GeneratorFile) GenerateCommentLine(str ...string) {
	if len(str) > 0 {
		g.P("//")
		p := []interface{}{"// "}
		for _, s := range str {
			p = append(p, s)
		}
		g.P(p...)
		g.P("//")
	}
}

func (g *GeneratorFile) GenerateFieldComment(field fproto.FieldElementTag, extraLines []string) bool {
	var comment *fproto.Comment
	switch xfld := field.(type) {
	case *fproto.FieldElement:
		comment = xfld.Comment
	case *fproto.MapFieldElement:
		comment = xfld.Comment
	case *fproto.OneOfFieldElement:
		comment = xfld.Comment
	}
	return g.GenerateComment(comment, extraLines)
}

// P prints the arguments to the generated output.  It handles strings and int32s, plus
// handling indirections because they may be *string, etc.
func (g *GeneratorFile) P(str ...interface{}) {
	g.havedata = true

	g.WriteString(g.indent)
	for _, v := range str {
		switch s := v.(type) {
		case string:
			g.WriteString(s)
		case *string:
			g.WriteString(*s)
		case bool:
			fmt.Fprintf(g, "%t", s)
		case *bool:
			fmt.Fprintf(g, "%t", *s)
		case int:
			fmt.Fprintf(g, "%d", s)
		case *int32:
			fmt.Fprintf(g, "%d", *s)
		case *int64:
			fmt.Fprintf(g, "%d", *s)
		case float64:
			fmt.Fprintf(g, "%g", s)
		case *float64:
			fmt.Fprintf(g, "%g", *s)
		default:
			panic(fmt.Sprintf("unknown type in printer: %T", v))
		}
	}
	g.WriteByte('\n')
}

// In Indents the output one tab stop.
func (g *GeneratorFile) In() { g.indent += "\t" }

// Out unindents the output one tab stop.
func (g *GeneratorFile) Out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[1:]
	}
}

/*
func (g *GeneratorFile) GenerateErrorCheck(extraRetVal string) {
	g.P("if err != nil {")
	g.In()
	if extraRetVal != "" {
		g.P("return ", extraRetVal, ", err")
	} else {
		g.P("return err")
	}
	g.Out()
	g.P("}")
}
*/
