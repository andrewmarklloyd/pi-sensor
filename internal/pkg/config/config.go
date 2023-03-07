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
	S3Config           S3Config
	Version            string
	AllowedAPIKeys     []string
	DatadogConfig      DatadogConfig
	WebPushConfig      WebPushConfig
}

type GoogleConfig struct {
	AuthorizedUsers string
	ClientId        string
	ClientSecret    string
	RedirectURL     string
	SessionSecret   string
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

type DatadogConfig struct {
	APIKey         string
	APPKey         string
	TokensMetadata []TokenMetadata
}

type TokenMetadata struct {
	Name       string
	Owner      string
	Expiration string
}

type WebPushConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
}
