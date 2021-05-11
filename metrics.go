package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ramUsageGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chia_host_used_ram",
		Help: "The number of process currently plotting",
	})

	ramMaxGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chia_host_max_ram",
		Help: "The number of process currently plotting",
	})

	plotsFinished = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "plots_finished",
		Help: "The number of process currently plotting",
	})

	processVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "plotter_progress",
		Help: "The number of process currently plotting",
	}, []string{
		"pid",
	})
)

func startRecording() {
	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", nil)
	if err != nil {
		log.Println(err)
	}

}

func recordMetrics() {
	go func() {
		for {
			completions := float64(0)
			for _, v := range plotterStates {
				p, _ := strconv.Atoi(v.State["phase"])
				processVec.WithLabelValues(fmt.Sprintf("%d", v.Pid)).Set(float64(p))
				completions += float64(v.Completions)
			}

			plotsFinished.Set(completions)

			// processGauge.Set(float64(len(plotterStates)))

			// processGauge.W

			total := (float64)(meminfo["MemTotal"]) / 1024.0 / 1024.0
			free := (float64)(meminfo["MemAvailable"]) / 1024.0 / 1024.0

			ramUsageGauge.Set(total - free)
			ramMaxGauge.Set(total)

			//opsProcessed.Inc()
			time.Sleep(2 * time.Second)
		}
	}()
}
