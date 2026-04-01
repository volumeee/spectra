package metrics

import (
	"sync"
	"sync/atomic"

	"github.com/spectra-browser/spectra/internal/port"
)

type pluginStats struct {
	requests    atomic.Int64
	success     atomic.Int64
	failed      atomic.Int64
	totalDurMs  atomic.Int64
}

// Collector is a thread-safe in-memory metrics collector.
type Collector struct {
	totalRequests atomic.Int64
	totalSuccess  atomic.Int64
	totalFailed   atomic.Int64
	totalTimedOut atomic.Int64
	totalDurMs    atomic.Int64
	byPlugin      sync.Map // map[string]*pluginStats
}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) RecordRequest(plugin, _ string, durationMs int64, success bool) {
	c.totalRequests.Add(1)
	c.totalDurMs.Add(durationMs)
	if success {
		c.totalSuccess.Add(1)
	} else {
		c.totalFailed.Add(1)
	}

	v, _ := c.byPlugin.LoadOrStore(plugin, &pluginStats{})
	ps := v.(*pluginStats)
	ps.requests.Add(1)
	ps.totalDurMs.Add(durationMs)
	if success {
		ps.success.Add(1)
	} else {
		ps.failed.Add(1)
	}
}

func (c *Collector) Snapshot() port.MetricsSnapshot {
	total := c.totalRequests.Load()
	totalDur := c.totalDurMs.Load()

	var avgDur float64
	if total > 0 {
		avgDur = float64(totalDur) / float64(total)
	}

	byPlugin := make(map[string]port.PluginMetrics)
	c.byPlugin.Range(func(k, v any) bool {
		ps := v.(*pluginStats)
		req := ps.requests.Load()
		dur := ps.totalDurMs.Load()
		var avg float64
		if req > 0 {
			avg = float64(dur) / float64(req)
		}
		byPlugin[k.(string)] = port.PluginMetrics{
			Requests:    req,
			Success:     ps.success.Load(),
			Failed:      ps.failed.Load(),
			AvgDuration: avg,
		}
		return true
	})

	return port.MetricsSnapshot{
		TotalRequests: total,
		TotalSuccess:  c.totalSuccess.Load(),
		TotalFailed:   c.totalFailed.Load(),
		TotalTimedOut: c.totalTimedOut.Load(),
		AvgDurationMs: avgDur,
		ByPlugin:      byPlugin,
	}
}
