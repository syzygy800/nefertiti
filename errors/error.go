package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"unicode"
)

// The maximum number of stackframes on any error.
var MaxStackDepth = 50

// Error is an error with an attached stacktrace.
// It can be used wherever the builtin error interface is expected.
type Error struct {
	Err    error
	stack  []uintptr
	frames []StackFrame
	suffix []string
}

// New makes an Error from the given value.
// If that value is already an error then it will be used directly, if not, it will be passed to fmt.Errorf("%v").
// The stacktrace will point to the line of code that called New.
func New(e interface{}) *Error {
	var err error

	switch e := e.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, stack[:])
	return &Error{
		Err:   err,
		stack: stack[:length],
	}
}

// Wrap makes an Error from the given value.
// If that value is already an error then it will be used directly, if not, it will be passed to fmt.Errorf("%v").
// The skip parameter indicates how far up the stack to start the stacktrace. 0 is from the current call, 1 from its caller, etc.
func Wrap(e interface{}, skip int) *Error {
	if e == nil {
		return nil
	}

	var err error

	switch e := e.(type) {
	case *Error:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	return &Error{
		Err:   err,
		stack: stack[:length],
	}
}

// Is detects whether the error is equal to a given error.
// Errors are considered equal by this function if they are the same object, or if they both contain the same error inside an errors.Error.
func Is(e error, original error) bool {
	if e == original {
		return true
	}

	if e, ok := e.(*Error); ok {
		return Is(e.Err, original)
	}

	if original, ok := original.(*Error); ok {
		return Is(e, original.Err)
	}

	return false
}

// Errorf creates a new error with the given message.
// You can use it as a drop-in replacement for fmt.Errorf() to provide descriptive errors in return values.
func Errorf(format string, a ...interface{}) *Error {
	return Wrap(fmt.Errorf(format, a...), 1)
}

// Error returns the underlying error's message.
func (err *Error) Error() string {
	out := err.Err.Error()
	for _, str := range err.suffix {
		if len(str) > 0 {
			if len(out) > 0 && out[len(out)-1] != '.' {
				out += "."
			}
			if !unicode.IsSpace(rune(str[0])) {
				out += " "
			}
			out += str
		}
	}
	return out
}

// Stack returns the callstack formatted the same way that go does in runtime/debug.Stack()
func (err *Error) Stack() []byte {
	buf := bytes.Buffer{}
	for _, frame := range err.StackFrames() {
		buf.WriteString(frame.String())
	}
	return buf.Bytes()
}

// Callers satisfies the bugsnag ErrorWithCallerS() interface so that the stack can be read out.
func (err *Error) Callers() []uintptr {
	return err.stack
}

// ErrorStack returns a string that contains both the error message and the callstack.
// The prefix parameter is used to add an optional prefix to the error message.
func (err *Error) ErrorStack(prefix string) string {
	return func() string {
		if prefix != "" {
			return prefix
		} else {
			return err.TypeName()
		}
	}() + " " + err.Error() + "\n" + string(err.Stack())
}

// StackFrames returns an array of frames containing information about the stack.
func (err *Error) StackFrames() []StackFrame {
	if err.frames == nil {
		err.frames = make([]StackFrame, len(err.stack))
		for i, pc := range err.stack {
			err.frames[i] = NewStackFrame(pc)
		}
	}
	return err.frames
}

// TypeName returns the type this error. e.g. *errors.stringError.
func (err *Error) TypeName() string {
	return reflect.TypeOf(err.Err).String()
}

// Appends a suffix to this error.
func Append(err error, suffix interface{}) *Error {
	if err == nil {
		return nil
	}

	var out *Error

	switch e := err.(type) {
	case *Error:
		out = e
	default:
		out = Wrap(e, 1)
	}

	if str, ok := suffix.(string); ok {
		out.suffix = append(out.suffix, str)
	} else {
		if buf, _ := json.Marshal(suffix); buf != nil {
			out.suffix = append(out.suffix, string(buf))
		}
	}

	return out
}

// Formats program counter, file name, and line number information as returned by runtime.Caller()
func FormatCaller(pc uintptr, file string, line int) string {
	name := runtime.FuncForPC(pc).Name()

	i := len(name) - 1
	for i >= 0 && name[i] != '/' {
		i--
	}
	if i >= 0 {
		name = name[i+1:]
	}

	return fmt.Sprintf("%s:%s:%d", filepath.Base(file), name, line)
}
