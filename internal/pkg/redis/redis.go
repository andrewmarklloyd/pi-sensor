package redis

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/go-redis/redis/v8"
)

const (
	statePrefix     = "state/"
	heartbeatPrefix = "heartbeat/"
	armingPrefix    = "arming/"
)

type Client struct {
	client redis.Client
}

func NewRedisClient(redisURL string, tlsEnabled bool) (Client, error) {
	redisClient := Client{}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return redisClient, err
	}
	if tlsEnabled {
		// todo: do we actually need this?
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	redisClient.client = *redis.NewClient(options)

	return redisClient, nil
}

func (c *Client) GetAllSensors(ctx context.Context) ([]string, error) {
	keys := c.client.Keys(ctx, fmt.Sprintf("%s*", statePrefix)).Val()
	var sensors []string
	for _, v := range keys {
		sensors = append(sensors, strings.ReplaceAll(v, statePrefix, ""))
	}
	return sensors, nil
}

func (c *Client) ReadAllState(ctx context.Context) ([]config.SensorStatus, error) {
	sensorList := []config.SensorStatus{}
	keys := c.client.Keys(ctx, fmt.Sprintf("%s*", statePrefix)).Val()
	for _, k := range keys {
		val, err := c.client.Get(ctx, k).Result()
		if err != nil {
			return []config.SensorStatus{}, err
		}
		status := config.SensorStatus{}
		err = json.Unmarshal([]byte(val), &status)
		if err != nil {
			return []config.SensorStatus{}, err
		}
		sensorList = append(sensorList, status)
	}

	return sensorList, nil
}

func (c *Client) ReadAllArming(ctx context.Context) (map[string]string, error) {
	armingState := make(map[string]string)
	keys := c.client.Keys(ctx, fmt.Sprintf("%s*", armingPrefix)).Val()
	for _, k := range keys {
		val, err := c.client.Get(ctx, k).Result()
		if err != nil {
			return armingState, err
		}

		armingState[strings.Replace(k, armingPrefix, "", -1)] = val
	}

	return armingState, nil
}

func (c *Client) ReadState(key string, ctx context.Context) (string, error) {
	val, err := c.client.Get(ctx, fmt.Sprintf("%s%s", statePrefix, key)).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}

func (c *Client) WriteState(key, value string, ctx context.Context) error {
	d := c.client.Set(ctx, fmt.Sprintf("%s%s", statePrefix, key), value, 0)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) WriteHeartbeat(key, value string, ctx context.Context) error {
	d := c.client.Set(ctx, fmt.Sprintf("%s%s", heartbeatPrefix, key), value, 0)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) WriteArming(key, value string, ctx context.Context) error {
	d := c.client.Set(ctx, fmt.Sprintf("%s%s", armingPrefix, key), value, 0)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ReadArming(key string, ctx context.Context) (string, error) {
	val, err := c.client.Get(ctx, fmt.Sprintf("%s%s", armingPrefix, key)).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}
