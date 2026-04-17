package terror

import "github.com/tigorhutasuhut/telemetry-go/tcaller"

// Wrap creates a new [Error] that wraps err as its cause.
//
// The caller is automatically captured from the Wrap call site unless
// overridden via [ErrorOptions.Caller].
func Wrap(err error, opts ...ErrorOption) Error {
	cfg := &ErrorOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt.applyError(cfg)
		}
	}

	e := &appError{
		caller: resolveErrorCaller(cfg.caller),
		fields: cfg.fields,
		source: []error{err},
	}
	if cfg.message != nil {
		e.message = *cfg.message
	}
	if cfg.code != nil {
		e.code = *cfg.code
	}
	return e
}

// Fail creates a new [Error] without wrapping an existing error.
//
// The caller is automatically captured from the Fail call site unless
// overridden via [ErrorOptions.Caller].
func Fail(opts ...ErrorOption) Error {
	cfg := &ErrorOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt.applyError(cfg)
		}
	}

	e := &appError{
		caller: resolveErrorCaller(cfg.caller),
		fields: cfg.fields,
	}
	if cfg.message != nil {
		e.message = *cfg.message
	}
	if cfg.code != nil {
		e.code = *cfg.code
	}
	return e
}

// Multi creates a new [Error] whose Source contains all non-nil errors from
// errs. Nil entries are silently discarded.
//
// The returned Error can be used directly or combined with [Error.Resolve] to
// return nil when all inputs were nil:
//
//	return terror.Multi(mayFail1(), mayFail2()).Resolve()
//
// The caller is automatically captured from the Multi call site.
func Multi(errs ...error) Error {
	e := &appError{caller: resolveErrorCaller(nil)}
	for _, err := range errs {
		if err != nil {
			e.source = append(e.source, err)
		}
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
