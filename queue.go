package main

import (
	"fmt"
)

var kittenChan = make(chan KittenTaskQueue)

const statusNew = 1
const statusInProgress = 2
const statusDone = 3

type KittenTaskQueue struct {
	Id  int
	Img string
}

func runKittenQueue() {
	for {
		select {
		case task := <-kittenChan:
			kittenTaskProcess(&task)
			return
		}
	}
}

func kittenTaskProcess(task *KittenTaskQueue) {
	fmt.Println(task)
	fmt.Println(task)
}
