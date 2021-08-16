package log

import (
	"fmt"
)

func Actionf(m string, a ...interface{}) {
	format(`►`, m, a...)
}

func Waitingf(m string, a ...interface{}) {
	format(`◎`, m, a...)
}

func Successf(m string, a ...interface{}) {
	format(`✔`, m, a...)
}

func Warningf(m string, a ...interface{}) {
	format(`⚠️`, m, a...)
}

func Failuref(m string, a ...interface{}) {
	format(`✗`, m, a...)
}

func format(tickmark, m string, a ...interface{}) {
	fmt.Println(tickmark, fmt.Sprintf(m, a...))
}
