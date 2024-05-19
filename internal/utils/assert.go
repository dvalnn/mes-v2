package utils

import "log"

func Assert(condition bool, message string) {
	if !condition {
		log.Panicln("[Assertion Failed]", message)
	}
}

type Assertion struct {
	Message   string
	Condition bool
}

func AssertMultiple(prefix string, assertions []Assertion) {
	for _, a := range assertions {
		if !a.Condition {
			log.Panicln("[Assertion Failed]", a.Message)
		}
	}
}
