package output

import (
	"fmt"
	"io"
)

type IndentWriter struct {
	prefix string
	w      io.Writer
	atLine bool
}

func NewIndentWriter(w io.Writer) *IndentWriter {
	return &IndentWriter{prefix: Indent, w: w}
}

func (w *IndentWriter) Write(p []byte) (int, error) {
	if w.w == nil {
		return len(p), nil
	}
	written := 0
	for _, b := range p {
		if !w.atLine {
			if _, err := fmt.Fprint(w.w, w.prefix); err != nil {
				return written, err
			}
			w.atLine = true
		}
		if _, err := w.w.Write([]byte{b}); err != nil {
			return written, err
		}
		written++
		if b == '\n' {
			w.atLine = false
		}
	}
	return written, nil
}

func (w *IndentWriter) Close() error {
	return nil
}
