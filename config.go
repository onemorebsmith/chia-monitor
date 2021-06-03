package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type UhaulConfig struct {
	StagingPaths []string `yaml:"StagingPaths"`
	FinalPaths   []string `yaml:"FinalPaths"`
}

type PlotterConfig struct {
	TempPath         string `yaml:"path"`
	FinalPath        string `yaml:"finalPath"`
	Ram              int    `yaml:"ram"`
	Tag              string `yaml:"tag"`
	StageConcurrency int    `yaml:"maxPerPhase"`
}

type MonitorConfig struct {
	TempPaths    []string        `yaml:"TempPaths"`
	StagingPaths []string        `yaml:"StagingPaths"`
	FinalPaths   []string        `yaml:"FinalPaths"`
	UhaulConfig  UhaulConfig     `yaml:"UHaul"`
	PathConfig   []PlotterConfig `yaml:"Plotter"`
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

	return config, err
}
