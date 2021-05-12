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
	"time"
)

type PlotterStates map[int]*PlotterState

var plotterStates = PlotterStates{}

type logEntry struct {
	pid int
	msg string
}

var procChannel = make(chan logEntry)

func monitorProcess(pid int) {
	if _, found := plotterStates[pid]; !found {
		// process entry doesn't exist already, create it and start monitoring
		ps := &PlotterState{}
		ps.Pid = pid
		ps.State = map[string]string{
			"phase": "init",
		}
		plotterStates[pid] = ps

		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Println(err)
			return
		}
		_ = proc

		fd := fmt.Sprintf("/proc/%d/fd/1", pid)
		log.Printf("Opening '%s' to for monitoring", fd)
		r, err := os.Open(fmt.Sprintf("/proc/%d/fd/1", pid))
		if err != nil {
			log.Println(err)
			return
		}

		go func(pid int) {
			r := bufio.NewReader(r)
			// seek to the end of the buffer so we don't replay/dupe data
			// if the stdout is redirected to a file or something
			r.Discard(r.Size())
			for {
				for {
					s, err := r.ReadString('\n')
					if len(s) == 0 && err == io.EOF {
						break
					}
					if err != nil {
						log.Printf("Error reading from '%d': %+v", pid, err)
						return
					}

					procChannel <- logEntry{msg: s, pid: pid}
				}
				time.Sleep(5 * time.Second)
			}
		}(pid)
	}
}

func startProcessMonitor() {
	go func() { // monitors plotter states
		for {
			s := <-procChannel

			ps, found := plotterStates[s.pid]
			if !found {
				ps = &PlotterState{}
				ps.Pid = s.pid
				ps.State = map[string]string{}
				plotterStates[s.pid] = ps
			}

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

		time.Sleep(30 * time.Second)
	}
}
