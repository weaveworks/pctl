package log

import (
	"fmt"
)

type PrintLogger struct{}

func (l PrintLogger) Actionf(format string, a ...interface{}) {
	fmt.Println(`►`, fmt.Sprintf(format, a...))
}

func (l PrintLogger) Waitingf(format string, a ...interface{}) {
	fmt.Println(`◎`, fmt.Sprintf(format, a...))
}

func (l PrintLogger) Successf(format string, a ...interface{}) {
	fmt.Println(`✔`, fmt.Sprintf(format, a...))
}

func (l PrintLogger) Warningf(format string, a ...interface{}) {
	fmt.Println(`⚠️`, fmt.Sprintf(format, a...))
}

func (l PrintLogger) Failuref(format string, a ...interface{}) {
	fmt.Println(`✗`, fmt.Sprintf(format, a...))
}
