package monitor

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spectra-browser/spectra/internal/port"
)

// SystemMonitor tracks CPU and memory usage using /proc on Linux.
// Falls back to runtime.MemStats on non-Linux systems.
type SystemMonitor struct {
	cpuLimit    float64
	memLimit    float64
	mu          sync.RWMutex
	lastStats   port.MonitorStats
	lastUpdated time.Time
	ttl         time.Duration
}

func New(cpuLimit, memLimit int) *SystemMonitor {
	return &SystemMonitor{
		cpuLimit: float64(cpuLimit),
		memLimit: float64(memLimit),
		ttl:      2 * time.Second,
	}
}

func (m *SystemMonitor) Overloaded() (bool, string) {
	stats := m.Stats()
	if stats.CPUPercent >= m.cpuLimit {
		return true, fmt.Sprintf("CPU at %.0f%% (limit: %.0f%%)", stats.CPUPercent, m.cpuLimit)
	}
	if stats.MemoryPercent >= m.memLimit {
		return true, fmt.Sprintf("memory at %.0f%% (limit: %.0f%%)", stats.MemoryPercent, m.memLimit)
	}
	return false, ""
}

func (m *SystemMonitor) Stats() port.MonitorStats {
	m.mu.RLock()
	if time.Since(m.lastUpdated) < m.ttl {
		s := m.lastStats
		m.mu.RUnlock()
		return s
	}
	m.mu.RUnlock()

	cpu := m.readCPU()
	mem := m.readMemory()

	overloaded := false
	reason := ""
	if cpu >= m.cpuLimit {
		overloaded = true
		reason = fmt.Sprintf("CPU at %.0f%%", cpu)
	} else if mem >= m.memLimit {
		overloaded = true
		reason = fmt.Sprintf("memory at %.0f%%", mem)
	}

	stats := port.MonitorStats{
		CPUPercent:    cpu,
		MemoryPercent: mem,
		Overloaded:    overloaded,
		Reason:        reason,
	}

	m.mu.Lock()
	m.lastStats = stats
	m.lastUpdated = time.Now()
	m.mu.Unlock()

	return stats
}

// readCPU reads CPU usage from /proc/stat (Linux) or returns 0 on other OS.
func (m *SystemMonitor) readCPU() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	s1 := readProcStat()
	time.Sleep(200 * time.Millisecond)
	s2 := readProcStat()

	total := float64((s2[0] + s2[1] + s2[2] + s2[3]) - (s1[0] + s1[1] + s1[2] + s1[3]))
	idle := float64(s2[3] - s1[3])
	if total == 0 {
		return 0
	}
	return (1 - idle/total) * 100
}

// readMemory reads memory usage from /proc/meminfo (Linux) or runtime on other OS.
func (m *SystemMonitor) readMemory() float64 {
	if runtime.GOOS != "linux" {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		if ms.Sys == 0 {
			return 0
		}
		return float64(ms.HeapInuse) / float64(ms.Sys) * 100
	}

	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	vals := make(map[string]uint64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 2 {
			n, _ := strconv.ParseUint(parts[1], 10, 64)
			vals[strings.TrimSuffix(parts[0], ":")] = n
		}
	}

	total := vals["MemTotal"]
	available := vals["MemAvailable"]
	if total == 0 {
		return 0
	}
	used := total - available
	return float64(used) / float64(total) * 100
}

func readProcStat() [4]uint64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return [4]uint64{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 5 {
			break
		}
		var vals [4]uint64
		for i := 0; i < 4; i++ {
			vals[i], _ = strconv.ParseUint(parts[i+1], 10, 64)
		}
		return vals
	}
	return [4]uint64{}
}
