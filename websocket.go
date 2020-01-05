package main

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var addr = flag.String("addr", socketWebSocket, "http service address")

var generalChan = make(chan []byte)

var upgrader = websocket.Upgrader{}
var connections = make(map[int]*connectionReceiver)

type connectionReceiver struct {
	conn         *websocket.Conn
	readChan     chan []byte
	writeChan    chan []byte
	closeConnect chan int
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	receiver := &connectionReceiver{conn: conn, readChan: make(chan []byte), writeChan: make(chan []byte), closeConnect: make(chan int)}
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
					generalChan <- []byte("General hello message")
					continue
				}
				receiver.readChan <- message
			}
		}(receiver)

		go func(receiver *connectionReceiver) {
			for {
				err := conn.WriteMessage(1, <-receiver.readChan)
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

func runWebSocketProcess() {
	go runWebSocketServer()
	for {
		select {
		case message := <-generalChan:
			for key, receiver := range connections {
				err := receiver.conn.WriteMessage(1, message)
				if err != nil {
					receiver.closeConnect <- 0
					delete(connections, key)
					continue
				}
			}
		}
	}
}

func runWebSocketServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", WebSocketHandler)
	log.Fatal(http.ListenAndServe(*addr, router))
}
