package engine

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	netio "github.com/shirou/gopsutil/v4/net"
)

type snapshotCollector struct {
	ctx      context.Context
	snapshot *SystemSnapshot
}

func Snapshot(ctx context.Context) (*SystemSnapshot, error) {
	snapshot, err := newBaseSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	collector := snapshotCollector{
		ctx:      ctx,
		snapshot: snapshot,
	}
	collector.captureCPU()
	collector.captureMemory()
	collector.captureDisk()
	collector.captureNetwork()
	collector.captureLoadAndUsers()
	enrichPlatformSnapshot(ctx, snapshot)
	snapshot.TopProcesses, snapshot.ProcessCount = captureTopProcesses(ctx)
	snapshot.HealthScore, snapshot.HealthLabel = deriveHealth(snapshot)
	snapshot.OperatorAlerts = deriveOperatorAlerts(snapshot)
	snapshot.Highlights = deriveHighlights(snapshot)
	return snapshot, nil
}

func newBaseSnapshot(ctx context.Context) (*SystemSnapshot, error) {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, err
	}
	snapshot := &SystemSnapshot{
		Platform:        runtime.GOOS,
		Architecture:    runtime.GOARCH,
		Hostname:        info.Hostname,
		OS:              info.OS,
		PlatformFamily:  info.PlatformFamily,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		BootTimeSeconds: info.BootTime,
		UptimeSeconds:   info.Uptime,
		CPUCores:        runtime.NumCPU(),
		CollectedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	return snapshot, nil
}

func (c snapshotCollector) captureCPU() {
	if physicalCores, err := cpu.CountsWithContext(c.ctx, false); err == nil && physicalCores > 0 {
		c.snapshot.CPUPhysicalCores = physicalCores
	}
	if infoStats, err := cpu.InfoWithContext(c.ctx); err == nil && len(infoStats) > 0 {
		c.snapshot.CPUModel = infoStats[0].ModelName
	}
	if values, err := safeCPUPercentWithContext(c.ctx, false); err == nil && len(values) > 0 {
		c.snapshot.CPUPercent = values[0]
	} else if err != nil {
		c.warn("cpu", err)
	}
	if perCore, err := safeCPUPercentWithContext(c.ctx, true); err == nil {
		c.snapshot.CPUPerCore = perCore
	} else {
		c.warn("cpu_per_core", err)
	}
}

func (c snapshotCollector) captureMemory() {
	if vm, err := mem.VirtualMemoryWithContext(c.ctx); err == nil {
		c.snapshot.MemoryUsedPercent = vm.UsedPercent
		c.snapshot.MemoryUsedBytes = vm.Used
		c.snapshot.MemoryTotalBytes = vm.Total
	} else {
		c.warn("memory", err)
	}
	if swap, err := mem.SwapMemoryWithContext(c.ctx); err == nil {
		c.snapshot.SwapUsedPercent = swap.UsedPercent
		c.snapshot.SwapUsedBytes = swap.Used
		c.snapshot.SwapTotalBytes = swap.Total
	} else {
		c.warn("swap", err)
	}
}

func (c snapshotCollector) captureDisk() {
	if usage, err := disk.UsageWithContext(c.ctx, systemDiskRoot()); err == nil {
		c.snapshot.DiskUsedPercent = usage.UsedPercent
		c.snapshot.DiskFreeBytes = usage.Free
		c.snapshot.DiskTotalBytes = usage.Total
	} else {
		c.warn("disk", err)
	}
	if ioCounters, err := disk.IOCountersWithContext(c.ctx); err == nil {
		var io DiskIOSnapshot
		for _, counter := range ioCounters {
			io.ReadBytes += counter.ReadBytes
			io.WriteBytes += counter.WriteBytes
		}
		if io.ReadBytes > 0 || io.WriteBytes > 0 {
			c.snapshot.DiskIO = &io
		}
	} else {
		c.warn("disk_io", err)
	}
}

func (c snapshotCollector) captureNetwork() {
	if counters, err := netio.IOCountersWithContext(c.ctx, false); err == nil && len(counters) > 0 {
		c.snapshot.NetworkRxBytes = counters[0].BytesRecv
		c.snapshot.NetworkTxBytes = counters[0].BytesSent
	} else if err != nil {
		c.warn("network", err)
	}
	if interfaces, err := netio.InterfacesWithContext(c.ctx); err == nil {
		c.snapshot.ActiveNetworkIfaces, c.snapshot.NetworkInterfaceCount = activeNetworkInterfaces(interfaces)
	} else {
		c.warn("network_interfaces", err)
	}
}

func (c snapshotCollector) captureLoadAndUsers() {
	if avg, err := load.AvgWithContext(c.ctx); err == nil {
		c.snapshot.Load1 = avg.Load1
	} else if runtime.GOOS != "windows" {
		c.warn("load", err)
	}
	if c.snapshot.CPUCores > 0 {
		c.snapshot.LoadPerCPU = c.snapshot.Load1 / float64(c.snapshot.CPUCores)
	}
	if users, err := host.UsersWithContext(c.ctx); err == nil {
		c.snapshot.LoggedInUsers = len(users)
	} else {
		c.warn("users", err)
	}
	if system, role, err := host.VirtualizationWithContext(c.ctx); err == nil {
		c.snapshot.VirtualizationSystem = system
		c.snapshot.VirtualizationRole = role
	}
}

func (c snapshotCollector) warn(scope string, err error) {
	if err == nil {
		return
	}
	c.snapshot.Warnings = append(c.snapshot.Warnings, scope+": "+err.Error())
}
