package fproto_gowrap

type FileOutput interface {
	Initialize() error
	Finalize() error
	Output(g *GeneratorFile) error
}
