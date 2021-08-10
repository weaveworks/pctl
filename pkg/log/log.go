package log

import (
	"fmt"
	"io"
)

type StderrLogger struct {
	Stderr io.Writer
}

func (l StderrLogger) Actionf(format string, a ...interface{}) {
	fmt.Fprintln(l.Stderr, `►`, fmt.Sprintf(format, a...))
}

func (l StderrLogger) Waitingf(format string, a ...interface{}) {
	fmt.Fprintln(l.Stderr, `◎`, fmt.Sprintf(format, a...))
}

func (l StderrLogger) Successf(format string, a ...interface{}) {
	fmt.Fprintln(l.Stderr, `✔`, fmt.Sprintf(format, a...))
}

func (l StderrLogger) Warningf(format string, a ...interface{}) {
	fmt.Fprintln(l.Stderr, `⚠️`, fmt.Sprintf(format, a...))
}

func (l StderrLogger) Failuref(format string, a ...interface{}) {
	fmt.Fprintln(l.Stderr, `✗`, fmt.Sprintf(format, a...))
}
