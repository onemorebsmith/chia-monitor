package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/encoding"
)

type Meminfo map[string]uint64
type PlotterStates map[int]*PlotterState

var meminfo = Meminfo{}
var plotterStates = PlotterStates{}

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

type logEntry struct {
	pid int
	msg string
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic", []byte(string(debug.Stack())), os.ModePerm)

			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}

		os.WriteFile("normal", []byte("test"), os.ModePerm)

		os.Exit(1)
	}()

	cfg, err := parseConfig("config.yaml")
	if err != nil {
		log.Println(err)
	}

	log.Println(cfg)

	monitorDrives(&cfg)

	//monitorDrives("sda")

	f, _ := os.OpenFile("stdout_redir", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer f.Close()
	redir := io.MultiWriter(os.Stdout, f)

	log.SetOutput(redir)

	log.Println(os.Getpid())
	//os.Stdout = f

	go startRecording()

	encoding.Register()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	go func() {
		for {
			var err error
			meminfo, err = parseMeminfo()
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	wg := sync.WaitGroup{}

	o, err := exec.Command("/usr/bin/pgrep", "-f", "chia plots create").Output()
	if err != nil {
		log.Fatal(err)
	}

	var procIds []int
	for _, s := range strings.Split(string(o), "\n") {
		pid, _ := strconv.Atoi(s)
		if pid > 0 {
			procIds = append(procIds, pid)
		}
	}

	output := make(chan logEntry)
	for _, pid := range procIds {
		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Println(err)
			continue
		}
		_ = proc

		fd := fmt.Sprintf("/proc/%d/fd/1", pid)
		log.Printf("Opening '%s' to for monitoring", fd)
		r, err := os.Open(fmt.Sprintf("/proc/%d/fd/1", pid))
		if err != nil {
			log.Println(err)
			continue
		}

		wg.Add(1)
		go func(pid int) {
			r := bufio.NewReader(r)
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

					output <- logEntry{msg: s, pid: pid}
				}
				time.Sleep(5 * time.Second)
			}
		}(pid)
	}

	go func() {
		for {
			s := <-output

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

	wg.Wait()
}
