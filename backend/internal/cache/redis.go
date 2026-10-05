package cache

import "github.com/redis/go-redis/v9"

var Redis *redis.Client

func NewRedisClient(add string) *redis.Client {
	Redis = redis.NewClient(&redis.Options{
		Addr:     add,
		Password: "",
		DB:       0,
	})

	return Redis
}
