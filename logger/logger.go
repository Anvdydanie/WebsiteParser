package logger

import (
	"log"
	"os"
	"time"
)

func Logger(msg string) {
	currentTime := time.Now()
	dateForLog := currentTime.Format("2006_01_02")

	file, err := os.OpenFile("./go-parser/go_logs_"+dateForLog+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	log.SetOutput(file)
	log.Print(msg)
}
