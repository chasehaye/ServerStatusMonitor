package main

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Server struct {
	Name     string        `yaml:"name"`
	URL      string        `yaml:"url"`
	Timeout  time.Duration `yaml:"timeout"`
	Interval time.Duration `yaml:"interval"`
}

func (s Server) interval(globalInterval time.Duration) time.Duration {
    if s.Interval == 0 {
        return globalInterval
    }
    return s.Interval
}

func (s Server) timeout() time.Duration {
	if s.Timeout == 0 {
		return 5 * time.Second
	}
	return s.Timeout
}

type Config struct {
	Interval time.Duration `yaml:"interval"`
	Servers  []Server      `yaml:"servers"`
}

func loadConfig(path string) (Config, error) {
	cfg := Config{
		Interval: 10 * time.Second, // default refresh interval
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
