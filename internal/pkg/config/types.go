package config

import "time"

const (
	SensorStatusTopic           = "sensor/status"
	SensorHeartbeatTopic        = "sensor/heartbeat"
	SensorRestartTopic          = "sensor/restart"
	HASensorStatusTopic         = "ha/sensor/status"
	HASensorStatusOpenWarnTopic = "ha/sensor/open/warn"
	HASensorLostConnectionTopic = "ha/sensor/connectionlost"
	HASensorArmingTopic         = "ha/sensor/arming"

	OPEN    = "OPEN"
	CLOSED  = "CLOSED"
	UNKNOWN = "UNKNOWN"

	HeartbeatTypeSensor = "sensor"
	HeartbeatTypeApp    = "app"

	OpenTimeout      = 5 * time.Minute
	HeartbeatTimeout = 5 * time.Minute
)

type Heartbeat struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

type SensorStatus struct {
	Source    string `json:"source"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

type SensorState struct {
	Sensors []SensorStatus    `json:"sensors"`
	Arming  map[string]string `json:"arming"`
}

type APIPayload struct {
	Source string `json:"source"`
	Armed  string `json:"armed"` // TODO: use bool. Need to handle zero value
}
