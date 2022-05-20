package clients

import (
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/aws"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/notification"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/postgres"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/redis"
)

type ServerClients struct {
	Redis     redis.Client
	Postgres  postgres.Client
	Messenger notification.Messenger
	Mqtt      mqtt.MqttClient
	AWS       aws.Client
}
