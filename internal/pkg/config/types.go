package config

import "time"

const (
	SensorStatusTopic    = "sensor/status"
	SensorHeartbeatTopic = "sensor/heartbeat"
	SensorRestartTopic   = "sensor/restart"

	OPEN    = "OPEN"
	CLOSED  = "CLOSED"
	UNKNOWN = "UNKNOWN"

	HeartbeatTypeSensor = "sensor"
	HeartbeatTypeApp    = "app"

	OpenTimeout      = 5 * time.Minute
	HeartbeatTimeout = 5 * time.Minute
)

type Heartbeat struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type SensorStatus struct {
	Source    string `json:"source"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type SensorState struct {
	Sensors []SensorStatus    `json:"sensors"`
	Arming  map[string]string `json:"arming"`
}

type APIPayload struct {
	Source string `json:"source"`
}
