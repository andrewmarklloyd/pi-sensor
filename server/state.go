package main

import (
	"context"

	"github.com/go-redis/redis/v8"
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
	keys := r.client.Keys(ctx, "*").Val()
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
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}

func (r *redisClient) WriteState(key string, value string) error {
	err := r.client.Set(ctx, key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}
