package main

import (
	"fmt"
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
	"plotSize":   []*regexp.Regexp{regexp.MustCompile(`Plot size is: (\d+)`)},
	"maxRam":     []*regexp.Regexp{regexp.MustCompile(`Buffer size is: (\d+)MiB`)},
	"bucketSize": []*regexp.Regexp{regexp.MustCompile(`Using (\d+) buckets`)},
	"phase":      []*regexp.Regexp{regexp.MustCompile(`.*Starting phase (\d)/*.`)},
	"table": []*regexp.Regexp{
		regexp.MustCompile(`Computing table (\d+)`),
		regexp.MustCompile(`Compressing tables (\d+)`),
	},
	"bucket":     []*regexp.Regexp{regexp.MustCompile(`.*Bucket (\d+)`)},
	"temp_drive": []*regexp.Regexp{regexp.MustCompile(`Starting plotting progress into temporary dirs: (.*) and`)},
}

var runCounter = regexp.MustCompile(`Total time = (\d+)`)
var phaseTime = regexp.MustCompile(`Time for phase (\d) = (\d+)`)
var copyTime = regexp.MustCompile(`Copy time = (\d+)`)
var compressPhase = regexp.MustCompile(`Compressing tables (\d+)`)

var phaseTimings = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "phase_timings",
	Help: "The number of process currently plotting",
}, []string{
	"pid",
	"phase",
})

var completionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "plot_complete_counter",
	Help: "Number of plots completed by the given pid",
}, []string{
	"pid",
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
		pi, _ := strconv.ParseFloat(p, 64)
		ti, _ := strconv.ParseFloat(t, 64)
		bi, _ := strconv.ParseFloat(b, 64)
		bsi, _ := strconv.ParseFloat(bs, 64)

		// p4, t7, b 32/32
		// (3 * 25) + (7/7) * 20 + (32/32) * 5 = 100
		progress = ((pi - 1) * 20) + ((ti / 7) * 20) + (bi/bsi)*5
	}

	plotterProgress.WithLabelValues(pid).Set(progress)
}

func phaseChanged(ps *PlotterState, phase string, duration int) {
	ps.State["phase"] = phase
	pid := fmt.Sprintf("%d", ps.Pid)
	t := ps.State["table"]

	tt := PhaseTime{
		Phase:    phase,
		Run:      ps.Completions,
		Duration: time.Second * time.Duration(duration),
	}

	// phase times
	ps.PhaseTimes = append(ps.PhaseTimes, tt)
	phaseTimings.WithLabelValues(fmt.Sprintf("%d", ps.Pid), ps.Phase).Set(ps.Duration.Seconds())
	plotterState.WithLabelValues(pid, phase, t).Set(1)

	updateProgress(ps)
}

func (s *PlotterState) Update(entry *logEntry) {
	for k, r := range processors {
		if val, valid := checkRegexes(entry.msg, r); valid {
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
		pid := fmt.Sprintf("%d", s.Pid)
		dur, _ := strconv.Atoi(val[0])
		phaseChanged(s, "final", dur)

		s.Completions++
		completionCounter.WithLabelValues(pid).Inc()
	}

	s.State["last"] = entry.msg
}
