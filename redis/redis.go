package redis

import (
	redisClient "github.com/go-redis/redis"
)

type MyRedis struct {
	client  *redisClient.Client
	channel string
}

func New(host string, channel string) *MyRedis {
	client := redisClient.NewClient(&redisClient.Options{Addr: host})

	redis := MyRedis{
		client:  client,
		channel: channel,
	}
	return &redis
}

func (redis *MyRedis) Publish(message string) {
	redis.client.LPush(redis.channel, message)
}

func (redis *MyRedis) ReceiveMessage() (string, error) {
	return redis.client.RPop(redis.channel).Result()
}
