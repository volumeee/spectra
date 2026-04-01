package domain

import "time"

type BrowserConfig struct {
	MaxInstances  int           `yaml:"max_instances" json:"max_instances"`
	LaunchTimeout time.Duration `yaml:"launch_timeout" json:"launch_timeout"`
	IdleTimeout   time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}
