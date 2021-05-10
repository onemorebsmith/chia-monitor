package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
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

var numberRegex = regexp.MustCompile(`\d+`)

func monitorDrives(dev string) {
	fname := fmt.Sprintf("/sys/block/%s/stat", dev)
	for {
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
		//stats.discardIOs, _ = strconv.ParseInt(vals[12], 10, 64)

		time.Sleep(5 * time.Second)
	}
}
