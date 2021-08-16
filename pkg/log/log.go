package log

import (
	"fmt"
)

type PrintLogger struct{}

func (l PrintLogger) Actionf(m string, a ...interface{}) {
	format(`►`, m, a...)
}

func (l PrintLogger) Waitingf(m string, a ...interface{}) {
	format(`◎`, m, a...)
}

func (l PrintLogger) Successf(m string, a ...interface{}) {
	format(`✔`, m, a...)
}

func (l PrintLogger) Warningf(m string, a ...interface{}) {
	format(`⚠️`, m, a...)
}

func (l PrintLogger) Failuref(m string, a ...interface{}) {
	format(`✗`, m, a...)
}

func format(tickmark, m string, a ...interface{}) {
	fmt.Println(tickmark, fmt.Sprintf(m, a...))
}
