package main

import "time"

// Check contains a check that must be evaluatable to bool
type Check struct {
	Name        string
	TestCode    string
	IsOk        bool
	LastOk      time.Time
	OkSince     time.Time
	Description string
	ToDoIfBad   string
	Props       properties
}

// KnownChecks contains all the Networks we can access
var KnownChecks = make(map[string]*Check)

func checkmain() {

	KnownChecks["op1 up"] = &Check{Name: "op1 up", IsOk: true}
	KnownChecks["oblacklan up"] = &Check{Name: "oblacklan up", IsOk: false}
	KnownChecks["oclearlan up"] = &Check{Name: "oclearlan up", IsOk: false}
	KnownChecks["roscore @ oblacklan"] = &Check{Name: "roscore @ oblacklan", IsOk: false}
}
