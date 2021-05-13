package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PhaseTime struct {
	Phase    string
	Run      int
	Duration time.Duration
}

type PlotterState struct {
	State       map[string]string
	Pid         int
	Completions int
	PhaseTimes  []PhaseTime

	Phase       string
	Table       string
	Bucket      string
	PlotSz      int
	BucketCount int
	MaxRamMb    int
	MaxThread   int
	Duration    time.Duration
}

var processors = map[string][]*regexp.Regexp{
	"plotSize":   {regexp.MustCompile(`Plot size is: (\d+)`)},
	"maxRam":     {regexp.MustCompile(`Buffer size is: (\d+)MiB`)},
	"bucketSize": {regexp.MustCompile(`Using (\d+) buckets`)},
	"phase":      {regexp.MustCompile(`.*Starting phase (\d)/*.`)},
	"table": {
		regexp.MustCompile(`Computing table (\d+)`),
		regexp.MustCompile(`Compressing tables (\d+)`),
	},
	"bucket":     {regexp.MustCompile(`.*Bucket (\d+)`)},
	"temp_drive": {regexp.MustCompile(`Starting plotting progress into temporary dirs: (.*) and`)},
	"plot_id":    {regexp.MustCompile(`ID: (\w+)`)},
}
var debugPid = 336480

var runCounter = regexp.MustCompile(`Total time = (\d+).* (\w+ \w+ \d{1,2} \d{2}:\d{2}:\d{2} \d{4})`)
var phaseTime = regexp.MustCompile(`Time for phase (\d) = (\d+)`)
var copyTime = regexp.MustCompile(`Copy time = (\d+)`)
var compressPhase = regexp.MustCompile(`Compressing tables (\d+)`)

var phaseTimings = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "phase_timings",
	Help: "The number of process currently plotting",
}, []string{
	"pid",
	"phase",
	"id",
})

// value is the timestamp when finished
var completionMarker = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "completed_plots",
	Help: "Number of plots completed by the given pid",
}, []string{
	"pid",
	"drive",
	"id",
})

var plotterState = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "plotter_state",
	Help: "Full plotter state breakdown",
}, []string{
	"pid",
	"phase",
	"table",
})

var plotterProgress = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "plotter_progress",
	Help: "Plotter progress %",
}, []string{
	"pid",
})

func checkRegexes(s string, reg []*regexp.Regexp) ([]string, bool) {
	for _, r := range reg {
		if v, ok := checkRegex(s, r); ok {
			return v, ok
		}
	}

	return nil, false
}

func checkRegex(s string, r *regexp.Regexp) ([]string, bool) {
	if r.Match([]byte(s)) {
		matches := r.FindStringSubmatch(s)
		if len(matches) > 1 {
			return matches[1:], true
		}
	}

	return nil, false
}

func updateProgress(ps *PlotterState) {
	pid := fmt.Sprintf("%d", ps.Pid)
	p := ps.State["phase"]
	t := ps.State["table"]
	b := ps.State["bucket"]
	bs := ps.State["bucketSize"]

	progress := float64(0)
	switch p {
	case "final":
		progress = 99
	case "copy":
		progress = 95
	default:
		pi, err := strconv.ParseFloat(p, 64)
		if err != nil {
			pi = 1
		}
		ti, err := strconv.ParseFloat(t, 64)
		if err != nil {
			ti = 1
		}
		bi, err := strconv.ParseFloat(b, 64)
		if err != nil {
			bi = 1
		}
		bsi, err := strconv.ParseFloat(bs, 64)
		if err != nil {
			bsi = 128
		}

		if progress < 0 || progress > 100 {
			log.Printf("[%d] Error determining progress %+v", ps.Pid, *ps)
			return
		}

		// p4, t7, b 32/32
		// bsi buckets = 1 table, 7 tables = 1 phase
		progress = ((pi - 1) * 30) + ((ti / 7) * 30) + (bi/bsi)*4.28571428571
	}

	if ps.Pid == debugPid {
		log.Printf("[%d] %f", ps.Pid, progress)
	}

	for _, pp := range statesNames {
		for _, tt := range tableNames {
			// clear previous metrics or they'll continue to send
			plotterState.DeleteLabelValues(pid, pp, tt)
		}
	}

	plotterState.WithLabelValues(pid, p, t).Set(progress)
	plotterProgress.WithLabelValues(pid).Set(progress)
}

var statesNames = []string{"1", "2", "3", "4", "copy", "final", "init"}
var tableNames = []string{"0", "1", "2", "3", "4", "5", "6", "7"}

func phaseChanged(ps *PlotterState, phase string, duration int) {
	ps.State["phase"] = phase

	plot_id := ps.State["plot_id"]

	tt := PhaseTime{
		Phase:    phase,
		Run:      ps.Completions,
		Duration: time.Second * time.Duration(duration),
	}

	// phase times
	ps.PhaseTimes = append(ps.PhaseTimes, tt)
	if plot_id == "" {
		plot_id = "unknown"
	}

	phaseTimings.WithLabelValues(fmt.Sprintf("%d", ps.Pid), ps.Phase, plot_id).Set(ps.Duration.Seconds())

	updateProgress(ps)
}

func (s *PlotterState) Update(entry *logEntry) {
	for k, r := range processors {
		if val, valid := checkRegexes(entry.msg, r); valid {
			if s.Pid == debugPid {
				log.Printf("[%d] %s = %s, [%s]", s.Pid, k, val[0], entry.msg)
			}
			switch k {
			case "phase": // phase we reset table and bucket
				s.State["table"] = "0"
				fallthrough
			case "table": // table we just reset bucket
				s.State["bucket"] = "0"
			default:
				// nothing
			}

			s.State[k] = val[0]
			updateProgress(s)
		}
	}

	if val, valid := checkRegex(entry.msg, phaseTime); valid {
		dur, _ := strconv.Atoi(val[1])
		phaseChanged(s, val[0], dur)
	}

	// if val, valid := checkRegex(entry.msg, compressPhase); valid {
	// 	s.State["table"] = val
	// 	phaseChanged(s, "compress", 0)
	// }

	if val, valid := checkRegex(entry.msg, copyTime); valid {
		dur, _ := strconv.Atoi(val[0])
		phaseChanged(s, "copy", dur)
	}

	if val, valid := checkRegex(entry.msg, runCounter); valid {
		if len(val) >= 2 {
			id := s.State["plot_id"]
			temp := s.State["temp_drive"]
			timestamp, _ := time.Parse(time.ANSIC, val[1])
			pid := fmt.Sprintf("%d", s.Pid)
			dur, _ := strconv.Atoi(val[0])
			phaseChanged(s, "final", dur)

			completionMarker.WithLabelValues(pid, temp, id).Set(float64(timestamp.Unix()))
			s.Completions++
		}
	}

	s.State["last"] = entry.msg
}
