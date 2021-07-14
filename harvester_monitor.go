package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	farmedChia = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_chia_farmed",
		Help: "Total amount of chia farmed via 'chia farm summary'",
	})

	netspaceEstimate = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "netspace_estimate_pib",
		Help: "Netspace estimate via 'chia farm summary'",
	})

	farmedRegex   = regexp.MustCompile(`Block rewards: (\d+)`)
	netspaceRegex = regexp.MustCompile(`Estimated network space: (\d+)`)
)

func startFarmMonitor(chiaPath string) {
	for {

		chiaProc := exec.Command("sh")
		chiaProc.Dir = chiaPath

		buffer := bytes.Buffer{}
		summaryCommand := "chia farm summary"
		buffer.Write([]byte(fmt.Sprintf("cd %s;. ./activate;chia init;%s", chiaPath, summaryCommand)))
		chiaProc.Stdin = &buffer

		res, err := chiaProc.Output()
		if err != nil {
			log.Printf("[Harvester] error running 'chia farm summary': %v", err)
			continue
		}

		s := string(res)
		if matches, found := checkRegex(s, farmedRegex); found {
			f, err := strconv.ParseFloat(matches[0], 64)
			if err != nil {
				log.Printf("[Harvester] error parsing output from 'chia farm summary': %v", err)
			}
			farmedChia.Set(f)
		}

		if matches, found := checkRegex(s, netspaceRegex); found {
			f, err := strconv.ParseFloat(matches[0], 64)
			if err != nil {
				log.Printf("[Harvester] error parsing output from 'chia farm summary': %v", err)
			}
			netspaceEstimate.Set(f)
		}

		time.Sleep(1 * time.Minute)
	}
}
