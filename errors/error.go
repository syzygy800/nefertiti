package errors

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
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
	return err.Err.Error()
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
// The prefix parameter is used to add a prefix to the error message.
// The suffix parameter is used to add a suffix to the error message.
func (err *Error) ErrorStack(prefix, suffix string) string {
	var msg string
	if prefix != "" {
		msg = prefix
	} else {
		msg = err.TypeName()
	}

	if msg != "" {
		msg = msg + " "
	}
	msg = msg + err.Error()
	if suffix != "" {
		if suffix[0] != ' ' {
			msg = msg + " "
		}
		msg = msg + suffix
	}

	if len(err.suffix) > 0 {
		for _, str := range err.suffix {
			msg = msg + "\n" + str
		}
	}

	msg = msg + "\n" + string(err.Stack())

	return msg
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

// Appends one ore more suffixes to this error. Every suffix is prefixed with this prefix.
func Append(err error, prefix string, suffix ...string) *Error {
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

	if prefix == "" {
		out.suffix = append(out.suffix, suffix...)
	} else {
		for _, line := range suffix {
			out.suffix = append(out.suffix, (prefix + line))
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
