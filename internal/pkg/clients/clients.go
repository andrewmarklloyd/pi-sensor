package clients

import (
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/aws"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/datadog"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/postgres"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/redis"
)

type ServerClients struct {
	Redis    redis.Client
	Postgres postgres.Client
	Mqtt     mqtt.MqttClient
	AWS      aws.Client
	DDClient datadog.Client
}
