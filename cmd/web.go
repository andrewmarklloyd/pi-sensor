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

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/clients"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/google"
	"github.com/dghubble/sessions"
	gmux "github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
)

const (
	channelName          = "sensor"
	sessionName          = "pi-sensor"
	sessionUserKey       = "9024685F-97A4-441E-90D3-F0F11AA7A602"
	post                 = "post"
	get                  = "get"
	sensorListChannel    = "sensor/list"
	sensorRestartChannel = "sensor/restart"
	unauthPath           = "/unauth"
)

var sessionStore sessions.Store[any]

type WebServer struct {
	httpServer    *http.Server
	serverClients clients.ServerClients
	socketConn    *websocket.Conn
}

var allowedAPIKeys []string

func newWebServer(serverConfig config.ServerConfig, clients clients.ServerClients) *WebServer {
	allowedAPIKeys = serverConfig.AllowedAPIKeys
	router := gmux.NewRouter().StrictSlash(true)

	w := WebServer{
		serverClients: clients,
	}

	router.Handle("/ws/", http.HandlerFunc(w.serveWs))

	oauth2Config := &oauth2.Config{
		ClientID:     serverConfig.GoogleConfig.ClientId,
		ClientSecret: serverConfig.GoogleConfig.ClientSecret,
		RedirectURL:  serverConfig.GoogleConfig.RedirectURL,
		Endpoint:     googleOAuth2.Endpoint,
		Scopes:       []string{"profile", "email"},
	}
	sessionStore = sessions.NewCookieStore[any](nil, []byte(serverConfig.GoogleConfig.SessionSecret), nil)
	stateConfig := gologin.DefaultCookieConfig
	router.Handle("/health", http.HandlerFunc(healthHandler)).Methods(get)
	router.Handle("/api/sensor/restart", requireLogin(http.HandlerFunc(w.sensorRestartHandler))).Methods(post)
	router.Handle("/api/sensor/openTimeout", requireLogin(http.HandlerFunc(w.sensorOpenTimeoutHandler))).Methods(post)
	router.Handle("/api/sensor/getOpenTimeout", requireLogin(http.HandlerFunc(w.getSensorOpenTimeoutHandler))).Methods(get)
	router.Handle("/api/sensor/arming", requireLogin(http.HandlerFunc(w.sensorArmingHandler))).Methods(post)
	router.Handle("/api/sensor/all", requireLogin(http.HandlerFunc(w.allSensorsHandler))).Methods(get)
	router.Handle("/api/report", requireLogin(http.HandlerFunc(w.reportHandler))).Methods(get)
	router.Handle("/google/login", google.StateHandler(stateConfig, google.LoginHandler(oauth2Config, nil)))
	router.Handle("/google/callback", google.StateHandler(stateConfig, google.CallbackHandler(oauth2Config, issueSession(serverConfig), nil)))
	router.HandleFunc("/logout", logoutHandler)
	router.HandleFunc(unauthPath, unauthHandler).Methods(get)
	spa := spaHandler{
		staticPath: "frontend/dist",
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
	return &w
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
		logger.Errorf("Error getting messages: %s", err)
		http.Error(w, "Error getting report", http.StatusBadRequest)
		return
	}
	json, _ := json.Marshal(messages)
	fmt.Fprintf(w, `{"messages":%s,"numPages":%d}`, string(json), numPages)
}

func (s WebServer) sensorArmingHandler(w http.ResponseWriter, req *http.Request) {
	var p config.APIPayload
	err := json.NewDecoder(req.Body).Decode(&p)
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	if p.Source == "all" {
		armingState, armingStateErr := s.serverClients.Redis.ReadAllArming(req.Context())
		if armingStateErr != nil {
			http.Error(w, "error getting all arming statuses", http.StatusBadRequest)
			return
		}

		for k := range armingState {
			err = s.serverClients.Mosquitto.PublishHASensorArming(config.APIPayload{
				Source: k,
				Armed:  p.Armed,
			})
			if err != nil {
				logger.Errorf("error publishing ha sensor arming: %s", err)
			}

			err := s.serverClients.Redis.WriteArming(k, p.Armed, req.Context())
			if err != nil {
				http.Error(w, "error setting all arming statuses", http.StatusBadRequest)
				return
			}
		}

		fmt.Fprintf(w, `{"status":"success", "armed":"%s"}`, p.Armed)
		return
	}

	armed := "false"
	if p.Armed == "" {
		// this is a toggle api call, switch value
		armedString, _ := s.serverClients.Redis.ReadArming(p.Source, req.Context())
		if armedString == "" || armedString == "false" {
			armed = "true"
		}
	} else {
		// armed is specified
		armed = p.Armed
	}

	err = s.serverClients.Mosquitto.PublishHASensorArming(config.APIPayload{
		Source: p.Source,
		Armed:  armed,
	})
	if err != nil {
		logger.Errorf("error publishing ha sensor arming: %s", err)
	}

	err = s.serverClients.Redis.WriteArming(p.Source, armed, req.Context())
	if err != nil {
		logger.Errorf("error writing arming to redis: %s", err)
		fmt.Fprintf(w, `{"status":"error", "error":"%s"}`, err.Error())
		return
	}

	fmt.Fprintf(w, `{"status":"success", "armed":"%s"}`, armed)
}

func (s WebServer) sensorRestartHandler(w http.ResponseWriter, req *http.Request) {
	var sensor config.APIPayload
	err := json.NewDecoder(req.Body).Decode(&sensor)
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}
	err = s.serverClients.Mosquitto.PublishSensorRestart(sensor.Source)
	if err != nil {
		http.Error(w, "Error publishing restart message", http.StatusBadRequest)
		return
	}
	logger.Infof("Publishing sensor restart message for %s", sensor.Source)
	fmt.Fprintf(w, "{\"status\":\"success\"}")
}

