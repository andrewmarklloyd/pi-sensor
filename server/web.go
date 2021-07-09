package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/google"
	"github.com/dghubble/sessions"
	gmux "github.com/gorilla/mux"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
)

const (
	// Maximum message size allowed from peer.
	maxMessageSize = 8192
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	channelName          = "sensor"
	sessionName          = "pi-sensor" // TODO need to make this dynamic?
	sessionUserKey       = "9024685F-97A4-441E-90D3-F0F11AA7A602"
	post                 = "post"
	get                  = "get"
	sensorListChannel    = "sensor/list"
	sensorRestartChannel = "sensor/restart"
	unauthPath           = "/unauth"
)

var sessionStore *sessions.CookieStore

type newClientHandlerFunc func()

type webServer struct {
	httpServer   *http.Server
	socketServer *gosocketio.Server
}

func newWebServer(serverConfig ServerConfig, newClientHandler newClientHandlerFunc, sensorRestartHandler http.HandlerFunc, sensorArmingHandler http.HandlerFunc) webServer {
	router := gmux.NewRouter().StrictSlash(true)
	socketServer := gosocketio.NewServer(transport.GetDefaultWebsocketTransport())
	socketServer.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
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
	sessionStore = sessions.NewCookieStore([]byte(serverConfig.googleConfig.sessionSecret), nil)
	stateConfig := gologin.DebugOnlyCookieConfig
	router.Handle("/api/sensor/restart", requireLogin(http.HandlerFunc(sensorRestartHandler))).Methods(post)
	router.Handle("/api/sensor/arming", requireLogin(http.HandlerFunc(sensorArmingHandler))).Methods(post)
	router.Handle("/google/login", google.StateHandler(stateConfig, google.LoginHandler(oauth2Config, nil)))
	router.Handle("/google/callback", google.StateHandler(stateConfig, google.CallbackHandler(oauth2Config, issueSession(serverConfig), nil)))
	router.HandleFunc("/logout", logoutHandler)
	router.HandleFunc(unauthPath, unauthHandler).Methods(get)
	spa := spaHandler{staticPath: "frontend/build", indexPath: "index.html"}
	router.PathPrefix("/").Handler(requireLogin(spa))

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

func (s webServer) sendMessage(channel string, message Message) {
	s.socketServer.BroadcastToAll(channel, message)
}

func (s webServer) sendSensorState(sensors map[string]string, arming map[string]string) {
	sensorList := SensorState{}
	for _, v := range sensors {
		m := toStruct(v)
		sensorList.Sensors = append(sensorList.Sensors, m)
	}
	armingMap := make(map[string]string)
	for k, v := range arming {
		t := strings.Replace(k, armingPrefix, "", -1)
		armingMap[t] = v
	}
	sensorList.Arming = armingMap
	json, _ := json.Marshal(sensorList)
	s.socketServer.BroadcastToAll(sensorListChannel, string(json))
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

// issueSession issues a cookie session after successful Google login
func issueSession(serverConfig ServerConfig) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		googleUser, err := google.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !strings.Contains(serverConfig.googleConfig.authorizedUsers, googleUser.Email) {
			http.Redirect(w, req, unauthPath, http.StatusFound)
			return
		}
		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = googleUser.Id
		session.Save(w)
		http.Redirect(w, req, "/", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

func logoutHandler(w http.ResponseWriter, req *http.Request) {
	sessionStore.Destroy(w, sessionName)
	http.Redirect(w, req, unauthPath, http.StatusFound)
}

func unauthHandler(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, filepath.Join("frontend/build", "unauth.html"))
}

// requireLogin redirects unauthenticated users to the login route.
func requireLogin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if !isAuthenticated(req) {
			http.Redirect(w, req, "/google/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

// isAuthenticated returns true if the user has a signed session cookie.
func isAuthenticated(req *http.Request) bool {
	if _, err := sessionStore.Get(req, sessionName); err == nil {
		return true
	}
	return false
}
