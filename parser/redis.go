package parser

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"os"
)

const prefix = "robot_checker_"

var redisAddr = os.Getenv("REDIS_ADDR")

/**
Записываем данные в redis
*/
func RedisSet(key string, data *report) {
	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		return
	}
	defer conn.Close()
	jsonData, _ := json.Marshal(data)

	_, err = conn.Do("HMSET", prefix+key, "data", jsonData)
}

/**
Получаем данные из redis
*/
func RedisGet(key string) *report {
	var result report

	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		return &report{}
	}
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("HGET", prefix+key, "data"))
	_ = json.Unmarshal(data, &result)
	if err != nil {
		return &report{}
	}

	return &result
}

func RedisGetBool(key string) bool {
	var result = false

	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		return false
	}
	defer conn.Close()

	result, err = redis.Bool(conn.Do("GET", prefix+key))
	if err != nil {
		return result
	}

	return result
}
