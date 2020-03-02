package main

import (
	"RobotChecker/handlers"
	"net/http"
)

func main() {
	http.HandleFunc("/start-check", handlers.StartCheckHandler)
	//http.HandleFunc("/stop-check", handlers.StopCheckHandler)

	http.ListenAndServe(":9999", nil)
}
