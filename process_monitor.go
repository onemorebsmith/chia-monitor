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

type ProcessMonitor struct {
	stateLock     *sync.Mutex
	plotterStates PlotterStates
}

type PlotterStates map[int]*PlotterState

type logEntry struct {
	pid  int
	msg  string
	live bool
}

var procChannel = make(chan logEntry)

func StartProcessMonitor() *ProcessMonitor {
	monitor := &ProcessMonitor{
		stateLock:     &sync.Mutex{},
		plotterStates: PlotterStates{},
	}

	go monitor.startProcessMonitor()

	return monitor
}

func (p *ProcessMonitor) monitorProcess(pid int) {
	p.stateLock.Lock()
	if _, found := p.plotterStates[pid]; !found {
		// process entry doesn't exist already, create it and start monitoring
		ps := &PlotterState{}
		ps.Pid = pid
		ps.State = map[string]string{
			"phase": "init",
			"table": "0",
		}
		ps.lastSeen = time.Now()
		p.plotterStates[pid] = ps
		p.stateLock.Unlock()

		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Println(err)
			return
		}

		go func(pid int) {
			path := fmt.Sprintf("/proc/%d/fd/1", proc.Pid)
			log.Printf("[Monitor] Opening '%s' to for monitoring", path)
			fd, err := os.Open(fmt.Sprintf("/proc/%d/fd/1", proc.Pid))
			if err != nil {
				log.Println(err)
				return
			}

			retries := 0
			r := bufio.NewReader(fd)
			live := false
			for {
				for {
					s, err := r.ReadString('\n')
					if len(s) == 0 && err == io.EOF {
						break
					}
					if err != nil {
						log.Printf("Monitor] Error reading from '%d': %+v", pid, err)
						if retries > 5 {
							log.Printf("[Monitor] Failed to read from pid %d, %+v", pid, err)
							p.stateLock.Lock()
							delete(p.plotterStates, pid)
							p.stateLock.Unlock()
							return
						}
						// try again in a bit
						time.Sleep(5 * time.Second)

						// retry opening & reset the buffer
						fd.Close()
						fd, err = os.Open(path)
						if err != nil {
							log.Println(err)
						}
						r = bufio.NewReader(fd)

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
		p.stateLock.Unlock()
	}
}

func (p *ProcessMonitor) startProcessMonitor() {
	go func() { // monitors plotter states
		for {
			s := <-procChannel

			p.stateLock.Lock()
			ps, found := p.plotterStates[s.pid]
			p.stateLock.Unlock()
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
			if err.Error() != "exit status 1" { // this means no processes
				log.Printf("[Monitor] Error fetching processes: %v", err)
			}
			goto wait
		}

		for _, s := range strings.Split(string(o), "\n") {
			pid, _ := strconv.Atoi(s)
			if pid > 0 {
				p.monitorProcess(pid)
			}
		}

		p.stateLock.Lock()
		for v, s := range p.plotterStates {
			if time.Since(s.lastSeen) > time.Duration(30*time.Minute) {
				log.Printf("[Monitor] Stopping monitor on pid %d due to inactivity", v)
				delete(p.plotterStates, v)
			}
		}
		p.stateLock.Unlock()
	wait:
		time.Sleep(30 * time.Second)
	}
}
