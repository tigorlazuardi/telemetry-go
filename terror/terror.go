package terror

import "github.com/tigorhutasuhut/telemetry-go/tcaller"

// Wrap creates a new [Error] that wraps err as its cause.
//
// The caller is automatically captured from the Wrap call site unless
// overridden via [ErrorOptions.Caller].
func Wrap(err error, opts ...ErrorOption) error {
	cfg := &ErrorOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt.applyError(cfg)
		}
	}

	e := &Error{
		Caller: resolveErrorCaller(cfg.caller),
		Fields: cfg.fields,
		Source: []error{err},
	}
	if cfg.message != nil {
		e.Message = *cfg.message
	}
	if cfg.code != nil {
		e.Code = *cfg.code
	}
	return e
}

// Fail creates a new [Error] without wrapping an existing error.
//
// The caller is automatically captured from the Fail call site unless
// overridden via [ErrorOptions.Caller].
func Fail(opts ...ErrorOption) error {
	cfg := &ErrorOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt.applyError(cfg)
		}
	}

	e := &Error{
		Caller: resolveErrorCaller(cfg.caller),
		Fields: cfg.fields,
	}
	if cfg.message != nil {
		e.Message = *cfg.message
	}
	if cfg.code != nil {
		e.Code = *cfg.code
	}
	return e
}

// resolveErrorCaller returns the explicit caller when set, otherwise captures
// the call site of the Wrap/Fail function (two frames above this one).
func resolveErrorCaller(explicit *tcaller.Caller) tcaller.Caller {
	if explicit != nil {
		return *explicit
	}
	return tcaller.New(2)
}
