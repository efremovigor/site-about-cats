package main

import (
	"os"
	"time"
)

const statusNew = 1
const statusInProgress = 2
const statusDone = 3
const statusDecline = 4
const statusWithError = 5

func KittenTaskHandle() {
	for {
		time.Sleep(1000)
		tasts := getKittenTasks(3, statusNew)
		for _, task := range tasts {
			task.Status = statusInProgress
			updateKittenTask(task)
			kittenTaskProcess(&task)
			updateKittenTask(task)
		}

	}
}

func kittenTaskProcess(task *KittenTaskDb) {
	if len(task.Data.Imgs) == 0 {
		task.Status = statusDecline
		return
	}
	for _, img := range task.Data.Imgs {
		if err := os.Rename(storageTmpFilePath+img, kittenImgPath+pathSeparator+kittenId+pathSeparator+img); err != nil {
			task.Status = statusWithError
			return
		}
	}
}
