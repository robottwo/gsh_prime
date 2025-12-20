package termfeatures

import "errors"

var (
	// ErrDumbTerminal indicates the terminal doesn't support escape sequences.
	ErrDumbTerminal = errors.New("dumb terminal: no escape sequence support")

	// ErrNotATerminal indicates the output is not connected to a terminal.
	ErrNotATerminal = errors.New("not connected to a terminal")
)
