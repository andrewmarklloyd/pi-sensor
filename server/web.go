package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
	gmux "github.com/gorilla/mux"
)

const (
	publicDir = "/frontend/build/"

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	channelName = "sensor"
)

var newClientHandlerFunc func()

var channel *gosocketio.Channel

type webServer struct {
	server *http.Server
}

func newWebServer(port string, sensorHandler http.HandlerFunc) webServer {
	router := gmux.NewRouter().StrictSlash(true)
	server := gosocketio.NewServer(transport.GetDefaultWebsocketTransport())
	server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		logger.Println("New client connected")
		channel = c
		channel.Join(channelName)
	})
	router.Handle("/socket.io/", server)
	router.Handle("/sensors", sensorHandler)
	spa := spaHandler{staticPath: "frontend/build", indexPath: "index.html"}
	router.PathPrefix("/").Handler(spa)

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return webServer{
		server: srv,
	}
}

func (s webServer) startServer() {
	logger.Println("Starting web server")
	logger.Fatal(s.server.ListenAndServe())
}

func (s webServer) sendMessage(source string, data string) {
	if channel != nil {
		logger.Println("sending message", source, data)
		channel.BroadcastTo(channelName, source, data)
	} else {
		logger.Println("channel is nil")
	}
}

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// if we failed to get the absolute path respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(h.staticPath, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}
