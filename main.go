package main

import (
	"io"
	"log"
	"os"
	"time"
)

var fastRate = 15 * time.Second
var slowRate = 1 * time.Minute

var processMonitor *ProcessMonitor

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
	processMonitor = StartProcessMonitor()

	go startPlotter(cfg.PlotterConfig, cfg.ChiaPath)
	if cfg.UhaulEnabled {
		go startUhaul(cfg.UhaulConfig)
	} else {
		log.Println("[WARN] UHaul disabled in cfg")
	}
	if cfg.FarmMonitorEnabled {
		go startFarmMonitor(cfg.ChiaPath)
	} else {
		log.Println("[WARN] Farm monitor disabled in cfg")
	}

	startRecording()

	for {
		time.Sleep(5 * time.Minute)
	}
}
