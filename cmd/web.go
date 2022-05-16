package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/clients"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
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
	sessionName          = "pi-sensor"
	sessionUserKey       = "9024685F-97A4-441E-90D3-F0F11AA7A602"
	post                 = "post"
	get                  = "get"
	sensorListChannel    = "sensor/list"
	sensorRestartChannel = "sensor/restart"
	unauthPath           = "/unauth"
)

var sessionStore *sessions.CookieStore

type WebServer struct {
	httpServer    *http.Server
	socketServer  *gosocketio.Server
	serverClients clients.ServerClients
}

func newWebServer(serverConfig config.ServerConfig, clients clients.ServerClients) WebServer {

	router := gmux.NewRouter().StrictSlash(true)
	socketServer := gosocketio.NewServer(transport.GetDefaultWebsocketTransport())

	w := WebServer{
		serverClients: clients,
		socketServer:  socketServer,
	}
	socketServer.On(gosocketio.OnConnection, w.newSocketConnection)

	router.Handle("/socket.io/", socketServer)
	oauth2Config := &oauth2.Config{
		ClientID:     serverConfig.GoogleConfig.ClientId,
		ClientSecret: serverConfig.GoogleConfig.ClientSecret,
		RedirectURL:  serverConfig.GoogleConfig.RedirectURL,
		Endpoint:     googleOAuth2.Endpoint,
		Scopes:       []string{"profile", "email"},
	}
	sessionStore = sessions.NewCookieStore([]byte(serverConfig.GoogleConfig.SessionSecret), nil)
	stateConfig := gologin.DebugOnlyCookieConfig
	router.Handle("/health", http.HandlerFunc(healthHandler)).Methods(get)
	// router.Handle("/api/sensor/restart", requireLogin(http.HandlerFunc(sensorRestartHandler))).Methods(post)
	router.Handle("/api/sensor/arming", requireLogin(http.HandlerFunc(w.sensorArmingHandler))).Methods(post)
	router.Handle("/api/sensor/all", requireLogin(http.HandlerFunc(w.allSensorsHandler))).Methods(get)
	router.Handle("/api/report", requireLogin(http.HandlerFunc(w.reportHandler))).Methods(get)
	router.Handle("/google/login", google.StateHandler(stateConfig, google.LoginHandler(oauth2Config, nil)))
	router.Handle("/google/callback", google.StateHandler(stateConfig, google.CallbackHandler(oauth2Config, issueSession(serverConfig), nil)))
	router.HandleFunc("/logout", logoutHandler)
	router.HandleFunc(unauthPath, unauthHandler).Methods(get)
	spa := spaHandler{
		staticPath: "frontend/build",
		indexPath:  "index.html",
	}
	router.PathPrefix("/").Handler(requireLogin(spa))

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + serverConfig.Port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	w.httpServer = srv
	return w
}

func (s WebServer) reportHandler(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	sensor := query.Get("sensor")
	pageString := query.Get("page")
	page, err := strconv.Atoi(pageString)
	if err != nil {
		http.Error(w, "Page not found", http.StatusBadRequest)
		return
	}
	if sensor == "" {
		http.Error(w, "Pass sensor in request", http.StatusBadRequest)
		return
	}
	messages, numPages, err := s.serverClients.Postgres.GetSensorStatus(sensor, page)
	if err != nil {
		logger.Fatalln("Error getting messages", err)
		http.Error(w, "Error getting report", http.StatusBadRequest)
		return
	}
	json, _ := json.Marshal(messages)
	fmt.Fprintf(w, fmt.Sprintf(`{"messages":%s,"numPages":%d}`, string(json), numPages))
}

func (s WebServer) newSocketConnection(c *gosocketio.Channel) {
	sensorList, stateErr := s.serverClients.Redis.ReadAllState(context.Background())
	if stateErr != nil {
		logger.Println("new socket connection error: reading redis state:", stateErr)
		return
	}

	armingState, armingStateErr := s.serverClients.Redis.ReadAllArming(context.Background())
	if armingStateErr != nil {
		logger.Println("new socket connection error: reading arming state from redis:", armingStateErr)
		return
	}

	sensorState := config.SensorState{
		Sensors: sensorList,
		Arming:  armingState,
	}

	json, _ := json.Marshal(sensorState)
	s.socketServer.BroadcastToAll(sensorListChannel, string(json))
}

func (s WebServer) sensorArmingHandler(w http.ResponseWriter, req *http.Request) {
	var p config.ArmingPayload
	err := json.NewDecoder(req.Body).Decode(&p)
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}
	armedString, _ := s.serverClients.Redis.ReadArming(p.Source, context.Background())
	armed := "false"
	if armedString == "" || armedString == "false" {
		armed = "true"
	}

	s.serverClients.Redis.WriteArming(p.Source, armed, context.Background())
	fmt.Fprintf(w, fmt.Sprintf(`{"status":"success", "armed":"%s"}`, armed))
}

func (s WebServer) allSensorsHandler(w http.ResponseWriter, req *http.Request) {
	sensors, err := s.serverClients.Redis.GetAllSensors(context.Background())
	if err != nil {
		logger.Fatalln("Error getting all keys", err)
		http.Error(w, "Error getting sensors", http.StatusBadRequest)
		return
	}
	json, _ := json.Marshal(sensors)
	fmt.Fprintf(w, fmt.Sprintf(`{"sensors":%s}`, string(json)))
}

func (s WebServer) SendMessage(channel string, status config.SensorStatus) {
	s.socketServer.BroadcastToAll(channel, status)
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
func issueSession(serverConfig config.ServerConfig) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		googleUser, err := google.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !strings.Contains(serverConfig.GoogleConfig.AuthorizedUsers, googleUser.Email) {
			http.Redirect(w, req, unauthPath, http.StatusFound)
			return
		}
		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = googleUser.Id
		session.Save(w)
		fmt.Println(session.Values)
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

func healthHandler(w http.ResponseWriter, req *http.Request) {
	allowedApiKey := os.Getenv("SERVER_API_KEY")
	apiKey := req.Header.Get("api-key")
	if apiKey == "" {
		http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
		return
	}
	if apiKey != allowedApiKey {
		http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(w, fmt.Sprintf(`{"version":"%s"}`, version))
}