package operations

import (
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/types"
)

import "testing"

func TestCalculateContainerStats(t *testing.T) {
	t.Run("standard stats calculation", func(t *testing.T) {
		var raw types.ContainerStatsRaw
		raw.CPUStats.CPUUsage.TotalUsage = 200000000
		raw.PreCPUStats.CPUUsage.TotalUsage = 100000000
		raw.CPUStats.SystemCPUUsage = 1000000000
		raw.PreCPUStats.SystemCPUUsage = 900000000
		raw.CPUStats.OnlineCPUs = 2

		// 50 MB usage out of 100 MB limit
		raw.MemoryStats.Usage = 50 * 1024 * 1024
		raw.MemoryStats.Limit = 100 * 1024 * 1024

		// Networks map
		raw.Networks = map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		}{
			"eth0": {RxBytes: 15 * 1024 * 1024, TxBytes: 25 * 1024 * 1024},
			"eth1": {RxBytes: 5 * 1024 * 1024, TxBytes: 10 * 1024 * 1024},
		}

		// Block I/O
		raw.BlkioStats.IOServiceBytesRecursive = []struct {
			Major uint64 `json:"major"`
			Minor uint64 `json:"minor"`
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		}{
			{Major: 8, Minor: 0, Op: "Read", Value: 12 * 1024 * 1024},
			{Major: 8, Minor: 0, Op: "Write", Value: 34 * 1024 * 1024},
			{Major: 8, Minor: 0, Op: "Sync", Value: 1000},
		}

		stats := CalculateContainerStats(raw)

		// CPU delta = 100,000,000; System delta = 100,000,000; 2 CPUs -> 1.0 * 2 * 100 = 200%
		if stats.CPUPerc != 200.0 {
			t.Errorf("expected CPU 200.0%%, got %.2f%%", stats.CPUPerc)
		}

		if stats.MemUsageMB != 50.0 {
			t.Errorf("expected MemUsageMB 50.0, got %.2f", stats.MemUsageMB)
		}

		if stats.MemLimitMB != 100.0 {
			t.Errorf("expected MemLimitMB 100.0, got %.2f", stats.MemLimitMB)
		}

		if stats.MemPerc != 50.0 {
			t.Errorf("expected MemPerc 50.0%%, got %.2f%%", stats.MemPerc)
		}

		// Net I/O sum: 15MB + 5MB = 20MB, 25MB + 10MB = 35MB
		expectedRx := uint64(20 * 1024 * 1024)
		expectedTx := uint64(35 * 1024 * 1024)
		if stats.NetRxBytes != expectedRx {
			t.Errorf("expected NetRxBytes %d, got %d", expectedRx, stats.NetRxBytes)
		}
		if stats.NetTxBytes != expectedTx {
			t.Errorf("expected NetTxBytes %d, got %d", expectedTx, stats.NetTxBytes)
		}

		// Block I/O: 12MB read, 34MB write
		expectedRead := uint64(12 * 1024 * 1024)
		expectedWrite := uint64(34 * 1024 * 1024)
		if stats.BlockReadBytes != expectedRead {
			t.Errorf("expected BlockReadBytes %d, got %d", expectedRead, stats.BlockReadBytes)
		}
		if stats.BlockWriteBytes != expectedWrite {
			t.Errorf("expected BlockWriteBytes %d, got %d", expectedWrite, stats.BlockWriteBytes)
		}
	})

	t.Run("inactive file cache deduction", func(t *testing.T) {
		var raw types.ContainerStatsRaw
		raw.MemoryStats.Usage = 100 * 1024 * 1024
		raw.MemoryStats.Limit = 200 * 1024 * 1024
		raw.MemoryStats.Stats.TotalInactiveFile = 20 * 1024 * 1024

		stats := CalculateContainerStats(raw)
		// 100 MB - 20 MB = 80 MB
		if stats.MemUsageMB != 80.0 {
			t.Errorf("expected MemUsageMB 80.0 after cache deduction, got %.2f", stats.MemUsageMB)
		}
	})

	t.Run("network fallback to single network struct", func(t *testing.T) {
		var raw types.ContainerStatsRaw
		raw.Network.RxBytes = 1000
		raw.Network.TxBytes = 2000

		stats := CalculateContainerStats(raw)
		if stats.NetRxBytes != 1000 || stats.NetTxBytes != 2000 {
			t.Errorf("expected NetRx=1000, NetTx=2000, got NetRx=%d, NetTx=%d", stats.NetRxBytes, stats.NetTxBytes)
		}
	})
}
