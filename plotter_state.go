package main

import (
	"regexp"
	"time"
)

type PlotterState struct {
	State map[string]string
	Pid   int

	Phase       string
	Table       string
	Bucket      string
	PlotSz      int
	BucketCount int
	MaxRamMb    int
	MaxThread   int
	Duration    time.Duration
}

var processors = map[string]*regexp.Regexp{
	"plotSize":   regexp.MustCompile(`Plot size is: (\d+)`),
	"maxRam":     regexp.MustCompile(`Buffer size is: (\d+)MiB`),
	"bucketSize": regexp.MustCompile(`Using (\d+) buckets`),
	"phase":      regexp.MustCompile(`.*Starting phase (\d)/*.`),
	"table":      regexp.MustCompile(`.*table (\d)`),
	"bucket":     regexp.MustCompile(`.*Bucket (\d)`),
}

func checkRegex(s string, r *regexp.Regexp) (string, bool) {
	if r.Match([]byte(s)) {
		matches := r.FindStringSubmatch(s)
		if len(matches) > 1 {
			return matches[1], true
		}
	}

	return "", false
}

func (s *PlotterState) Update(entry *logEntry) {
	for k, r := range processors {
		if val, valid := checkRegex(entry.msg, r); valid {
			s.State[k] = val
		}
	}

	s.State["last"] = entry.msg
}
