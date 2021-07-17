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
var driveRegex = regexp.MustCompile(`/dev/(\D{3})\d+`)

func monitorDrives(dev mountMapping) {
	m := driveRegex.FindStringSubmatch(dev.mount)
	if len(m) < 2 {
		log.Printf("Failed to start monitoring on drive %s, format unexpected", dev.mount)
		return
	}

	fname := fmt.Sprintf("/sys/block/%s/stat", m[1])
	b, err := os.ReadFile(fname)
	if err != nil {
		log.Printf("Failed to start monitoring on drive %s: %v", dev.mount, err)
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

type DfOutput struct {
	devMount        string
	totalBlocks     int64
	usedBlocks      int64
	availableBlocks int64
	usePct          float64
	mount           string
}

var dfColRegex = regexp.MustCompile(`\s+`)

func df(path string, flags ...string) (*DfOutput, error) {
	// todo: df utility

	args := flags
	args = append(args, path)

	o, err := exec.Command("/bin/df", args...).Output()
	if err != nil {
		log.Printf("[DriveMonitor] Error calling 'df -h' for path '%s': %v", err, path)
	}

	rows := strings.Split(string(o), "\n")
	// first row is header, second is data
	if len(rows) < 1 {
		return nil, fmt.Errorf("unexpected output from df")
	}

	// clean up the results a bit
	rows[1] = dfColRegex.ReplaceAllString(rows[1], " ")
	cols := strings.Split(rows[1], " ")
	if len(cols) < 6 {
		return nil, fmt.Errorf("unexpected output from df")
	}

	out := &DfOutput{}
	out.devMount = cols[0]
	out.totalBlocks, err = strconv.ParseInt(cols[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unexpected output from df: %v", err)
	}

	out.usedBlocks, err = strconv.ParseInt(cols[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unexpected output from df: %v", err)
	}

	out.availableBlocks, err = strconv.ParseInt(cols[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unexpected output from df: %v", err)
	}

	out.usePct = float64(out.availableBlocks) / float64(out.totalBlocks)

	out.mount = cols[5]
	return out, nil
}

func pathToDevice(path string) (string, error) {
	dfStats, err := df(path)
	if err != nil {
		return "", err
	}

	log.Printf("Mapped '%s' => '%s'", path, dfStats.devMount)

	return dfStats.devMount, nil
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
