package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got connection")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	writer, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = writer.Write([]byte("hello"))
	if err != nil {
		panic(err)
	}

	if err := writer.Close(); err != nil {
		panic(err)
		return
	}

	for {
		_, message, err := conn.ReadMessage()
		fmt.Println(string(message))
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

	}

}

/*
conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	writer, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = w.Write([]byte("hello"))
	if err != nil {
		panic(err)
	}

	if err := writer.Close(); err != nil {
		panic(err)
		return
	}
*/
