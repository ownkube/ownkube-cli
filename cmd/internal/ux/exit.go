package ux

// ExitError ends okctl with Code. Err, when set, is printed like any other
// error; a nil Err exits quietly because the command already said why (a
// drift report, a child process's own output).
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }
