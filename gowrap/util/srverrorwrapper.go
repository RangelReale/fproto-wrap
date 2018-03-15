package fproto_gowrap_util

type ServerErrorType int

const (
	SET_IMPORT ServerErrorType = iota
	SET_CALL
	SET_EXPORT
)

type ServerErrorWrapper interface {
	WrapError(ServerErrorType, error) error
}
