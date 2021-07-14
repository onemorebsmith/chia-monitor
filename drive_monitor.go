package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

	driveWrites = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "drive_writes",
		Help: "drive writes per second, mb",
	}, []string{
		"device",
		"path",
	})

	driveReads = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "drive_reads",
		Help: "drive reads per second, mb",
	}, []string{
		"device",
		"path",
	})
)

// https://www.kernel.org/doc/html/latest/block/stat.html
type DriveStats struct {
	readIOs      int64 // requests
	readMerges   int64 // requests
	readSectors  int64 // sectors
	readTicks    int64 // ms
	writeIOs     int64 // requests
	writeMerges  int64 //requests
	writeSectors int64 //sectors
	writeTicks   int64 // ms
	inFlight     int64 // requests
	ioTicks      int64 // ms
	waitQueue    int64 // ms
	//discardIOs    int64 // requests
	//discardMerges int64 // requests
	//discardTicks  int64 // ms
	//flushIOs      int64 // requests
	//flushTicks    int64 // ms
}

type mountMapping struct {
	mount string
	path  string
}

var mountStats = map[string]*DriveStats{}
var numberRegex = regexp.MustCompile(`\d+`)

func monitorDrives(dev mountMapping) {
	fname := fmt.Sprintf("/sys/block/%s/stat", dev.mount)
	b, err := os.ReadFile(fname)
	if err != nil {
		return
	}

	vals := numberRegex.FindAllString(string(b), -1)
	if len(vals) < 11 {
		return
	}

	stats := &DriveStats{}
	stats.readIOs, _ = strconv.ParseInt(vals[1], 10, 64)
	stats.readMerges, _ = strconv.ParseInt(vals[2], 10, 64)
	stats.readSectors, _ = strconv.ParseInt(vals[3], 10, 64)
	stats.readTicks, _ = strconv.ParseInt(vals[4], 10, 64)
	stats.writeIOs, _ = strconv.ParseInt(vals[5], 10, 64)
	stats.writeMerges, _ = strconv.ParseInt(vals[6], 10, 64)	
	stats.writeSectors, _ = strconv.ParseInt(vals[7], 10, 64)
	stats.writeTicks, _ = strconv.ParseInt(vals[8], 10, 64)
	stats.inFlight, _ = strconv.ParseInt(vals[9], 10, 64)
	stats.ioTicks, _ = strconv.ParseInt(vals[10], 10, 64)
	stats.waitQueue, _ = strconv.ParseInt(vals[11], 10, 64)

	if previous, exists := mountStats[dev.path]; exists {
		//reads/writes are in UNIX 512-byte sectors
		writesBytes := (stats.writeSectors - previous.writeSectors) * 512
		readsBytes := (stats.readSectors - previous.readSectors) * 512

		// prevent rollover issues when the linux counters rollover
		if writesBytes < 0 {
			writesBytes = 0
		}

		if readsBytes < 0 {
			readsBytes = 0
		}

		driveWrites.WithLabelValues(dev.mount, dev.path).Add(float64(writesBytes))
		driveReads.WithLabelValues(dev.mount, dev.path).Add(float64(readsBytes))
	}

	mountStats[dev.path] = stats
}

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

var driveRegex = regexp.MustCompile(`/dev/(\D{3})\d+`)

func pathToDevice(path string) (string, error) {
	o, err := exec.Command("/bin/df", "-h", path).Output()
	if err != nil {
		log.Printf("[DriveMonitor] Error calling 'df -h' for path '%s': %v", err, path)
	}

	rows := strings.Split(string(o), "\n")
	// first row is header, second is data
	if len(rows) < 1 {
		return "", fmt.Errorf("unexpected output from df")
	}

	mount := strings.Split(rows[1], " ")[0]

	m := driveRegex.FindStringSubmatch(mount)
	if len(m) < 2 {
		return "", fmt.Errorf("unexpected output from df")
	}

	log.Printf("Mapped '%s' => '%s'", path, m[1])

	return m[1], nil
}

func startDriveMonitoring(cfg DriveMonitorConfig) {
	// monitor temp paths
	go func(p []string) {

		var mounts []mountMapping
		for _, v := range p {
			mount, err := pathToDevice(v)
			if err != nil {
				log.Println(err)
				continue
			}

			mounts = append(mounts,
				mountMapping{mount: mount, path: v})
		}

		for {
			for _, v := range mounts {
				monitorDrives(v)
			}

			for _, v := range p {
				var stat unix.Statfs_t
				err := unix.Statfs(v, &stat)
				if err != nil {
					return
				}

				driveFree.WithLabelValues(v).Set(float64(stat.Bfree*uint64(stat.Bsize)) / 1024 / 1024)
				driveUsage.WithLabelValues(v).Set(float64((stat.Blocks-stat.Bfree)*uint64(stat.Bsize)) / 1024 / 1024)
			}

			time.Sleep(10 * time.Second)
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

			time.Sleep(slowRate)
		}
	}(validatePaths(append(cfg.FinalPaths, cfg.StagingPaths...)))

	// go func(p []string){

	// }()
}
