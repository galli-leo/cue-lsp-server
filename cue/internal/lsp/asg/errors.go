package asg

import (
	"fmt"

	"cuelang.org/go/cue/errors"

	"cuelang.org/go/cue/token"
)

// When displaying an error in an editor, we don't only want to indicate the starting location of an error.
// Usually we would like to have a certain range marked as red.
//
// This error type is used for that. It implements the errors.Error interface.
type RangeError struct {
	pos token.Pos
	end token.Pos
	errors.Message

	// The underlying error that triggered this one, if any.
	err error
}

func Newf(r PosRange, msg string, args ...interface{}) *RangeError {
	return &RangeError{
		pos:     r.Pos(),
		end:     r.End(),
		Message: errors.NewMessage(msg, args),
	}
}

func Wrapf(r PosRange, err error, msg string, args ...interface{}) *RangeError {
	return &RangeError{
		pos:     r.Pos(),
		end:     r.End(),
		Message: errors.NewMessage(msg, args),
		err:     err,
	}
}

func (e *RangeError) Path() []string              { return []string{} }
func (e *RangeError) InputPositions() []token.Pos { return []token.Pos{e.pos, e.end} }
func (e *RangeError) Position() token.Pos         { return e.pos }
func (e *RangeError) Unwrap() error               { return e.err }
func (e *RangeError) Cause() error                { return e.err }

func (e *RangeError) Range() (token.Pos, token.Pos) {
	return e.pos, e.end
}

// Error implements the error interface.
func (e *RangeError) Error() string {
	if e.err == nil {
		return e.Message.Error()
	}
	if e.Message.Error() == "" {
		return e.err.Error()
	}
	return fmt.Sprintf("%s: %s", e.Message.Error(), e.err)
}
