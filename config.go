package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type MonitorConfig struct {
	TempPaths    []string `yaml:"TempPaths"`
	StagingPaths []string `yaml:"StagingPaths"`
	FinalPaths   []string `yaml:"FinalPaths"`
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
