package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type UhaulConfig struct {
	StagingPaths []string `yaml:"StagingPaths"`
	FinalPaths   []string `yaml:"FinalPaths"`
}

type MonitorConfig struct {
	TempPaths    []string    `yaml:"TempPaths"`
	StagingPaths []string    `yaml:"StagingPaths"`
	FinalPaths   []string    `yaml:"FinalPaths"`
	UhaulConfig  UhaulConfig `yaml:"UHaul"`
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
