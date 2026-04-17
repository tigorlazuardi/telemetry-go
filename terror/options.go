package terror

import "github.com/tigorhutasuhut/telemetry-go/tcaller"

// ErrorOption configures [Wrap] and [Fail].
type ErrorOption interface{ applyError(*ErrorOptions) }

// ErrorOptions builds options for [Wrap] and [Fail].
//
// When multiple options are passed, later singular values override earlier
// ones, while slice-based values are appended.
type ErrorOptions struct {
	message *string
	code    *Code
	caller  *tcaller.Caller
	fields  []any
}

// Err creates a new ErrorOptions builder.
func Err() *ErrorOptions { return &ErrorOptions{} }

func (o *ErrorOptions) applyError(cfg *ErrorOptions) {
	if o == nil {
		return
	}
	if o.message != nil {
		msg := *o.message
		cfg.message = &msg
	}
	if o.code != nil {
		code := *o.code
		cfg.code = &code
	}
	if o.caller != nil {
		caller := *o.caller
		cfg.caller = &caller
	}
	if len(o.fields) > 0 {
		cfg.fields = append(cfg.fields, o.fields...)
	}
}

// Message sets the human-readable error message.
func (o *ErrorOptions) Message(msg string) *ErrorOptions {
	o.message = &msg
	return o
}

// Code sets the error code.
func (o *ErrorOptions) Code(code Code) *ErrorOptions {
	o.code = &code
	return o
}

// Caller overrides the source location recorded on the error.
func (o *ErrorOptions) Caller(caller tcaller.Caller) *ErrorOptions {
	o.caller = &caller
	return o
}

// Fields appends structured context fields to the error.
func (o *ErrorOptions) Fields(fields ...any) *ErrorOptions {
	o.fields = append(o.fields, fields...)
	return o
}
