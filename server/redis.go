package main

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

const (
	statePrefix     = "state/"
	heartbeatPrefix = "heartbeat/"
	armingPrefix    = "arming/"
)

var ctx = context.Background()

type redisClient struct {
	client redis.Client
}

func newRedisClient(redisURL string) (redisClient, error) {
	redisClient := redisClient{}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return redisClient, err
	}
	redisClient.client = *redis.NewClient(options)

	return redisClient, nil
}

func (r *redisClient) ReadAllState() (map[string]string, error) {
	state := make(map[string]string)
	keys := r.client.Keys(ctx, fmt.Sprintf("%s*", statePrefix)).Val()
	for _, k := range keys {
		val, err := r.client.Get(ctx, k).Result()
		if err != nil {
			return state, err
		}
		state[k] = val
	}
	return state, nil
}

func (r *redisClient) ReadState(key string) (string, error) {
	val, err := r.client.Get(ctx, fmt.Sprintf("%s%s", statePrefix, key)).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}

func (r *redisClient) WriteState(key string, value string) error {
	d := r.client.Set(ctx, fmt.Sprintf("%s%s", statePrefix, key), value, 0)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *redisClient) WriteHeartbeat(key string, value string) error {
	d := r.client.Set(ctx, fmt.Sprintf("%s%s", heartbeatPrefix, key), value, 0)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *redisClient) WriteArming(key string, value string) error {
	d := r.client.Set(ctx, fmt.Sprintf("%s%s", armingPrefix, key), value, 0)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *redisClient) ReadArming(key string) (string, error) {
	val, err := r.client.Get(ctx, fmt.Sprintf("%s%s", armingPrefix, key)).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}
