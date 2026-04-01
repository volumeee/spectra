package browser

import (
	"github.com/go-rod/rod/lib/launcher"
)

type LaunchConfig struct {
	Headless bool
	NoSandbox bool
}

func DefaultLaunchConfig() *LaunchConfig {
	return &LaunchConfig{Headless: true, NoSandbox: true}
}

func (c *LaunchConfig) LaunchURL() (string, error) {
	l := launcher.New().Headless(c.Headless)
	if c.NoSandbox {
		l = l.NoSandbox(true)
	}
	return l.Launch()
}
