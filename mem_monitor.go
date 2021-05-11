package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Meminfo map[string]uint64

var meminfo = Meminfo{}
var meminfoRegex = regexp.MustCompile(`(\w+):\s+(\d+)\s(\w+)`)

func parseMeminfo() (Meminfo, error) {
	o, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	ret := Meminfo{}
	r := bufio.NewReader(o)
	for {
		s, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}

		if meminfoRegex.Match([]byte(s)) {
			matches := meminfoRegex.FindStringSubmatch(s)
			if len(matches) > 2 {
				v, err := strconv.ParseUint(matches[2], 10, 64)
				if err != nil {
					return nil, err
				}
				ret[matches[1]] = v
			}
		}
	}

	return ret, nil
}

func startMemMonitor() {
	go func() {
		for {
			var err error
			meminfo, err = parseMeminfo()
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(fastRate)
		}
	}()
}
