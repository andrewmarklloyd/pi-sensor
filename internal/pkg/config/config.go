package config

type ServerConfig struct {
	AppName            string
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
	S3Config           S3Config
	Version            string
	AllowedAPIKeys     []string
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

type S3Config struct {
	AccessKeyID       string
	SecretAccessKey   string
	Region            string
	URL               string
	Bucket            string
	MaxRetentionRows  int
	RetentionEnabled  bool
	FullBackupEnabled bool
}
