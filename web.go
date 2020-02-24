package main

import (
	"crypto/md5"
	"encoding/hex"
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
)

var store = sessions.NewCookieStore([]byte(sessionUniKey))

func getSession(r *http.Request) (session *sessions.Session) {
	session, _ = store.Get(r, sessionUserUniKey)
	return
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

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

type LoginJsonRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if err := session.Save(r, w); err != nil {
		logChannel <- LogChannel{Message: "session wasn't save"}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", domain)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	tmpl, _ := template.ParseFiles(templatePath + "index.html")
	if err := tmpl.Execute(w, ""); err != nil {
		logChannel <- LogChannel{Message: "template error"}
	}
}

func ApiTopicSender(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("kittenName")
	description := r.PostFormValue("kittenDesc")

	file, fileHeaders, err := r.FormFile("kittenImage")
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

	data, _ := json.Marshal(JsonSenderTopicResponse{Ok: true, Data: JsonSenderTopicResponseData{TaskId: id}})
	sendOkResponse(w, string(data))
}

func ApiGetKittens(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(KittensCatalogJsonResponse{Kittens: getKittensCatalog()})
	sendOkResponse(w, string(data))
}

func ApiLogin(w http.ResponseWriter, r *http.Request) {
	var request LoginJsonRequest
	session := getSession(r)
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || session.IsNew == true {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var data string
	if request.Login == adminLogin && request.Password == adminPassword {
		session.Values["AUTH_SID"] = GetMD5Hash(session.ID + authSalt)
		if err := session.Save(r, w); err != nil {
			logChannel <- LogChannel{Message: "session wasn't save"}
		}

		http.SetCookie(w, &http.Cookie{Name: "AUTH_SID", Value: GetMD5Hash(session.ID + authSalt)})
		data = "{success:true}"

	} else {
		data = "{success:false}"
	}

	sendOkResponse(w, data)
}

func sendOkResponse(w http.ResponseWriter, data string) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", ip+":"+port)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	fmt.Fprintln(w, data)
}

func runWebServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", IndexHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/topic/send", ApiTopicSender).Methods(http.MethodPost)
	router.HandleFunc("/api/catalog", ApiGetKittens).Methods(http.MethodGet)
	router.HandleFunc("/api/login", ApiLogin).Methods(http.MethodPost)

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
