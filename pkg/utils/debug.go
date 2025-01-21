package utils

import "fmt"

func PanicIf(condition bool, msg string, args ...any) {
	if condition {
		panic(fmt.Sprintf(msg, args...))
	}
}
