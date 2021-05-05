package main

import (
	"context"
	"os"

	"github.com/go-redis/redis/v8"
)

// Client allows reading and writing state of the sensors
type Client struct {
	redisClient redis.Client
}

var ctx = context.Background()

// Init configures the state client
func (s *Client) Init() error {
	options, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		return err
	}
	s.redisClient = *redis.NewClient(options)

	return nil
}

// ReadAllState gets all keys and values in state
func (s *Client) ReadAllState() (map[string]string, error) {
	state := make(map[string]string)
	keys := s.redisClient.Keys(ctx, "*").Val()
	for _, k := range keys {
		val, err := s.redisClient.Get(ctx, k).Result()
		if err != nil {
			return state, err
		}
		state[k] = val
	}
	return state, nil
}

// ReadState returns the sensor state
func (s *Client) ReadState(key string) (string, error) {
	val, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}

// WriteState sets the sensor state
func (s *Client) WriteState(key string, value string) error {
	err := s.redisClient.Set(ctx, key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}
