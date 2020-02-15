package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
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

func (k KittenTaskDbData) Value() (driver.Value, error) {
	if &k != nil {
		b, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		return string(b), nil
	}
	return nil, nil
}

// Scan implements the sql.Scanner interface
func (r KittenTaskDbData) Scan(src interface{}) error {
	return nil
}

func getConnectionToDb() (db *sql.DB) {
	db, err := sql.Open("sqlite3", "storage/identifier.sqlite")
	if err != nil {
		panic(err)
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
		panic(err)
	}
	return res.LastInsertId()
}

func getKittenTasks(count int, status int) []KittenTaskDb {
	db := getConnectionToDb()
	defer db.Close()
	rows, err := db.Query("SELECT kitten_task_id, data FROM kitten_task WHERE status = $1 LIMIT 3 FOR UPDATE", count, status)
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		task := new(KittenTaskDb)
		err = rows.Scan(&task.KittenTaskId, &task.Data)
		return []KittenTaskDb{*task}
	}
	return []KittenTaskDb{}
}

func updateKittenTask(task KittenTaskDb) {
	db := getConnectionToDb()
	defer db.Close()
	res, err := db.Exec("UPDATE kitten_task SET status=$1, data=$2, modified=$3 where kitten_task_id = $4", task.Status, task.Data, time.Now(), task.KittenTaskId)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}

func getKittensCatalog() (kittens []*KittenView) {
	db := getConnectionToDb()
	defer db.Close()

	rows, err := db.Query("SELECT kitten.kitten_id,kitten.name,kitten.description,kitten_img.url FROM kitten LEFT JOIN kitten_img ON kitten_img.kitten_id = kitten.kitten_id")
	if err != nil {
		panic(err)
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

func createKitten(kitten KittenDb, img []KittenImgDb) {
	db := getConnectionToDb()
	defer db.Close()
	transaction, err := db.Begin()
	if err != nil {
		transaction.Rollback()
		panic(err)
	}
	res, err := db.Exec("INSERT INTO kitten (kitten_id, name,description ,modified,created)	VALUES ($1, $2, $3, $4)", kitten.KittenId, kitten.Name, kitten.Description, time.Now(), time.Now())

	if err != nil {
		transaction.Rollback()
		panic(err)
	}
	transaction.Commit()
}
