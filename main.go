package main

import (
	"io"
	"log"
	"os"
	"time"
)

var fastRate = 15 * time.Second
var slowRate = 1 * time.Minute

func main() {
	logFile, err := os.OpenFile("monitor.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.Println("====== Startup Finished ======")
	cfg, err := parseConfig("config.yaml")
	if err != nil {
		log.Println(err)
	}

	go startMemMonitor()
	go startDriveMonitoring(&cfg)
	go startProcessMonitor()
	go startUhaul(cfg.UhaulConfig)
	startRecording()

	for {
		time.Sleep(5 * time.Minute)
	}
}
