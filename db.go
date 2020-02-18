package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"time"
)

type KittenDb struct {
	KittenId    int
	Name        string
	Description string
	Modified    string
}

type KittenImgDb struct {
	KittenImgId int
	KittenId    int
	Url         string
	Modified    string
}

type KittenTaskDb struct {
	KittenTaskId int              `json:"kitten_task_id"`
	Status       int              `json:"status"`
	Data         KittenTaskDbData `json:"data" db:"data"`
	Modified     time.Time        `json:"modified"`
	Created      time.Time        `json:"created"`
}

type KittenTaskDbData struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Imgs        []string `json:"imgs"`
}

func (data KittenTaskDbData) Value() (driver.Value, error) {
	if &data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		return string(b), nil
	}
	return nil, nil
}

// Scan implements the sql.Scanner interface
func (data *KittenTaskDbData) Scan(src interface{}) error {
	if src != nil {
		err := json.Unmarshal([]byte(src.(string)), &data)
		if err != nil {
			panic(err)
		}

	}
	return nil
}

func getConnectionToDb() (db *sql.DB) {
	db, err := sql.Open("sqlite3", "storage/identifier.sqlite")
	if err != nil {
		logChannel <- LogChannel{Message: err.Error()}
	}
	return
}

type KittenView struct {
	Id          int              `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Imgs        []*KittenImgView `json:"imgs"`
}

type KittenImgView struct {
	Url string `json:"url"`
}

func createKittenTask(name string, desc string, imgs []string) (int64, error) {
	db := getConnectionToDb()
	defer db.Close()
	res, err := db.Exec("INSERT INTO kitten_task (status, data,modified,created)	VALUES ($1, $2, $3, $4)", statusNew, KittenTaskDbData{Name: name, Description: desc, Imgs: imgs}, time.Now(), time.Now())
	if err != nil {
		logChannel <- LogChannel{Message: err.Error()}
	}
	return res.LastInsertId()
}

func GetKittenTasks(count int, status int) []KittenTaskDb {
	db := getConnectionToDb()
	defer db.Close()
	rows, err := db.Query("SELECT kitten_task_id,status, data FROM kitten_task WHERE status = $1 LIMIT $2", status, count)
	defer rows.Close()
	if err != nil {
		logChannel <- LogChannel{Message: err.Error()}
	}
	var tasks []KittenTaskDb
	for rows.Next() {
		task := &KittenTaskDb{}
		err = rows.Scan(&task.KittenTaskId, &task.Status, &task.Data)
		tasks = append(tasks, *task)
	}
	return tasks
}

func updateKittenTask(task KittenTaskDb) {
	db := getConnectionToDb()
	defer db.Close()
	if _, err := db.Exec("UPDATE kitten_task SET status=$1, data=$2, modified=$3 where kitten_task_id = $4", task.Status, task.Data, time.Now(), task.KittenTaskId); err != nil {
		logChannel <- LogChannel{Message: err.Error()}
	}
	writeToEveryone("Задание №" + strconv.Itoa(task.KittenTaskId) + " получило статус: \"" + statusMapName[task.Status] + "\"")
	if task.Status == statusDone || task.Status == statusWithError {
		generalChan <- createSocketMessage(socketMessageEvent, "reload_catalog")
	}
}

func getKittensCatalog() (kittens []*KittenView) {
	db := getConnectionToDb()
	defer db.Close()

	rows, err := db.Query("SELECT kitten.kitten_id,kitten.name,kitten.description,kitten_img.url FROM kitten LEFT JOIN kitten_img ON kitten_img.kitten_id = kitten.kitten_id")
	if err != nil {
		logChannel <- LogChannel{Message: err.Error()}
	}

	kittensMap := map[int]*KittenView{}

	for rows.Next() {
		kitten := &KittenView{}
		kittenImg := &KittenImgView{}
		err := rows.Scan(&kitten.Id, &kitten.Name, &kitten.Description, &kittenImg.Url)
		if err != nil {
			panic(err)
		}

		if _, ok := kittensMap[kitten.Id]; !ok {
			kitten.Imgs = []*KittenImgView{kittenImg}
			kittensMap[kitten.Id] = kitten
		} else {
			kitten = kittensMap[kitten.Id]
			kitten.Imgs = append(kitten.Imgs, kittenImg)
			kittensMap[kitten.Id] = kitten
			continue
		}

		kittens = append(kittens, kitten)
	}
	return
}

func createKitten(kitten *KittenDb, imgs []KittenImgDb) {
	db := getConnectionToDb()
	defer db.Close()
	transaction, err := db.Begin()
	if err != nil {
		rollback(transaction, LogChannel{Message: "Error creating transaction"})
		return
	}
	res, err := db.Exec("INSERT INTO kitten (name,description ,modified,created)	VALUES ($1, $2, $3, $4)", kitten.Name, kitten.Description, time.Now(), time.Now())
	if err != nil {
		rollback(transaction, LogChannel{Message: err.Error()})
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		rollback(transaction, LogChannel{Message: err.Error()})
		return
	}
	kitten.KittenId = int(id)

	for _, img := range imgs {
		if _, err := db.Exec("INSERT INTO kitten_img (kitten_id, url ,modified,created)	VALUES ($1, $2, $3, $4)", kitten.KittenId, img.Url, time.Now(), time.Now()); err != nil {
			rollback(transaction, LogChannel{Message: err.Error()})
			return
		}
	}
	if err := transaction.Commit(); err != nil {
		rollback(transaction, LogChannel{Message: err.Error()})
	}
}

func rollback(transaction *sql.Tx, log LogChannel) {
	logChannel <- log
	if err := transaction.Rollback(); err != nil {
		logChannel <- LogChannel{Message: err.Error()}
	}
}
