package config

import "time"

const (
	SensorStatusTopic    = "sensor/status"
	SensorHeartbeatTopic = "sensor/heartbeat"
	SensorRestartTopic   = "sensor/restart"
	HASensorStatusTopic  = "ha/sensor/status"
	HASensorArmingTopic  = "ha/sensor/arming"

	OPEN    = "OPEN"
	CLOSED  = "CLOSED"
	UNKNOWN = "UNKNOWN"

	HeartbeatTypeSensor = "sensor"
	HeartbeatTypeApp    = "app"

	DefaultOpenTimeoutMinutes = 5
	MinOpenTimeoutMinutes     = 1
	MaxOpenTimeoutMinutes     = 60

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

type SensorConfig struct {
	Source             string `json:"source"`
	OpenTimeoutMinutes int32  `json:"openTimeoutMinutes"`
}

type SensorState struct {
	Sensors []SensorStatus    `json:"sensors"`
	Arming  map[string]string `json:"arming"`
}

type APIPayload struct {
	Source      string `json:"source"`
	Armed       string `json:"armed"` // TODO: use bool. Need to handle zero value
	OpenTimeout int    `json:"openTimeout,omitempty"`
}
