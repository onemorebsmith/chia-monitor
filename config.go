package main

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type UhaulConfig struct {
	PruneDateRaw string `yaml:"PruneDate"`
	PruneDate    time.Time
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
	PoolKey          string        `yaml:"poolKey"`
	StageConcurrency int           `yaml:"maxActivePlotters"`
	MaxPhase1        int           `yaml:"maxPhase1"`
	MinCooldown      time.Duration `yaml:"minDelay"`
	StartDelay       time.Duration `yaml:"startDelay"`
}

type DriveMonitorConfig struct {
	TempPaths    []string `yaml:"TempPaths"`
	FinalPaths   []string `yaml:"FinalPaths"`
	StagingPaths []string `yaml:"StagingPaths"`
}

type MonitorConfig struct {
	UhaulConfig         UhaulConfig        `yaml:"UHaul"`
	DriveMonitorConfig  DriveMonitorConfig `yaml:"DriveMonitor"`
	PlotterConfig       []*PlotterConfig   `yaml:"Plotter"`
	ChiaPath            string             `yaml:"ChiaPath"`
	FarmMonitorEnabled  bool               `yaml:"FarmMonitorEnabled"`
	UhaulEnabled        bool               `yaml:"UhaulEnabled"`
	PlotterEnabled      bool               `yaml:"PlotterEnabled"`
	DriveMonitorEnabled bool               `yaml:"DriveMonitorEnabled"`
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

	if config.UhaulConfig.PruneDateRaw != "" {
		config.UhaulConfig.PruneDate, err = time.Parse("1-2-2006", config.UhaulConfig.PruneDateRaw)
		if err != nil {
			return config, err
		}
	}

	return config, err
}
