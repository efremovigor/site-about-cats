package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt" // пакет для форматированного ввода вывода
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io"
	"log"      // пакет для логирования
	"net/http" // пакет для поддержки HTTP протокола
	"os"
	"time"
)

const ip = "127.0.0.1"
const port = "9000"
const webSocketPort = "9001"
const socket = ip + ":" + port
const socketWebSocket = ip + ":" + webSocketPort
const readTimeoutRequest = 5 * time.Second
const writeTimeoutRequest = 10 * time.Second
const pathSeparator = "/"
const publicPath = "public" + pathSeparator
const templatePath = publicPath + "templates" + pathSeparator
const storagePath = "storage" + pathSeparator
const storageTmpFilePath = storagePath + "tmp" + pathSeparator

type JsonResponse struct {
	Ok   bool   `json:"ok"`
	Name string `json:"name"`
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)
	tmpl, _ := template.ParseFiles(templatePath + "index.html")
	tmpl.Execute(w, "")
}

func ApiTopicSender(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	data, _ := json.Marshal(JsonResponse{Ok: true, Name: r.PostFormValue("name")})

	file, fileHeaders, err := r.FormFile("fileupload")
	if err != nil {
		return
	}

	defer file.Close()

	// copy example
	f, err := os.OpenFile(storageTmpFilePath+fileHeaders.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	io.Copy(f, file)

	fmt.Fprintln(w, string(data))
}

func ApiGetKittens() {

}

var addr = flag.String("addr", socketWebSocket, "http service address")

var upgrader = websocket.Upgrader{} // use default options

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {

		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}

		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func createWebSocketServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", WebSocketHandler)
	log.Fatal(http.ListenAndServe(*addr, router))
}

func createWebServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", IndexHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/topic/send", ApiTopicSender).Methods(http.MethodPost)

	var dir string
	flag.StringVar(&dir, "dir", ".", "the directory to serve files from. Defaults to the current dir")

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(publicPath+http.Dir(dir))))

	srv := &http.Server{
		ReadTimeout:  readTimeoutRequest,
		WriteTimeout: writeTimeoutRequest,
		Addr:         socket,
		Handler:      router,
	}
	log.Fatal(srv.ListenAndServe())
}

type Kitten struct {
	KittenId    int
	Name        string
	Description string
	Modified    string
}

type KittenImg struct {
	KittenImgId int
	KittenId    int
	Url         string
	Modified    string
}

type KittenTask struct {
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

type KittensCatalogJsonResponse struct {
	Kittens []KittenView `json:"kittens"`
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

func main() {
	go createWebServer()
	go createWebSocketServer()
	catalog := getKittensCatalog()
	fmt.Println(catalog)

	select {}
}
