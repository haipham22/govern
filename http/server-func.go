package http

import "context"

// ServerFunc provides function-based server interface.
type ServerFunc struct {
	StartF    func() error
	ShutdownF func(context.Context) error
}

// Start executes the start function.
func (sf ServerFunc) Start() error {
	if sf.StartF == nil {
		return nil
	}
	return sf.StartF()
}

// Shutdown executes the shutdown function.
func (sf ServerFunc) Shutdown(ctx context.Context) error {
	if sf.ShutdownF == nil {
		return nil
	}
	return sf.ShutdownF(ctx)
}

// NewServerFuncFromServer converts Server to ServerFunc.
func NewServerFuncFromServer(s *Server) ServerFunc {
	return ServerFunc{
		StartF: s.Start,
		ShutdownF: func(ctx context.Context) error {
			return s.Shutdown(ctx)
		},
	}
}
