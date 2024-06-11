package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader          = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	activeConnections int32
	messagesReceived  int32
)

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		conn.Close()
		atomic.AddInt32(&activeConnections, -1)
	}()

	atomic.AddInt32(&activeConnections, 1)
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			break
		}
		atomic.AddInt32(&messagesReceived, 1)
		if err := conn.WriteMessage(messageType, p); err != nil {
			fmt.Println(err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/", websocketHandler)
	go func() {
		for {
			log.Printf("Active connections: %d, Messages received: %d", atomic.LoadInt32(&activeConnections), atomic.LoadInt32(&messagesReceived))
			time.Sleep(1 * time.Second)
		}
	}()
	log.Fatal(http.ListenAndServe(":9000", nil))
}
