package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const signalUpServer = 1
const signalDownServer = 2
const signalReloadServer = 3

var store = sessions.NewCookieStore([]byte(Config.current.Session.UniKey))
var webServerProcess = WebServerProcess{ReloadChan: make(chan int)}

func getSession(r *http.Request) (session *sessions.Session) {
	session, _ = store.Get(r, sessionUserUniKey)
	return
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

type WebServerProcess struct {
	Current    *WebServerInstance
	New        *WebServerInstance
	Router     mux.Router
	RTimeout   time.Duration
	WTimeout   time.Duration
	NeedReload bool
	ReloadChan chan int
}

type WebServerInstance struct {
	Server http.Server
	Chan   chan int
	Group  sync.WaitGroup
	Host   string
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

type SimpleResponse struct {
	Success bool `json:"success"`
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if err := session.Save(r, w); err != nil {
		logChannel <- LogChannel{Message: "session wasn't save"}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", Config.current.getWebTcpSocket())
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
	response := SimpleResponse{}

	if request.Login == adminLogin && request.Password == adminPassword {
		session.Values["AUTH_SID"] = GetMD5Hash(session.ID + authSalt)
		if err := session.Save(r, w); err != nil {
			logChannel <- LogChannel{Message: "session wasn't save"}
		}

		http.SetCookie(w, &http.Cookie{Name: "AUTH_SID", Value: GetMD5Hash(session.ID + authSalt)})
		response.Success = true

	} else {
		response.Success = false
	}
	data, _ := json.Marshal(response)
	sendOkResponse(w, string(data))
}

func ApiGetConfig(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(Config)
	sendOkResponse(w, string(data))
}

func ApiSetConfig(w http.ResponseWriter, r *http.Request) {
	json.NewDecoder(r.Body).Decode(&Config.new)
	if Config.current.Web.Ip != Config.new.Web.Ip || Config.current.Web.Port != Config.new.Web.Port {
		webServerProcess.New = createNewWebInstance(webServerProcess, Config.new.getWebTcpSocket())
		webServerProcess.NeedReload = true
		logChannel <- LogChannel{Message: fmt.Sprintf("Назначен новый адресс для web-server - %s", webServerProcess.New.Host)}
	}

	if Config.current.WebSocket.Ip != Config.new.WebSocket.Ip || Config.current.WebSocket.Port != Config.new.WebSocket.Port {
		webSocketServerProcess.New = createNewWebInstance(webServerProcess, Config.new.getWebSocketTcpSocket())
		webSocketServerProcess.NeedReload = true
		logChannel <- LogChannel{Message: fmt.Sprintf("Назначен новый адресс для websocket-server - %s", webSocketServerProcess.New.Host)}
	}

	data, _ := json.Marshal(Config.new)

	sendOkResponse(w, string(data))
}

func ApiReloadService(w http.ResponseWriter, r *http.Request) {
	reloadServer()
	data, _ := json.Marshal(SimpleResponse{Success: true})
	sendOkResponse(w, string(data))
}

func sendOkResponse(w http.ResponseWriter, data string) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", Config.current.getWebTcpSocket())
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	fmt.Fprintln(w, data)
}

func startWebServer(instance *WebServerInstance) {
	defer instance.Group.Done()
	if err := instance.Server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}
}

func reloadServer() {
	Config.switchConfig()

	if webServerProcess.NeedReload {
		logChannel <- LogChannel{Message: "Перегружаем web-server"}
		webServerProcess.ReloadChan <- signalReloadServer
		webServerProcess.NeedReload = false
	}

	if webSocketServerProcess.NeedReload {
		logChannel <- LogChannel{Message: "Перегружаем websocket-server"}
		webSocketServerProcess.ReloadChan <- signalReloadServer
		webSocketServerProcess.NeedReload = false
	}
}

func (process *WebServerProcess) run() {
	go process.Current.run()
	go func(process *WebServerProcess) {
		for {
			<-process.ReloadChan
			time.Sleep(2 * time.Second)
			if _, err := net.DialTimeout("tcp", process.New.Host, time.Second); err == nil {
				logChannel <- LogChannel{Message: fmt.Sprintf("Socket error - %s", err)}
				return
			}
			process.New = createNewWebInstance(*process, process.New.Host)
			go process.New.run()
			process.New.Chan <- signalUpServer
			time.Sleep(2 * time.Second)

			process.Current.Chan <- signalDownServer
			time.Sleep(2 * time.Second)
			process.Current = process.New
			process.New = &WebServerInstance{}
		}
	}(process)
	process.Current.Chan <- signalUpServer
}

func (instance *WebServerInstance) run() {
	for {
		switch <-instance.Chan {
		case signalUpServer:
			logChannel <- LogChannel{Message: fmt.Sprintf("Start web-server: host %s", instance.Host)}
			instance.Group = sync.WaitGroup{}
			instance.Group.Add(1)
			go startWebServer(instance)
			logChannel <- LogChannel{Message: fmt.Sprintf("Web-server started(%s)", instance.Host)}
		case signalDownServer:
			logChannel <- LogChannel{Message: fmt.Sprintf("Stop web-server(%s)", instance.Host)}
			if err := instance.Server.Shutdown(context.TODO()); err != nil {
				panic(err)
			}
			instance.Group.Wait()
			logChannel <- LogChannel{Message: fmt.Sprintf("Web-server stoped(%s)", instance.Host)}
			return
		}
	}
}

func createNewWebInstance(process WebServerProcess, host string) (instance *WebServerInstance) {

	instance = &WebServerInstance{Chan: make(chan int), Host: host}

	instance.Server = http.Server{
		ReadTimeout:  process.RTimeout,
		WriteTimeout: process.WTimeout,
		Addr:         host,
		Handler:      &process.Router,
	}
	return
}

func runWebServerHandler() {
	webServerProcess.Router = *mux.NewRouter()
	webServerProcess.Router.HandleFunc("/", IndexHandler).Methods(http.MethodGet).Name("qwe")
	webServerProcess.Router.HandleFunc("/api/topic/send", ApiTopicSender).Methods(http.MethodPost)
	webServerProcess.Router.HandleFunc("/api/catalog", ApiGetKittens).Methods(http.MethodGet)
	webServerProcess.Router.HandleFunc("/api/login", ApiLogin).Methods(http.MethodPost)
	webServerProcess.Router.HandleFunc("/api/admin/config", ApiGetConfig).Methods(http.MethodGet)
	webServerProcess.Router.HandleFunc("/api/admin/config", ApiSetConfig).Methods(http.MethodPost)
	webServerProcess.Router.HandleFunc("/api/admin/reload-config", ApiReloadService).Methods(http.MethodPost)
	webServerProcess.Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(publicPath))))

	webServerProcess.RTimeout = readTimeoutRequest
	webServerProcess.WTimeout = writeTimeoutRequest

	webServerProcess.Current = createNewWebInstance(webServerProcess, Config.current.getWebTcpSocket())

	webServerProcess.run()
}
