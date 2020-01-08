package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
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

func createKittenTask(name string, desc string, imgs []string) {
	db := getConnectionToDb()
	defer db.Close()
	_, err := db.Exec("INSERT INTO kitten_task (status, data,modified,created)	VALUES ($1, $2, $3, $4)", statusNew, KittenTaskDbData{Name: name, Description: desc, Imgs: imgs}, time.Now(), time.Now())
	if err != nil {
		panic(err)
	}
}

func getNewKittenTasks() []KittenTaskDb {
	db := getConnectionToDb()
	defer db.Close()
	transaction, err := db.Begin()
	if err != nil {
		transaction.Rollback()
		panic(err)
	}
	rows, err := transaction.Query("SELECT kitten_task_id, data FROM kitten_task WHERE status = $1 LIMIT 3", statusNew)
	if err != nil {
		transaction.Rollback()
		panic(err)
	}
	transaction.Commit()

	for rows.Next() {
		task := new(KittenTaskDb)
		err = rows.Scan(&task.KittenTaskId, &task.Data)
		return []KittenTaskDb{*task}
	}
	return []KittenTaskDb{}
}

func getKittensCatalog() (kittens []*KittenView) {
	db := getConnectionToDb()
	defer db.Close()

	rows, err := db.Query("select kitten.kitten_id,kitten.name,kitten.description,kitten_img.url from kitten LEFT JOIN kitten_img ON kitten_img.kitten_id = kitten.kitten_id")
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
