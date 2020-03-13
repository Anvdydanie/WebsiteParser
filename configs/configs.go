package configs

import (
	"github.com/joho/godotenv"
	"os"
)

func init() {
	err := godotenv.Load("./../.env")
	if err != nil {
		panic("Ошибка при загрузке env файла. " + err.Error())
	}
}

func GoPort() string {
	result, exists := os.LookupEnv("PARSER_PORT")
	if exists {
		return result
	} else {
		return "9999"
	}
}

func RedisAddr() string {
	redisHost, existsHost := os.LookupEnv("REDIS_HOST")
	redisPort, existsPort := os.LookupEnv("REDIS_PORT")
	if existsHost && existsPort {
		return redisHost + ":" + redisPort
	} else {
		return "localhost:6379"
	}
}

func NodeAddr() string {
	port, exists := os.LookupEnv("PORT_BACKEND")
	if exists {
		return "http://localhost:" + port
	} else {
		return "http://localhost:9090"
	}
}

func LogsPath() string {
	result, exists := os.LookupEnv("PARSER_LOGS_PATH")
	if exists {
		return result
	} else {
		return "./go-parser/"
	}
}
