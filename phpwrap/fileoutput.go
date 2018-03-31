package fproto_phpwrap

type FileOutput interface {
	Initialize() error
	Finalize() error
	Output(g *GeneratorFile) error
}
