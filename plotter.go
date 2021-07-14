package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var ownedPlotters map[string][]*os.Process
var lastLaunched map[string]time.Time

func startPlotter(cfg []*PlotterConfig, chiaPath string) {
	if len(cfg) == 0 {
		log.Println("[Plotter] No config specified, skipping init")
		return
	}

	cfgMap := map[string]PlotterConfig{}
	ownedPlotters = map[string][]*os.Process{}
	lastLaunched = map[string]time.Time{}

	for _, v := range cfg {
		cfgMap[v.TempPath] = *v
	}

	monitor(cfgMap, chiaPath)
	// for _, k := range cfg {

	// }
}

// type plotterState struct {
// 	workdir string

// 	plotters []*PlotterState
// }

var template = `chia plots create -n 1 -r {CORES} -k 32 -c {POOL_KEY}  -u {BUCKETS} -b {RAM} -t {TEMP_PATH} -d {FINAL_PATH} -x  2>&1 > {LOGFILE}.log &`

func startPlot(cfg PlotterConfig, chiaPath string) {
	log.Printf("[%s] Starting plot on %s => %s", cfg.Tag, cfg.TempPath, cfg.FinalPath)
	lastLaunched[cfg.Tag] = time.Now()
	chiaProc := exec.Command("sh")
	chiaProc.Dir = chiaPath

	cwd, _ := os.Getwd()

	buffer := bytes.Buffer{}
	plotTag := fmt.Sprintf("%s_%d", cfg.Tag, time.Now().UTC().Unix())

	plotterCommand := strings.ReplaceAll(template, "{CORES}", cfg.Cores)
	plotterCommand = strings.ReplaceAll(plotterCommand, "{BUCKETS}", cfg.Buckets)
	plotterCommand = strings.ReplaceAll(plotterCommand, "{RAM}", cfg.Ram)
	plotterCommand = strings.ReplaceAll(plotterCommand, "{TEMP_PATH}", fmt.Sprintf("%s/%s", cfg.TempPath, plotTag))
	plotterCommand = strings.ReplaceAll(plotterCommand, "{FINAL_PATH}", cfg.FinalPath)
	plotterCommand = strings.ReplaceAll(plotterCommand, "{LOGFILE}", fmt.Sprintf("%s/plotter_logs/%s", cwd, plotTag))
	plotterCommand = strings.ReplaceAll(plotterCommand, "{POOL_KEY}", cfg.PoolKey)
	log.Printf("[%s] Plot command: `%s`", cfg.Tag, plotterCommand)

	buffer.Write([]byte(fmt.Sprintf("cd %s;. ./activate;chia init;%s", chiaPath, plotterCommand)))
	chiaProc.Stdin = &buffer

	//chiaProc.Stdout = log.Writer()
	//chiaProc.Stderr = os.Stderr

	//chiaProc.Process.Release()

	err := chiaProc.Start()
	if err != nil {
		log.Printf("[%s] Error starting/running plotter: %+v", cfg.Tag, err)
		return
	}
	if chiaProc.Process != nil {
		//pid := chiaProc.Process.Pid
		chiaProc.Process.Release()
		ownedPlotters[cfg.Tag] = append(ownedPlotters[cfg.Tag], chiaProc.Process)
		//monitorProcess(pid)
	}

	// err := chiaProc.Run()
	// if err != nil {
	// 	log.Printf("Failed initializing chia client: %+v", err)
	// 	return
	// }
}

func monitor(cfgMap map[string]PlotterConfig, chiaPath string) {
	time.Sleep(time.Second * 60)

	start := time.Now()

	for {
		states := map[string][]*PlotterState{}

		pm := processMonitor

		pm.stateLock.Lock()
		for _, v := range pm.plotterStates {
			if drive, exists := v.State["temp_drive"]; exists {
				//drive := path.Dir(drive)
				drive = filepath.Clean(filepath.Join(drive, ".."))
				//path.Match(pattern string, name string)
				states[drive] = append(states[drive], v)
			}

		}

		pm.stateLock.Unlock()

		log.Println("==================================")

		for k, cfg := range cfgMap {
			byPhase := map[string][]*PlotterState{}
			plotters := states[k]
			active := 0
			log.Printf("[Plotter][%s] ------------------", cfg.Tag)
			if time.Since(start) < cfg.StartDelay {
				wait := cfg.StartDelay - time.Since(start)
				log.Printf("\tWill start potter in approx %f minutes due to start delay", (cfg.MinCooldown-wait).Seconds()/60)
				continue
			}
			for _, v := range plotters {
				phase := v.State["phase"]
				prog := v.State["progress"]
				if phase != "copy" {
					active++
					log.Printf("\t%d state: %s, progress: %s", v.Pid, phase, prog)
					byPhase[phase] = append(byPhase[phase], v)
				}
			}

			log.Printf("	[%d/%d] active plotters", len(plotters), cfg.StageConcurrency)

			if len(plotters) < cfg.StageConcurrency {
				if p1Plotters, exists := byPhase["1"]; exists {
					if len(p1Plotters) >= cfg.MaxPhase1 {
						log.Printf("\t%d plotters in phase 1, max %d, waiting to launch more", len(p1Plotters), cfg.MaxPhase1)
						continue
					}
					log.Printf("\t%d plotters in phase 1, max %d, starting a plotter", len(p1Plotters), cfg.MaxPhase1)
				}

				lastStarted := lastLaunched[cfg.Tag]
				elapsed := time.Since(lastStarted)
				if elapsed < cfg.MinCooldown {
					log.Printf("\tIn cooldown, will launch plotter in approx %f minutes", (cfg.MinCooldown-elapsed).Seconds()/60)
					continue
				}

				startPlot(cfg, chiaPath)
			}
		}

		time.Sleep(time.Second * 60 * 5)
	}
}
