package logger

import (
	"fmt"
	"io"
	"os"
)

type Logger struct {
	out io.Writer
	err io.Writer
}

func New() *Logger {
	return NewWithWriters(os.Stdout, os.Stderr)
}

func NewWithWriters(out, err io.Writer) *Logger {
	return &Logger{out: out, err: err}
}

func (l *Logger) Info(format string, args ...any) {
	fmt.Fprintf(l.out, format+"\n", args...)
}

func (l *Logger) Error(format string, args ...any) {
	fmt.Fprintf(l.err, format+"\n", args...)
}
