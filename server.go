package main

import (
	"RobotChecker/handlers"
	"fmt"
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

	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err.Error())
	}
}
