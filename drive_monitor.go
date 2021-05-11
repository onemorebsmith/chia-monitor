package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sys/unix"
)

var (
	driveUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "drive_used_mb",
		Help: "drive used space in mb",
	}, []string{
		"path",
	})

	driveFree = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "drive_free_mb",
		Help: "drive free space in mb",
	}, []string{
		"path",
	})

	plotCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "plot_count",
		Help: "number of plots in each final folder",
	}, []string{
		"path",
	})
)

func validatePaths(paths []string) []string {
	var validPaths []string
	for _, v := range paths {
		if _, err := os.Stat(v); os.IsNotExist(err) {
			log.Printf("Monitor path '%s' does not exist", v)
			// path does not exist
			continue
		}

		validPaths = append(validPaths, v)
	}
	return validPaths
}

func monitorDrives(cfg *MonitorConfig) {

	// monitor temp paths
	go func(p []string) {
		for {
			for _, v := range p {
				var stat unix.Statfs_t
				err := unix.Statfs(v, &stat)
				if err != nil {
					return
				}

				driveFree.WithLabelValues(v).Set(float64(stat.Bfree*uint64(stat.Bsize)) / 1024 / 1024)
				driveUsage.WithLabelValues(v).Set(float64((stat.Blocks-stat.Bfree)*uint64(stat.Bsize)) / 1024 / 1024)
			}

			time.Sleep(30 * time.Second)
		}
	}(validatePaths(cfg.TempPaths))

	go func(p []string) { // Monitor plot count
		for {
			for _, v := range p {
				count := 0
				files, _ := ioutil.ReadDir(v)
				for _, f := range files {
					if filepath.Ext(f.Name()) == ".plot" {
						count++
					}
				}
				plotCount.WithLabelValues(v).Set(float64(count))
			}

			time.Sleep(1 * time.Minute)
		}
	}(validatePaths(append(cfg.FinalPaths, cfg.StagingPaths...)))
}
