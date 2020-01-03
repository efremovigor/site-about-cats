package main

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var addr = flag.String("addr", socketWebSocket, "http service address")

var upgrader = websocket.Upgrader{} // use default options

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)

	go func(conn *websocket.Conn) {
		readChan := make(chan []byte)
		closeConnect := make(chan int)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		go func(conn *websocket.Conn) {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					closeConnect <- 0
					break
				}
				readChan <- message
			}
		}(conn)

		go func(conn *websocket.Conn) {
			for {
				err := conn.WriteMessage(1, <-readChan)
				if err != nil {
					closeConnect <- 0
					break
				}
			}
		}(conn)
		for {
			select {
			case <-closeConnect:
				defer conn.Close()
				break
			}
		}

	}(conn)

}

func runWebSocketServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", WebSocketHandler)
	log.Fatal(http.ListenAndServe(*addr, router))
}
