package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

const socketMessageText = "info"
const socketMessageEvent = "event"

var generalChan = make(chan socketMessageInterface)

var upgrader = websocket.Upgrader{}
var connections = make(map[int]*connectionReceiver)
var webSocketServerProcess = WebServerProcess{ReloadChan: make(chan int)}

type (
	socketMessageInterface interface {
		getMessage() []byte
	}
	socketMessage struct {
		MessageType string `json:"message_type"`
		Data        string `json:"data"`
	}
	connectionReceiver struct {
		conn         *websocket.Conn
		readChan     chan socketMessageInterface
		writeChan    chan socketMessageInterface
		closeConnect chan int
	}
)

func (message socketMessage) getMessage() []byte {
	jsonData, _ := json.Marshal(&message)
	return jsonData
}

func createSocketMessage(messageType string, text string) socketMessage {
	return socketMessage{MessageType: messageType, Data: text}
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	receiver := &connectionReceiver{conn: conn, readChan: make(chan socketMessageInterface), writeChan: make(chan socketMessageInterface), closeConnect: make(chan int)}
	connectionId := len(connections)
	connections[connectionId] = receiver

	go func(connectionId int) {

		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		go func(receiver *connectionReceiver) {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					receiver.closeConnect <- 0
					break
				}
				if string(message) == "general" {
					generalChan <- createSocketMessage(socketMessageText, "General hello message")
					continue
				}
				receiver.readChan <- createSocketMessage(socketMessageText, string(message))
			}
		}(receiver)

		go func(receiver *connectionReceiver) {
			for {
				message := <-receiver.readChan
				err := conn.WriteMessage(1, message.getMessage())
				if err != nil {
					receiver.closeConnect <- 0
					break
				}
			}
		}(receiver)

		for {
			select {
			case <-receiver.closeConnect:
				defer conn.Close()
				break
			}
		}

	}(connectionId)

}

func runWebSocketHandler() {
	go runWebSocketServer()
	for {
		select {
		case message := <-generalChan:
			for key, receiver := range connections {
				err := receiver.conn.WriteMessage(1, message.getMessage())
				if err != nil {
					receiver.closeConnect <- 0
					delete(connections, key)
					continue
				}
			}
		}
	}
}

func writeToEveryone(message string) {
	generalChan <- createSocketMessage(socketMessageText, message)
}

func runWebSocketServer() {
	webSocketServerProcess.Router = mux.NewRouter()
	webSocketServerProcess.Router.HandleFunc("/", WebSocketHandler)
	webSocketServerProcess.Current = createNewWebInstance(webSocketServerProcess, Config.current.getWebSocketTcpSocket())
	webSocketServerProcess.run()
}
