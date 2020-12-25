package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/gorilla/websocket"
)

const (
	// Maximum message size allowed from peer.
	maxMessageSize = 8192
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
)

var socketIOServer *socketio.Server
var upgrader = websocket.Upgrader{}

// NewServer returns a new ServeMux with app routes.
func NewServer() {
	// Set the router as the default one shipped with Gin
	router := gin.Default()

	var err error
	socketIOServer, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Serve frontend static files
	router.Use(static.Serve("/", static.LocalFile("./frontend/build", true)))
	router.GET("/socket.io", socketHandler)
	router.POST("/socket.io", socketHandler)
	router.Handle("WS", "/socket.io", gin.HandlerFunc(socketHandler))
	router.Handle("WSS", "/socket.io", gin.HandlerFunc(socketHandler))

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("PORT must be set")
	}

	fmt.Println(fmt.Sprintf("Running web server on port %s", port))
	router.Run(fmt.Sprintf(":%s", port))
}

func socketHandler(c *gin.Context) {
	socketIOServer.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})

	socketIOServer.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	socketIOServer.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})

	socketIOServer.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})

	socketIOServer.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})

	socketIOServer.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("closed", reason)
	})

}