func (s WebServer) getSensorOpenTimeoutHandler(w http.ResponseWriter, req *http.Request) {
	source := req.URL.Query().Get("source")

	if source == "" {
		http.Error(w, `{"status":"error","error":"must pass 'source' as query arg"}`, http.StatusBadRequest)
		return
	}

	allCfg, err := s.serverClients.Postgres.GetSensorConfig()
	if err != nil {
		logger.Errorf("getting sensor config: %s", err)
		http.Error(w, `{"status":"error","error":"Error getting open timeout from database"}`, http.StatusInternalServerError)
		return
	}

	openTimeout := config.DefaultOpenTimeoutMinutes
	for _, cfg := range allCfg {
		if cfg.Source == source {
			openTimeout = int(cfg.OpenTimeoutMinutes)
		}
	}

	fmt.Fprintf(w, `{"status":"success","openTimeout":%d}`, openTimeout)
}

func (s WebServer) sensorOpenTimeoutHandler(w http.ResponseWriter, req *http.Request) {
	var sensor config.APIPayload
	err := json.NewDecoder(req.Body).Decode(&sensor)
	if err != nil {
		logger.Errorf("unmarshalling sensor open timeout payload: %w", err)
		http.Error(w, `{"status":"error","error":"Error parsing request"}`, http.StatusBadRequest)
		return
	}

	if sensor.OpenTimeout < config.MinOpenTimeoutMinutes || sensor.OpenTimeout > config.MaxOpenTimeoutMinutes {
		msg := fmt.Sprintf("open timeout must be between %d and %d", config.MinOpenTimeoutMinutes, config.MaxOpenTimeoutMinutes)
		logger.Errorf("error in sensor open timeout handler: %s", msg)
		http.Error(w, fmt.Sprintf(`{"status":"error","error":"%s"}`, msg), http.StatusBadRequest)
		return
	}

	cfg := config.SensorConfig{
		Source:             sensor.Source,
		OpenTimeoutMinutes: int32(sensor.OpenTimeout),
	}
	err = s.serverClients.Postgres.WriteSensorConfig(cfg)
	if err != nil {
		logger.Errorf("writing sensor config: %s", err)
		http.Error(w, `{"status":"error","error":"Error writing open timeout to database"}`, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "{\"status\":\"success\"}")
}

func (s WebServer) allSensorsHandler(w http.ResponseWriter, req *http.Request) {
	sensors, err := s.serverClients.Redis.GetAllSensors(context.Background())
	if err != nil {
		logger.Errorf("Error getting all keys: %s", err)
		http.Error(w, "Error getting sensors", http.StatusBadRequest)
		return
	}
	json, _ := json.Marshal(sensors)
	fmt.Fprintf(w, `{"sensors":%s}`, string(json))
}

func (s *WebServer) SendMessage(channel string, status config.SensorStatus) {
	if s.socketConn == nil {
		return
	}

	statusJson, _ := json.Marshal(status)
	writer, err := s.socketConn.NextWriter(websocket.TextMessage)
	if err != nil {
		if e, ok := err.(*websocket.CloseError); ok {
			logger.Warnf("websocket close error %d getting websocket writer trying to send message: %s", e.Code, e)
			return
		}
		if strings.Contains(err.Error(), "broken pipe") {
			logger.Warnf("websocket close error broken pipe: %w", err)
			return
		}

		logger.Errorf("got unknown error getting websocket writer trying to send message: %s", err)
		return
	}

	message := config.WebsocketMessage{
		Message: string(statusJson),
		Channel: config.SensorStatusTopic,
	}

	jsonMessage, _ := json.Marshal(message)

	_, err = writer.Write(jsonMessage)
	if err != nil {
		logger.Errorf("writing websocket message: %s", err)
		return
	}

	if err := writer.Close(); err != nil {
		logger.Errorf("closing websocket writer: %s", err)
		return
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
		session.Set(sessionUserKey, googleUser.Id)
		session.Set("user-email", googleUser.Email)
		if err := session.Save(w); err != nil {
			logger.Errorf("error saving session: %s", err.Error())
			http.Redirect(w, req, unauthPath, http.StatusFound)
			return
		}
		http.Redirect(w, req, "/", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

func logoutHandler(w http.ResponseWriter, req *http.Request) {
	sessionStore.Destroy(w, sessionName)
	http.Redirect(w, req, unauthPath, http.StatusFound)
}

func unauthHandler(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, filepath.Join("frontend/dist", "unauth.html"))
}

// requireLogin redirects unauthenticated users to the login route.
func requireLogin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if !isAuthenticated(req) {
			if !strings.Contains(req.Header.Get("User-Agent"), "DigitalOcean Uptime Probe") {
				// logger.Warnf("Unauthenticated request, X-Forwarded-For: %s, User-Agent: %s", req.Header["X-Forwarded-For"], req.Header["User-Agent"])
				http.Redirect(w, req, "/google/login", http.StatusFound)
			}
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

	if validAPIKey(req.Header.Get("api-key")) {
		return true
	}

	return false
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	apiKey := req.Header.Get("api-key")

	if !validAPIKey(apiKey) {
		http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(w, `{"version":"%s"}`, version)
}

func validAPIKey(apiKey string) bool {
	if apiKey == "" {
		return false
	}
	allowed := false
	for _, key := range allowedAPIKeys {
		if key == apiKey {
			allowed = true
		}
	}
	return allowed
}
