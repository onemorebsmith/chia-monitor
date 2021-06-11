package main

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type UhaulConfig struct {
	StagingPaths []string `yaml:"StagingPaths"`
	FinalPaths   []string `yaml:"FinalPaths"`
}

type PlotterConfig struct {
	TempPath         string        `yaml:"tempPath"`
	FinalPath        string        `yaml:"finalPath"`
	Ram              string        `yaml:"ram"`
	Tag              string        `yaml:"tag"`
	Buckets          string        `yaml:"buckets"`
	Cores            string        `yaml:"cores"`
	StageConcurrency int           `yaml:"maxActivePlotters"`
	MaxPhase1        int           `yaml:"maxPhase1"`
	MinCooldown      time.Duration `yaml:"minDelay"`
}

type MonitorConfig struct {
	TempPaths          []string         `yaml:"TempPaths"`
	StagingPaths       []string         `yaml:"StagingPaths"`
	FinalPaths         []string         `yaml:"FinalPaths"`
	UhaulConfig        UhaulConfig      `yaml:"UHaul"`
	PlotterConfig      []*PlotterConfig `yaml:"Plotter"`
	ChiaPath           string           `yaml:"ChiaPath"`
	FarmMonitorEnabled bool             `yaml:"FarmMonitorEnabled"`
	UhaulEnabled       bool             `yaml:"UhaulEnabled"`
}

func parseConfig(path string) (MonitorConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return MonitorConfig{}, err
	}

	config := MonitorConfig{}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return MonitorConfig{}, err
	}

	for _, v := range config.PlotterConfig {
		if v.Buckets == "" {
			v.Buckets = "128"
		}

		if v.Ram == "" {
			v.Ram = "4000"
		}

		if v.Cores == "" {
			v.Cores = "2"
		}

		v.TempPath = filepath.Clean(v.TempPath)
		v.FinalPath = filepath.Clean(v.FinalPath)

		if v.TempPath == "" {
			v.TempPath = v.FinalPath
		}
	}

	return config, err
}
