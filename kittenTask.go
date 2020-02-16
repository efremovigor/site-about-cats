package main

import (
	"os"
	"strconv"
)

const statusNew = 1
const statusInProgress = 2
const statusDone = 3
const statusDecline = 4
const statusWithError = 5

func KittenTaskHandle() {
	for {
		tasks := GetKittenTasks(3, statusNew)
		for _, task := range tasks {
			logChannel <- LogChannel{Message: "Took " + string(len(tasks)) + " tasks"}
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

	kitten := KittenDb{Name: task.Data.Name, Description: task.Data.Description}
	kittenImgs := []KittenImgDb{}

	for _, img := range task.Data.Imgs {
		kittenImgs = append(kittenImgs, KittenImgDb{Url: img})
	}

	createKitten(&kitten, kittenImgs)

	// Save image storage, move images
	for _, img := range task.Data.Imgs {
		kittenImgStoragePath := kittenImgPath + strconv.Itoa(kitten.KittenId) + pathSeparator
		if _, err := os.Stat(kittenImgStoragePath); os.IsNotExist(err) {
			os.MkdirAll(kittenImgStoragePath, os.ModePerm)
		}

		if err := os.Rename(storageTmpFilePath+img, kittenImgStoragePath+img); err != nil {
			task.Status = statusWithError
			return
		}
	}
	task.Status = statusDone
}
