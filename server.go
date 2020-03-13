package main

import (
	"RobotChecker/configs"
	"RobotChecker/handlers"
	"RobotChecker/logger"
	"net/http"
)

func main() {
	server := &http.Server{Addr: ":" + configs.GoPort()}

	http.HandleFunc("/start-check", handlers.StartCheckHandler)

	// создаем файл с логами
	logger.Logger("Запускаем сервер go...")
	// запускаем вебсервер
	if err := server.ListenAndServe(); err != nil {
		logger.Logger("Ошибка при запуске вебсервера: " + err.Error())
	}
}
