package config

import (
	"github.com/XANi/collectd2metrics/promwriter"
	"gopkg.in/yaml.v2"
	"time"
)

type Config struct {
	PrometheusWriter []promwriter.Config `yaml:"prometheus_writer"`
}

func (c *Config) GetDefaultConfig() string {
	cfg := Config{
		PrometheusWriter: []promwriter.Config{{
			URL:              "http://127.0.0.1:8480/insert/0:0/prometheus/api/v1/import/prometheus",
			Timeout:          time.Second * 30,
			MaxBatchDuration: time.Second * 5,
			MaxBatchLength:   1000,
		}},
	}
	out, _ := yaml.Marshal(&cfg)
	return string(out)
}
