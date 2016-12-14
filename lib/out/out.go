package out

import (
	"fmt"

	"github.com/fatih/color"
)

var white *color.Color
var bold *color.Color
var cyan *color.Color
var yellow *color.Color
var red *color.Color

func init() {
	white = color.New(color.FgWhite)
	bold = color.New(color.Bold)
	cyan = color.New(color.FgCyan)
	yellow = color.New(color.FgYellow)
	red = color.New(color.FgRed)
}

// Normf prints a normal message.
func Normf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

// Boldf prints a bold message.
func Boldf(format string, v ...interface{}) {
	bold.PrintfFunc()(format, v...)
}

// Examf prints an example message.
func Examf(format string, v ...interface{}) {
	cyan.PrintfFunc()(format, v...)
}

// Warnf prints a warning message.
func Warnf(format string, v ...interface{}) {
	yellow.PrintfFunc()(format, v...)
}

// Errof prints an error message.
func Errof(format string, v ...interface{}) {
	red.PrintfFunc()(format, v...)
}
