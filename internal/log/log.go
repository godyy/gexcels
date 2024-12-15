package log

import (
	"fmt"
	"log"
)

func green(format string) string {
	return "\033[32m" + format + "\033[0m"
}

func red(format string) string {
	return "\033[31m" + format + "\033[0m"
}

func Println(args ...interface{}) {
	log.Println(args...)
}

func Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func PrintfGreen(format string, args ...interface{}) {
	Printf(green(format), args...)
}

func PrintlnGreen(args ...interface{}) {
	Println("\033[32m" + fmt.Sprintln(args...) + "\033[0m")
}

func PrintfRed(format string, args ...interface{}) {
	Printf(red(format), args...)
}

func PrintlnRed(args ...interface{}) {
	Println("\033[31m" + fmt.Sprint(args...) + "\033[0m")
}

func Errorf(format string, args ...interface{}) {
	PrintfRed(format, args...)
}

func Errorln(args ...interface{}) {
	PrintlnRed(args...)
}
