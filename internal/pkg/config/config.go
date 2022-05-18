package config

type ServerConfig struct {
	MqttBrokerURL      string
	MqttServerUser     string
	MqttServerPassword string
	Topic              string
	RedisURL           string
	RedisTLSURL        string
	PostgresURL        string
	Port               string
	MockMode           bool
	GoogleConfig       GoogleConfig
	TwilioConfig       TwilioConfig
	Version            string
}

type GoogleConfig struct {
	AuthorizedUsers string
	ClientId        string
	ClientSecret    string
	RedirectURL     string
	SessionSecret   string
}

type TwilioConfig struct {
	AccountSID string
	AuthToken  string
	To         string
	From       string
}
