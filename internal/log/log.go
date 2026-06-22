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

func Println(args ...any) {
	log.Println(args...)
}

func Printf(format string, args ...any) {
	log.Printf(format, args...)
}

func PrintfGreen(format string, args ...any) {
	Printf(green(format), args...)
}

func PrintlnGreen(args ...any) {
	Println("\033[32m" + fmt.Sprintln(args...) + "\033[0m")
}

func PrintfRed(format string, args ...any) {
	Printf(red(format), args...)
}

func PrintlnRed(args ...any) {
	Println("\033[31m" + fmt.Sprint(args...) + "\033[0m")
}

func Errorf(format string, args ...any) {
	PrintfRed(format, args...)
}

func Errorln(args ...any) {
	PrintlnRed(args...)
}
