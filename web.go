package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var store = sessions.NewCookieStore([]byte("my-secret-key"))

type JsonSenderTopicResponse struct {
	Ok       bool                        `json:"ok"`
	Data     JsonSenderTopicResponseData `json:"data"`
	UserName string                      `json:"user_name"`
}

type JsonSenderTopicResponseData struct {
	TaskId int64 `json:"taskId"`
}

type KittensCatalogJsonResponse struct {
	Kittens []*KittenView `json:"kittens"`
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "super-cookies")
	session.Values["username"] = "new - " + time.Now().String()
	http.SetCookie(w, &http.Cookie{Name: "super-puper", Value: session.Values["username"].(string)})

	session.Save(r, w)

	w.WriteHeader(http.StatusOK)
	tmpl, _ := template.ParseFiles(templatePath + "index.html")
	tmpl.Execute(w, "")
}

func ApiTopicSender(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("name")
	description := r.PostFormValue("description")

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

	id, err := createKittenTask(name, description, []string{fileHeaders.Filename})
	logChannel <- LogChannel{Message: fmt.Sprintf("New kitten topic with id: %d, name:%s, desc:%s ", id, name, description)}
	writeToEveryone("Задание на добавление \"" + name + "\" - Принята №" + strconv.Itoa(int(id)))
	session, _ := store.Get(r, "cookie-name")

	data, _ := json.Marshal(JsonSenderTopicResponse{Ok: true, Data: JsonSenderTopicResponseData{TaskId: id}, UserName: session.Values["username"].(string)})
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(string(data))))
	fmt.Fprintln(w, string(data))
}

func ApiGetKittens(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	data, _ := json.Marshal(KittensCatalogJsonResponse{Kittens: getKittensCatalog()})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(string(data))))
	fmt.Fprintln(w, string(data))

}

func runWebServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", IndexHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/topic/send", ApiTopicSender).Methods(http.MethodPost)
	router.HandleFunc("/api/catalog", ApiGetKittens).Methods(http.MethodGet)

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
