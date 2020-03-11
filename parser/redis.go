package parser

import (
	"RobotChecker/logger"
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
		logger.Logger("Ошибка redis в методе dial: " + err.Error())
		return
	}
	defer conn.Close()
	jsonData, _ := json.Marshal(data)

	_, err = conn.Do("HMSET", prefix+key, "data", jsonData)
	if err != nil {
		logger.Logger("Ошибка redis в методе hmset: " + err.Error())
	}
}

/**
Получаем данные из redis
*/
func RedisGet(key string) *report {
	var result report

	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		logger.Logger("Ошибка redis в методе dial: " + err.Error())
		return &report{}
	}
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("HGET", prefix+key, "data"))
	_ = json.Unmarshal(data, &result)
	if err != nil {
		logger.Logger("Ошибка redis в методе hget: " + err.Error())
		return &report{}
	}

	return &result
}

func RedisGetBool(key string) bool {
	var result = false

	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		logger.Logger("Ошибка redis в методе dial: " + err.Error())
		return false
	}
	defer conn.Close()

	result, err = redis.Bool(conn.Do("GET", prefix+key))
	if err != nil {
		logger.Logger("Ошибка redis в методе getBool: " + err.Error())
		return result
	}

	return result
}
