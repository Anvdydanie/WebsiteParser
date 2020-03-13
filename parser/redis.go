package parser

import (
	"RobotChecker/configs"
	"RobotChecker/logger"
	"encoding/json"
	"github.com/gomodule/redigo/redis"
)

const prefix = "robot_checker_"

var connection *redis.Conn

/**
Создаем соединение redis
*/
func init() {
	conn, err := redis.Dial("tcp", configs.RedisAddr())
	if err != nil {
		logger.Logger("Ошибка при соединении с redis: " + err.Error())
		panic("Не удалось установить соединение с redis. " + err.Error())
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
	result, _ := redis.Bool(conn.Do("GET", prefix+key))

	return result
}
