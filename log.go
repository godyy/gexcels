package gexcels

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

func println(args ...interface{}) {
	log.Println(args...)
}

func printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func printfGreen(format string, args ...interface{}) {
	printf(green(format), args...)
}

func printlnGreen(args ...interface{}) {
	println("\033[32m" + fmt.Sprintln(args...) + "\033[0m")
}

func printfRed(format string, args ...interface{}) {
	printf(red(format), args...)
}

func printlnRed(args ...interface{}) {
	println("\033[31m" + fmt.Sprint(args...) + "\033[0m")
}

func errorf(format string, args ...interface{}) {
	printfRed(format, args...)
}

func errorln(args ...interface{}) {
	printlnRed(args...)
}
