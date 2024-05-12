package utils

import "log"

func Assert(condition bool, message string) {
	if !condition {
		log.Panicln("utils.Assertion failed:", message)
	}
}
