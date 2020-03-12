package parser

import (
	"RobotChecker/logger"
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"os"
)

const prefix = "robot_checker_"

var redisAddr = os.Getenv("REDIS_ADDR")

var connection *redis.Conn

/**
Создаем соединение redis
*/
func init() {
	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		logger.Logger("Ошибка при соединении с redis: " + err.Error())
		panic("Не удалось установить соединение с redis")
	}
	connection = &conn
}

/**
Записываем данные в redis
*/
func RedisSet(key string, data *report) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Logger("Ошибка redis в методе RedisSet: " + err.Error())
	}

	conn := *connection
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
	conn := *connection
	data, err := redis.Bytes(conn.Do("HGET", prefix+key, "data"))
	if err != nil {
		logger.Logger("Ошибка redis в методе hget: " + err.Error())
		return &report{}
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		logger.Logger("Не удалось преобразовать данные из json: " + err.Error())
		return &report{}
	}

	return &result
}

func RedisGetBool(key string) bool {
	conn := *connection
	result, err := redis.Bool(conn.Do("GET", prefix+key))
	if err != nil && err.Error() != "nil returned" {
		logger.Logger("Ошибка redis в методе getBool: " + err.Error())
	}

	return result
}
