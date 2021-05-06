package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/google"
	gmux "github.com/gorilla/mux"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
)

const (
	publicDir = "/frontend/build/"

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	channelName = "sensor"
)

type newClientHandlerFunc func()

var channel *gosocketio.Channel

type webServer struct {
	httpServer   *http.Server
	socketServer *gosocketio.Server
}

func newWebServer(serverConfig ServerConfig, newClientHandler newClientHandlerFunc) webServer {
	router := gmux.NewRouter().StrictSlash(true)
	socketServer := gosocketio.NewServer(transport.GetDefaultWebsocketTransport())
	socketServer.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		logger.Println("New client connected")
		channel = c
		// c.BroadcastTo("chat", "message", msg)
		newClientHandler()
	})
	router.Handle("/socket.io/", socketServer)
	oauth2Config := &oauth2.Config{
		ClientID:     serverConfig.googleConfig.clientId,
		ClientSecret: serverConfig.googleConfig.clientSecret,
		RedirectURL:  serverConfig.googleConfig.redirectUrl,
		Endpoint:     googleOAuth2.Endpoint,
		Scopes:       []string{"profile", "email"},
	}
	stateConfig := gologin.DebugOnlyCookieConfig
	router.Handle("/google/login", google.StateHandler(stateConfig, google.LoginHandler(oauth2Config, nil)))
	spa := spaHandler{staticPath: "frontend/build", indexPath: "index.html"}
	router.PathPrefix("/").Handler(spa)

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + serverConfig.port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return webServer{
		httpServer:   srv,
		socketServer: socketServer,
	}
}

func (s webServer) startServer() {
	logger.Println("Starting web server")
	logger.Fatal(s.httpServer.ListenAndServe())
}

func (s webServer) sendMessage(message string) {
	messageStruct := toStruct(message)
	s.socketServer.BroadcastToAll("sensor/status", messageStruct)
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
