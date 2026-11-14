package main

import (
	"fmt"
	"time"
)

type Level int

const (
	Debug Level = iota // 0; type and expression repeat on the lines below
	Info               // 1
	Warn               // 2
	Error              // 3
)

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	}
	return fmt.Sprintf("Level(%d)", int(l))
}

const (
	_  = iota             // skip zero
	KB = 1 << (10 * iota) // 1 << 10
	MB                    // 1 << 20
	GB                    // 1 << 30
)

const huge = 1 << 100 // untyped: exact at compile time, never stored at run time

func main() {
	fmt.Println(Info, Error, Level(7)) // fmt calls String()
	fmt.Println(KB, MB, GB)
	fmt.Println(huge >> 98) // fine: only the final value must fit an int

	timeout := 2 * time.Second // the untyped 2 takes on time.Duration
	fmt.Println(timeout)
}
