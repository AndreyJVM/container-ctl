package system

import (
	"fmt"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	defaultUnknownHost = "Unknown"
	defaultUnknownOS   = "Unknown"
	defaultEmptyKernel = "-"
	defaultEmptyUptime = "-"
	defaultCoreCount   = 1
	bytesPerGB         = 1024 * 1024 * 1024
	secondsPerDay      = uint64(86400)
	secondsPerHour     = uint64(3600)
	secondsPerMinute   = uint64(60)
)

var (
	hostInfoGetter      = func() (*host.InfoStat, error) { return host.Info() }
	osHostnameGetter    = os.Hostname
	cpuPercentGetter    = cpu.Percent
	cpuCountsGetter     = cpu.Counts
	virtualMemoryGetter = mem.VirtualMemory
)

// SystemInfo holds basic host system information.
type SystemInfo struct {
	Hostname  string
	OS        string
	Kernel    string
	Uptime    string
	UptimeSec uint64
}

// CPUInfo holds processor utilization percentage and core count.
type CPUInfo struct {
	Percent float64
	Cores   int
}

// MemoryInfo holds system RAM statistics in gigabytes and percentage.
type MemoryInfo struct {
	TotalGB     float64
	UsedGB      float64
	AvailableGB float64
	Percent     float64
}

// GetSystemInfo retrieves basic host system information including hostname, OS, kernel, and uptime.
func GetSystemInfo() SystemInfo {
	hInfo, err := hostInfoGetter()
	hostname, _ := osHostnameGetter()

	if err != nil || hInfo == nil {
		if hostname == "" {
			hostname = defaultUnknownHost
		}
		return SystemInfo{
			Hostname: hostname,
			OS:       defaultUnknownOS,
			Kernel:   defaultEmptyKernel,
			Uptime:   defaultEmptyUptime,
		}
	}

	finalHostname := hInfo.Hostname
	if finalHostname == "" {
		finalHostname = hostname
	}
	if finalHostname == "" {
		finalHostname = defaultUnknownHost
	}

	platform := strings.TrimSpace(fmt.Sprintf("%s %s", hInfo.Platform, hInfo.PlatformVersion))
	if platform == "" {
		platform = hInfo.OS
	}
	if platform == "" {
		platform = defaultUnknownOS
	}

	kernel := hInfo.KernelVersion
	if kernel == "" {
		kernel = defaultEmptyKernel
	}

	return SystemInfo{
		Hostname:  finalHostname,
		OS:        platform,
		Kernel:    kernel,
		Uptime:    formatUptime(hInfo.Uptime),
		UptimeSec: hInfo.Uptime,
	}
}

// GetCPUInfo retrieves current overall CPU utilization percentage and logical core count.
func GetCPUInfo() CPUInfo {
	percentages, err := cpuPercentGetter(0, false)
	pct := 0.0
	if err == nil && len(percentages) > 0 {
		pct = percentages[0]
		if pct < 0 {
			pct = 0
		}
	}

	cores, err := cpuCountsGetter(true)
	if err != nil || cores <= 0 {
		cores = defaultCoreCount
	}

	return CPUInfo{
		Percent: pct,
		Cores:   cores,
	}
}

// GetMemoryInfo retrieves virtual memory (RAM) utilization statistics.
func GetMemoryInfo() MemoryInfo {
	vm, err := virtualMemoryGetter()
	if err != nil || vm == nil {
		return MemoryInfo{}
	}

	return MemoryInfo{
		TotalGB:     float64(vm.Total) / bytesPerGB,
		UsedGB:      float64(vm.Used) / bytesPerGB,
		AvailableGB: float64(vm.Available) / bytesPerGB,
		Percent:     vm.UsedPercent,
	}
}

// formatUptime converts uptime in seconds to a human-readable duration string.
func formatUptime(uptimeSec uint64) string {
	days := uptimeSec / secondsPerDay
	hours := (uptimeSec % secondsPerDay) / secondsPerHour
	minutes := (uptimeSec % secondsPerHour) / secondsPerMinute

	if days > 0 {
		return fmt.Sprintf("%dd %02d:%02d", days, hours, minutes)
	}
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}
