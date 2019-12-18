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
