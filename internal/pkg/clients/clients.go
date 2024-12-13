package clients

import (
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/aws"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/crypto"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/datadog"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/postgres"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/redis"
)

type ServerClients struct {
	Redis       redis.Client
	Postgres    postgres.Client
	Mosquitto   mqtt.MqttClient
	MosquittoV2 mqtt.MqttClient
	AWS         aws.Client
	DDClient    datadog.Client
	CryptoUtil  crypto.Util
}
