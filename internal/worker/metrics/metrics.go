package metrics

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	"xvault/internal/worker/client"
)

// Collector collects system metrics
type Collector struct {
	startTime     time.Time
	storagePath   string
	activeJobsPtr *int
}

// NewCollector creates a new metrics collector
func NewCollector(storagePath string, activeJobsPtr *int) *Collector {
	return &Collector{
		startTime:     time.Now(),
		storagePath:   storagePath,
		activeJobsPtr: activeJobsPtr,
	}
}

// Collect gathers current system metrics
func (c *Collector) Collect() *client.SystemMetrics {
	metrics := &client.SystemMetrics{
		UptimeSeconds: int64(time.Since(c.startTime).Seconds()),
	}

	if c.activeJobsPtr != nil {
		metrics.ActiveJobs = *c.activeJobsPtr
	}

	// Collect CPU usage
	metrics.CPUPercent = c.getCPUPercent()

	// Collect memory usage
	c.collectMemoryMetrics(metrics)

	// Collect disk usage for storage path
	c.collectDiskMetrics(metrics)

	return metrics
}

// getCPUPercent reads CPU usage from /proc/stat
func (c *Collector) getCPUPercent() float64 {
	// Read initial CPU stats
	idle1, total1 := c.readCPUStat()
	if total1 == 0 {
		return 0
	}

	// Wait a short interval
	time.Sleep(100 * time.Millisecond)

	// Read CPU stats again
	idle2, total2 := c.readCPUStat()
	if total2 == 0 {
		return 0
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)

	if totalDelta == 0 {
		return 0
	}

	return (1.0 - idleDelta/totalDelta) * 100.0
}

// readCPUStat reads the CPU stats from /proc/stat
func (c *Collector) readCPUStat() (idle, total uint64) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0, 0
			}

			var values []uint64
			for _, field := range fields[1:] {
				if v, err := strconv.ParseUint(field, 10, 64); err == nil {
					values = append(values, v)
				}
			}

			if len(values) >= 4 {
				// user, nice, system, idle, iowait, irq, softirq
				for _, v := range values {
					total += v
				}
				idle = values[3] // idle is the 4th field
				if len(values) > 4 {
					idle += values[4] // add iowait
				}
			}
			break
		}
	}

	return idle, total
}

// collectMemoryMetrics reads memory info from /proc/meminfo
func (c *Collector) collectMemoryMetrics(metrics *client.SystemMetrics) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()

	var memTotal, memAvailable, memFree, buffers, cached int64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}

		// Values in /proc/meminfo are in kB
		switch fields[0] {
		case "MemTotal:":
			memTotal = value * 1024
		case "MemAvailable:":
			memAvailable = value * 1024
		case "MemFree:":
			memFree = value * 1024
		case "Buffers:":
			buffers = value * 1024
		case "Cached:":
			cached = value * 1024
		}
	}

	metrics.MemoryTotalBytes = memTotal

	// Use MemAvailable if present (more accurate), otherwise calculate
	if memAvailable > 0 {
		metrics.MemoryUsedBytes = memTotal - memAvailable
	} else {
		metrics.MemoryUsedBytes = memTotal - memFree - buffers - cached
	}

	if memTotal > 0 {
		metrics.MemoryPercent = float64(metrics.MemoryUsedBytes) / float64(memTotal) * 100.0
	}
}

// collectDiskMetrics gets disk usage for the storage path
func (c *Collector) collectDiskMetrics(metrics *client.SystemMetrics) {
	var stat StatFs
	if err := Statfs(c.storagePath, &stat); err != nil {
		return
	}

	metrics.DiskTotalBytes = int64(stat.Blocks) * int64(stat.Bsize)
	metrics.DiskFreeBytes = int64(stat.Bavail) * int64(stat.Bsize)
	metrics.DiskUsedBytes = metrics.DiskTotalBytes - int64(stat.Bfree)*int64(stat.Bsize)

	if metrics.DiskTotalBytes > 0 {
		metrics.DiskPercent = float64(metrics.DiskUsedBytes) / float64(metrics.DiskTotalBytes) * 100.0
	}
}
