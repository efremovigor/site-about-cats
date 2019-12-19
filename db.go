package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
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
	KittenTaskId int
	Status       int
	Data         string
	Modified     string
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

//func saveKitten()  {
//
//}

func getKittensCatalog() (kittens []*KittenView) {
	db := getConnectionToDb()

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
	defer db.Close()
	return
}
