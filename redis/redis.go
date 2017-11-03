package redis

import (
	redisClient "github.com/go-redis/redis"
)

type MyRedis struct {
	client  *redisClient.Client
	sub     *redisClient.PubSub
	channel string
}

func New(channel string) *MyRedis {
	client := redisClient.NewClient(&redisClient.Options{Addr: "localhost:6379"})

	redis := MyRedis{
		client:  client,
		sub:     client.Subscribe(channel),
		channel: channel,
	}
	return &redis
}

func (redis *MyRedis) Publish(message string) {
	redis.client.Publish(redis.channel, message)
}

func (redis *MyRedis) ReceiveMessage() (*redisClient.Message, error) {
	return redis.sub.ReceiveMessage()
}
