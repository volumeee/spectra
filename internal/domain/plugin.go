package domain

import "time"

type PluginStatus string

const (
	PluginStatusStopped  PluginStatus = "stopped"
	PluginStatusRunning  PluginStatus = "running"
	PluginStatusError    PluginStatus = "error"
)

type PluginManifest struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Command string   `json:"command"`
	Methods []string `json:"methods"`
}

type PluginInfo struct {
	Manifest  PluginManifest `json:"manifest"`
	Status    PluginStatus   `json:"status"`
	StartedAt *time.Time     `json:"started_at,omitempty"`
}
