package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PlotterStates map[int]*PlotterState

var stateLock = sync.Mutex{}
var plotterStates = PlotterStates{}

type logEntry struct {
	pid  int
	msg  string
	live bool
}

var procChannel = make(chan logEntry)

func monitorProcess(pid int) {
	stateLock.Lock()
	if _, found := plotterStates[pid]; !found {
		// process entry doesn't exist already, create it and start monitoring
		ps := &PlotterState{}
		ps.Pid = pid
		ps.State = map[string]string{
			"phase": "init",
			"table": "0",
		}
		plotterStates[pid] = ps
		stateLock.Unlock()

		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Println(err)
			return
		}
		_ = proc

		fd := fmt.Sprintf("/proc/%d/fd/1", pid)
		log.Printf("Opening '%s' to for monitoring", fd)
		ps.fd1, err = os.Open(fmt.Sprintf("/proc/%d/fd/1", pid))
		if err != nil {
			log.Println(err)
			return
		}

		go func(pid int) {
			retries := 0
			r := bufio.NewReader(ps.fd1)
			live := false
			for {
				for {
					s, err := r.ReadString('\n')
					if len(s) == 0 && err == io.EOF {
						break
					}
					if err != nil {
						log.Printf("Error reading from '%d': %+v", pid, err)
						if retries > 5 {
							return
						}
						// try again in a bit
						time.Sleep(30 * time.Second)
						retries++
						continue
					}

					procChannel <- logEntry{msg: s, pid: pid, live: live}
				}
				live = true // we're at the latest data, start sending events
				time.Sleep(5 * time.Second)
			}
		}(pid)
	} else {
		stateLock.Unlock()
	}
}

func startProcessMonitor() {
	go func() { // monitors plotter states
		for {
			s := <-procChannel

			stateLock.Lock()
			ps, found := plotterStates[s.pid]
			stateLock.Unlock()
			if !found {
				continue
			}

			ps.lastSeen = time.Now()
			ps.Update(&s)
		}
	}()

	for {
		o, err := exec.Command("/usr/bin/pgrep", "-f", "chia plots create").Output()
		if err != nil {
			log.Fatal(err)
		}

		for _, s := range strings.Split(string(o), "\n") {
			pid, _ := strconv.Atoi(s)
			if pid > 0 {
				monitorProcess(pid)
			}
		}

		stateLock.Lock()
		for v, s := range plotterStates {
			if time.Since(s.lastSeen) > time.Duration(60*time.Minute) {
				log.Printf("[Monitor] Stopping monitor on pid %d due to activity", v)
				// cleanup
				s.fd1.Close()
				delete(plotterStates, v)
			}
		}
		stateLock.Unlock()

		time.Sleep(30 * time.Second)
	}
}
