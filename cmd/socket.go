package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *WebServer) serveWs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got connection")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("upgrading websocket connection: %s", err)
		return
	}

	s.socketConn = conn

	sensorList, stateErr := s.serverClients.Redis.ReadAllState(context.Background())
	if stateErr != nil {
		logger.Errorf("new socket connection error: reading redis state: %s", stateErr)
		return
	}

	armingState, armingStateErr := s.serverClients.Redis.ReadAllArming(context.Background())
	if armingStateErr != nil {
		logger.Errorf("new socket connection error: reading arming state from redis: %s", armingStateErr)
		return
	}

	sensorState := config.SensorState{
		Sensors: sensorList,
		Arming:  armingState,
	}

	stateJson, _ := json.Marshal(sensorState)
	writer, err := s.socketConn.NextWriter(websocket.TextMessage)
	if err != nil {
		logger.Errorf("getting websocket writer on initial connection: %s", err)
		return
	}

	message := config.WebsocketMessage{
		Message: string(stateJson),
		Channel: sensorListChannel,
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
