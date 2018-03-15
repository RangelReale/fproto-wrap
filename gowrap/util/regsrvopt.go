package fproto_gowrap_util

type RegServerOptions struct {
	ErrorWrapper ServerErrorWrapper
}

type RegServerOption func(*RegServerOptions)

// Adds an error wrapper
func WithServerErrorWrapper(w ServerErrorWrapper) RegServerOption {
	return func(o *RegServerOptions) {
		o.ErrorWrapper = w
	}
}
