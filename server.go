package main

import (
	"RobotChecker/handlers"
	"RobotChecker/logger"
	"net/http"
	"os"
)

/*
При запуске необходимо указывать
GO_PORT
NODE_PORT
REDIS_ADDR
Пример:
GO_PORT=9999 NODE_PORT=9090 REDIS_ADDR=localhost:6379 go run server.go
*/
func main() {
	server := &http.Server{Addr: ":" + os.Getenv("GO_PORT")}

	http.HandleFunc("/start-check", handlers.StartCheckHandler)

	// создаем файл с логами
	logger.Logger("Запускаем сервер go...")
	// запускаем вебсервер
	if err := server.ListenAndServe(); err != nil {
		logger.Logger("Ошибка при запуске вебсервера: " + err.Error())
	}
}
