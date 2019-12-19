package main

import (
	"fmt"
)

var kittenChan = make(chan KittenTaskQueue)

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
