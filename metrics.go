package main

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ramUsageGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chia_host_used_ram",
		Help: "The current ram usage for the host",
	})

	ramMaxGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chia_host_max_ram",
		Help: "The current max ram for the host",
	})

	swapUsageGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chia_host_swap_usage",
		Help: "The current swap usage for the host",
	})

	// phaseTime = promauto.NewCounterVec(prometheus.CounterOpts{
	// 	Name: "plotter_state",
	// 	Help: "Time spent in each phase",
	// }, []string{
	// 	"pid",
	// 	"phase",
	// })
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
		// give the program some time to settle/process before we start sending metrics
		time.Sleep(30 * time.Second)
		for {
			// completions := float64(0)
			// for _, v := range plotterStates {

			// 	pi, _ := strconv.ParseFloat(p, 64)
			// 	ti, _ := strconv.ParseFloat(t, 64)
			// 	bi, _ := strconv.ParseFloat(b, 64)
			// 	bsi, _ := strconv.ParseFloat(bs, 64)

			// 	// p4, t7, b 32/32
			// 	// (3 * 25) + (7/7) * 20 + (32/32) * 5 = 100
			// 	progress := ((pi - 1) * 20) + ((ti / 7) * 20) + (bi/bsi)*5

			// 	pid := fmt.Sprintf("%d", v.Pid)
			// 	//processVec.WithLabelValues(pid).Set(float64(p))
			// 	completions += float64(v.Completions)

			// 	//phaseTime.WithLabelValues(pid, p).Inc()
			// }

			// processGauge.Set(float64(len(plotterStates)))

			// processGauge.W

			total := (float64)(meminfo["MemTotal"]) / 1024.0 / 1024.0
			free := (float64)(meminfo["MemAvailable"]) / 1024.0 / 1024.0
			swapTotal := (float64)(meminfo["SwapTotal"]) / 1024.0 / 1024.0
			swapFree := (float64)(meminfo["SwapFree"]) / 1024.0 / 1024.0

			ramUsageGauge.Set(total - free)
			ramMaxGauge.Set(total)
			swapUsageGauge.Set(swapTotal - swapFree)

			//opsProcessed.Inc()
			time.Sleep(15 * time.Second)
		}
	}()
}
