package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	gmux "github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	publicDir = "/frontend/build/"

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
)

var ws *websocket.Conn

var upgrader = websocket.Upgrader{}

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
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

// NewServer returns a new ServeMux with app routes.
func NewServer() {
	// Set the router as the default one shipped with Gin
	router := gmux.NewRouter().StrictSlash(true)
	spa := spaHandler{staticPath: "frontend/build", indexPath: "index.html"}
	router.Handle("/ws", http.HandlerFunc(websocketHandler))
	router.PathPrefix("/").Handler(spa)

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("PORT must be set")
	}

	// allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	upgrader.CheckOrigin = func(r *http.Request) bool {
		// TODO: only allow specified origins
		return true
	}

	srv := &http.Server{
		Handler: router,
		Addr:    "0.0.0.0:" + port,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func send(example string) {
	if ws != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(example))
	}
}

func websocketHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	ws, err = upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	ws.SetReadLimit(maxMessageSize)
}
