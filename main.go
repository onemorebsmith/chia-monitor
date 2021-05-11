package main

import (
	"log"
	"time"
)

var fastRate = 15 * time.Second
var slowRate = 1 * time.Minute

func main() {
	cfg, err := parseConfig("config.yaml")
	if err != nil {
		log.Println(err)
	}

	go startMemMonitor()
	go startDriveMonitoring(&cfg)
	go startProcessMonitor()
	startRecording()

	for {
		time.Sleep(5 * time.Minute)
	}
}
