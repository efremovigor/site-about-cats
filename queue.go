package main

import (
	"fmt"
)

var logChannel = make(chan LogChannel, 10)

type LogChannel struct {
	Message string
}

func LoggerHandle() {
	for {
		select {
		case task := <-logChannel:
			senLog(&task)
		}
	}
}

func senLog(task *LogChannel) {
	fmt.Println(task.Message)
}
