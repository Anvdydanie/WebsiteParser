package main

import (
	"RobotChecker/handlers"
	"fmt"
	"net/http"
)

func main() {
	server := &http.Server{Addr: ":9999"}

	http.HandleFunc("/start-check", handlers.StartCheckHandler)

	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err.Error())
	}
}
