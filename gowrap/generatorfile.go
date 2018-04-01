package fproto_gowrap

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/RangelReale/fdep"
	"github.com/RangelReale/fproto"
	"github.com/RangelReale/fproto-wrap"
)

// A single generated output file
type GeneratorFile struct {
	generator *Generator
	FileId    string
	Suffix    string

	*bytes.Buffer
	indent string

	imports  map[string]string
	havedata bool
}

// Creates a new generator file
func NewGeneratorFile(generator *Generator, fileId string, suffix string) *GeneratorFile {
	return &GeneratorFile{
		generator: generator,
		FileId:    fileId,
		Suffix:    suffix,
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

// Declares a dependency using a DepFile.
func (g *GeneratorFile) FileDep(depfile *fdep.DepFile, defalias string, is_gowrap bool) string {
	if depfile == nil {
		depfile = g.G().GetDepFile()
	}
	var p string
	if is_gowrap && !depfile.IsSamePackage(g.G().GetDepFile()) && g.G().IsFileWrap(depfile) {
		p = g.G().GoWrapPackage(depfile)
	} else {
		p = depfile.GoPackage()
	}
	return g.Dep(p, defalias)
}

// Declares a dependency and returns the alias to be used on this file.
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

	// Reformat generated code.
	fset := token.NewFileSet()
	raw := tmp.Bytes()
	ast, err := parser.ParseFile(fset, "", tmp, parser.ParseComments)
	if err != nil {
		// Print out the bad code with line numbers.
		// This should never happen in practice, but it can while changing generated code,
		// so consider this a debugging aid.
		var src bytes.Buffer
		s := bufio.NewScanner(bytes.NewReader(raw))
		for line := 1; s.Scan(); line++ {
			fmt.Fprintf(&src, "%5d\t%s\n", line, s.Bytes())
		}
		return errors.New(fmt.Sprint("bad Go source code was generated:", err.Error(), "\n"+src.String()))
	}

	// write into the requested io.Writer
	err = (&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(w, fset, ast)
	if err != nil {
		return fmt.Errorf("generated Go source code could not be reformatted:", err.Error())
	}

	return nil
}

// Returns the expected output file path and name
func (g *GeneratorFile) Filename() string {
	p := g.G().GoWrapPackage(g.G().GetDepFile())
	return path.Join(p, strings.TrimSuffix(path.Base(g.G().GetDepFile().FilePath), path.Ext(g.G().GetDepFile().FilePath))+g.Suffix+".fwpb.go")
}

func (g *GeneratorFile) generateHeader() {
	p := fproto_wrap.BaseName(g.G().GoWrapPackage(g.G().GetDepFile()))

	g.P("// Code generated by fproto-gowrap. DO NOT EDIT.")
	g.P("// source file: ", g.G().GetDepFile().FilePath)

	g.P("package ", p)
	g.P()
}

func (g *GeneratorFile) generateImports() {
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
}

func (g *GeneratorFile) GenerateComment(comment *fproto.Comment) bool {
	if comment != nil && len(comment.Lines) > 0 {
		cstr := "//"
		if comment.ExtraSlash {
			cstr += "/"
		}
		for _, dl := range comment.Lines {
			g.P(cstr, " ", strings.TrimSpace(dl))
		}
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
