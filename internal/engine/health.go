package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	netio "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

var cpuPercentWithContext = cpu.PercentWithContext

type ProcessSnapshot struct {
	PID            int32   `json:"pid"`
	Name           string  `json:"name"`
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryPercent  float64 `json:"memory_percent"`
	MemoryRSSBytes uint64  `json:"memory_rss_bytes"`
}

type BatterySnapshot struct {
	Percent          float64 `json:"percent"`
	State            string  `json:"state"`
	RemainingMinutes int     `json:"remaining_minutes,omitempty"`
	CycleCount       int     `json:"cycle_count,omitempty"`
	Condition        string  `json:"condition,omitempty"`
	CapacityPercent  int     `json:"capacity_percent,omitempty"`
}

type BluetoothDeviceSnapshot struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	Battery   string `json:"battery,omitempty"`
}

type ProxySnapshot struct {
	Enabled bool   `json:"enabled"`
	HTTP    string `json:"http,omitempty"`
	HTTPS   string `json:"https,omitempty"`
	Bypass  string `json:"bypass,omitempty"`
}

type DiskIOSnapshot struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
}

type SystemSnapshot struct {
	Platform              string                    `json:"platform"`
	Architecture          string                    `json:"architecture"`
	Hostname              string                    `json:"hostname"`
	OS                    string                    `json:"os"`
	PlatformFamily        string                    `json:"platform_family,omitempty"`
	PlatformVersion       string                    `json:"platform_version"`
	KernelVersion         string                    `json:"kernel_version,omitempty"`
	BootTimeSeconds       uint64                    `json:"boot_time_seconds,omitempty"`
	HardwareModel         string                    `json:"hardware_model,omitempty"`
	CPUModel              string                    `json:"cpu_model,omitempty"`
	GPUModel              string                    `json:"gpu_model,omitempty"`
	GPUUsagePercent       float64                   `json:"gpu_usage_percent,omitempty"`
	GPURendererPercent    float64                   `json:"gpu_renderer_percent,omitempty"`
	GPUTilerPercent       float64                   `json:"gpu_tiler_percent,omitempty"`
	DisplayResolution     string                    `json:"display_resolution,omitempty"`
	DisplayRefreshRate    string                    `json:"display_refresh_rate,omitempty"`
	DisplayCount          int                       `json:"display_count,omitempty"`
	BluetoothPowered      bool                      `json:"bluetooth_powered,omitempty"`
	BluetoothConnected    int                       `json:"bluetooth_connected_count,omitempty"`
	BluetoothDevices      []BluetoothDeviceSnapshot `json:"bluetooth_devices,omitempty"`
	ThermalState          string                    `json:"thermal_state,omitempty"`
	CPUTempCelsius        float64                   `json:"cpu_temp_celsius,omitempty"`
	FanSpeedRPM           int                       `json:"fan_speed_rpm,omitempty"`
	SystemPowerWatts      float64                   `json:"system_power_watts,omitempty"`
	AdapterPowerWatts     float64                   `json:"adapter_power_watts,omitempty"`
	BatteryPowerWatts     float64                   `json:"battery_power_watts,omitempty"`
	UptimeSeconds         uint64                    `json:"uptime_seconds"`
	CPUCores              int                       `json:"cpu_cores"`
	CPUPhysicalCores      int                       `json:"cpu_physical_cores,omitempty"`
	PerformanceCores      int                       `json:"performance_cores,omitempty"`
	EfficiencyCores       int                       `json:"efficiency_cores,omitempty"`
	CPUPercent            float64                   `json:"cpu_percent"`
	CPUPerCore            []float64                 `json:"cpu_per_core,omitempty"`
	MemoryUsedPercent     float64                   `json:"memory_used_percent"`
	MemoryUsedBytes       uint64                    `json:"memory_used_bytes"`
	MemoryTotalBytes      uint64                    `json:"memory_total_bytes"`
	SwapUsedPercent       float64                   `json:"swap_used_percent"`
	SwapUsedBytes         uint64                    `json:"swap_used_bytes"`
	SwapTotalBytes        uint64                    `json:"swap_total_bytes"`
	DiskUsedPercent       float64                   `json:"disk_used_percent"`
	DiskFreeBytes         uint64                    `json:"disk_free_bytes"`
	DiskTotalBytes        uint64                    `json:"disk_total_bytes"`
	DiskIO                *DiskIOSnapshot           `json:"disk_io,omitempty"`
	NetworkRxBytes        uint64                    `json:"network_rx_bytes"`
	NetworkTxBytes        uint64                    `json:"network_tx_bytes"`
	NetworkInterfaceCount int                       `json:"network_interface_count,omitempty"`
	ActiveNetworkIfaces   []string                  `json:"active_network_ifaces,omitempty"`
	Load1                 float64                   `json:"load1"`
	LoadPerCPU            float64                   `json:"load_per_cpu"`
	ProcessCount          int                       `json:"process_count"`
	LoggedInUsers         int                       `json:"logged_in_users"`
	VirtualizationSystem  string                    `json:"virtualization_system,omitempty"`
	VirtualizationRole    string                    `json:"virtualization_role,omitempty"`
	Battery               *BatterySnapshot          `json:"battery,omitempty"`
	PowerSource           string                    `json:"power_source,omitempty"`
	Proxy                 *ProxySnapshot            `json:"proxy,omitempty"`
	HealthScore           int                       `json:"health_score"`
	HealthLabel           string                    `json:"health_label"`
	OperatorAlerts        []string                  `json:"operator_alerts,omitempty"`
	Highlights            []string                  `json:"highlights,omitempty"`
	TopProcesses          []ProcessSnapshot         `json:"top_processes,omitempty"`
	CollectedAt           string                    `json:"collected_at"`
	Warnings              []string                  `json:"warnings,omitempty"`
}

var enrichPlatformSnapshot = func(context.Context, *SystemSnapshot) {}

func Snapshot(ctx context.Context) (*SystemSnapshot, error) {
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
	if physicalCores, err := cpu.CountsWithContext(ctx, false); err == nil && physicalCores > 0 {
		snapshot.CPUPhysicalCores = physicalCores
	}
	if infoStats, err := cpu.InfoWithContext(ctx); err == nil && len(infoStats) > 0 {
		snapshot.CPUModel = strings.TrimSpace(infoStats[0].ModelName)
	}
	if values, err := safeCPUPercentWithContext(ctx, false); err == nil && len(values) > 0 {
		snapshot.CPUPercent = values[0]
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "cpu: "+err.Error())
	}
	if perCore, err := safeCPUPercentWithContext(ctx, true); err == nil {
		snapshot.CPUPerCore = perCore
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "cpu_per_core: "+err.Error())
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		snapshot.MemoryUsedPercent = vm.UsedPercent
		snapshot.MemoryUsedBytes = vm.Used
		snapshot.MemoryTotalBytes = vm.Total
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "memory: "+err.Error())
	}
	if swap, err := mem.SwapMemoryWithContext(ctx); err == nil {
		snapshot.SwapUsedPercent = swap.UsedPercent
		snapshot.SwapUsedBytes = swap.Used
		snapshot.SwapTotalBytes = swap.Total
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "swap: "+err.Error())
	}
	if usage, err := disk.UsageWithContext(ctx, systemDiskRoot()); err == nil {
		snapshot.DiskUsedPercent = usage.UsedPercent
		snapshot.DiskFreeBytes = usage.Free
		snapshot.DiskTotalBytes = usage.Total
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "disk: "+err.Error())
	}
	if counters, err := netio.IOCountersWithContext(ctx, false); err == nil && len(counters) > 0 {
		snapshot.NetworkRxBytes = counters[0].BytesRecv
		snapshot.NetworkTxBytes = counters[0].BytesSent
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "network: "+err.Error())
	}
	if interfaces, err := netio.InterfacesWithContext(ctx); err == nil {
		snapshot.ActiveNetworkIfaces, snapshot.NetworkInterfaceCount = activeNetworkInterfaces(interfaces)
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "network_interfaces: "+err.Error())
	}
	if ioCounters, err := disk.IOCountersWithContext(ctx); err == nil {
		var io DiskIOSnapshot
		for _, counter := range ioCounters {
			io.ReadBytes += counter.ReadBytes
			io.WriteBytes += counter.WriteBytes
		}
		if io.ReadBytes > 0 || io.WriteBytes > 0 {
			snapshot.DiskIO = &io
		}
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "disk_io: "+err.Error())
	}
	if avg, err := load.AvgWithContext(ctx); err == nil {
		snapshot.Load1 = avg.Load1
	} else if runtime.GOOS != "windows" {
		snapshot.Warnings = append(snapshot.Warnings, "load: "+err.Error())
	}
	if snapshot.CPUCores > 0 {
		snapshot.LoadPerCPU = snapshot.Load1 / float64(snapshot.CPUCores)
	}
	if users, err := host.UsersWithContext(ctx); err == nil {
		snapshot.LoggedInUsers = len(users)
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "users: "+err.Error())
	}
	if system, role, err := host.VirtualizationWithContext(ctx); err == nil {
		snapshot.VirtualizationSystem = system
		snapshot.VirtualizationRole = role
	}
	enrichPlatformSnapshot(ctx, snapshot)
	snapshot.TopProcesses, snapshot.ProcessCount = captureTopProcesses(ctx)
	snapshot.HealthScore, snapshot.HealthLabel = deriveHealth(snapshot)
	snapshot.OperatorAlerts = deriveOperatorAlerts(snapshot)
	snapshot.Highlights = deriveHighlights(snapshot)
	return snapshot, nil
}

func captureTopProcesses(ctx context.Context) ([]ProcessSnapshot, int) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, 0
	}
	candidates := make([]ProcessSnapshot, 0, len(procs))
	for _, proc := range procs {
		select {
		case <-ctx.Done():
			return candidates, len(procs)
		default:
		}
		name, err := proc.NameWithContext(ctx)
		if err != nil || name == "" {
			continue
		}
		memInfo, err := proc.MemoryInfoWithContext(ctx)
		if err != nil || memInfo == nil || memInfo.RSS == 0 {
			continue
		}
		memoryPercent, _ := proc.MemoryPercentWithContext(ctx)
		candidates = append(candidates, ProcessSnapshot{
			PID:            proc.Pid,
			Name:           name,
			MemoryPercent:  float64(memoryPercent),
			MemoryRSSBytes: memInfo.RSS,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].MemoryRSSBytes == candidates[j].MemoryRSSBytes {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].MemoryRSSBytes > candidates[j].MemoryRSSBytes
	})
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}
	for idx := range candidates {
		proc, err := process.NewProcessWithContext(ctx, candidates[idx].PID)
		if err != nil {
			continue
		}
		if cpuPercent, err := proc.CPUPercentWithContext(ctx); err == nil {
			candidates[idx].CPUPercent = cpuPercent
		}
	}
	return candidates, len(procs)
}

func safeCPUPercentWithContext(ctx context.Context, percpu bool) (values []float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cpu sampling panic: %v", r)
			values = nil
		}
	}()
	return cpuPercentWithContext(ctx, 0, percpu)
}

func deriveHealth(snapshot *SystemSnapshot) (int, string) {
	score := 100
	score -= penalty(snapshot.CPUPercent, 55, 30, 1.2)
	score -= penalty(snapshot.MemoryUsedPercent, 70, 25, 1.0)
	score -= penalty(snapshot.SwapUsedPercent, 10, 10, 2.0)
	score -= penalty(snapshot.DiskUsedPercent, 82, 25, 1.5)
	if snapshot.Load1 > float64(runtime.NumCPU()) {
		score -= 10
	}
	if len(snapshot.Warnings) > 0 {
		score -= min(len(snapshot.Warnings)*3, 12)
	}
	if score < 0 {
		score = 0
	}
	switch {
	case score >= 85:
		return score, "healthy"
	case score >= 65:
		return score, "watch"
	case score >= 40:
		return score, "strained"
	default:
		return score, "critical"
	}
}

func deriveHighlights(snapshot *SystemSnapshot) []string {
	highlights := make([]string, 0, 5)
	if snapshot.MemoryUsedPercent >= 80 {
		highlights = append(highlights, fmt.Sprintf("Memory pressure high at %.1f%%", snapshot.MemoryUsedPercent))
	}
	if snapshot.SwapUsedBytes > 0 && (snapshot.SwapUsedPercent >= 5 || snapshot.SwapUsedBytes >= 512*1024*1024) {
		highlights = append(highlights, fmt.Sprintf("Swap in use: %s", domain.HumanBytes(int64(snapshot.SwapUsedBytes))))
	}
	if snapshot.DiskUsedPercent >= 85 {
		highlights = append(highlights, fmt.Sprintf("Disk pressure high at %.1f%% used", snapshot.DiskUsedPercent))
	}
	if snapshot.LoadPerCPU >= 1.0 {
		highlights = append(highlights, fmt.Sprintf("Load is %.2fx core capacity", snapshot.LoadPerCPU))
	}
	if len(snapshot.TopProcesses) > 0 {
		highlights = append(highlights, fmt.Sprintf("Top process: %s using %s RSS", snapshot.TopProcesses[0].Name, domain.HumanBytes(int64(snapshot.TopProcesses[0].MemoryRSSBytes))))
	}
	if snapshot.VirtualizationSystem != "" {
		label := snapshot.VirtualizationSystem
		if snapshot.VirtualizationRole != "" {
			label += " (" + snapshot.VirtualizationRole + ")"
		}
		highlights = append(highlights, "Virtualized: "+label)
	}
	if snapshot.HardwareModel != "" {
		highlights = append(highlights, "Hardware: "+snapshot.HardwareModel)
	}
	if snapshot.CPUModel != "" {
		highlights = append(highlights, "CPU: "+snapshot.CPUModel)
	}
	if snapshot.GPUUsagePercent > 0 {
		gpu := fmt.Sprintf("GPU load: %.0f%%", snapshot.GPUUsagePercent)
		if snapshot.GPURendererPercent > 0 || snapshot.GPUTilerPercent > 0 {
			parts := []string{gpu}
			if snapshot.GPURendererPercent > 0 {
				parts = append(parts, fmt.Sprintf("render %.0f%%", snapshot.GPURendererPercent))
			}
			if snapshot.GPUTilerPercent > 0 {
				parts = append(parts, fmt.Sprintf("tiler %.0f%%", snapshot.GPUTilerPercent))
			}
			gpu = strings.Join(parts, " • ")
		}
		highlights = append(highlights, gpu)
	}
	if snapshot.ThermalState != "" && snapshot.ThermalState != "nominal" {
		highlights = append(highlights, "Thermal state: "+snapshot.ThermalState)
	}
	if snapshot.CPUTempCelsius > 0 {
		highlights = append(highlights, fmt.Sprintf("CPU temp: %.1f°C", snapshot.CPUTempCelsius))
	}
	if snapshot.Battery != nil {
		label := fmt.Sprintf("Battery %.0f%% %s", snapshot.Battery.Percent, snapshot.Battery.State)
		if snapshot.PowerSource != "" {
			label += " on " + snapshot.PowerSource
		}
		highlights = append(highlights, label)
	}
	if snapshot.Proxy != nil && snapshot.Proxy.Enabled {
		target := snapshot.Proxy.HTTP
		if target == "" {
			target = snapshot.Proxy.HTTPS
		}
		if target == "" {
			target = "configured"
		}
		highlights = append(highlights, "Proxy enabled: "+target)
	}
	if n := len(snapshot.Warnings); n > 0 {
		warnWord := map[bool]string{true: "warning", false: "warnings"}[n == 1]
		highlights = append(highlights, fmt.Sprintf("%d live metric %s", n, warnWord))
	}
	if len(highlights) > 12 {
		return highlights[:12]
	}
	return highlights
}

func deriveOperatorAlerts(snapshot *SystemSnapshot) []string {
	alerts := make([]string, 0, 8)
	if snapshot.DiskUsedPercent >= 90 {
		alerts = append(alerts, fmt.Sprintf("disk pressure %.1f%% used", snapshot.DiskUsedPercent))
	}
	if snapshot.MemoryUsedPercent >= 85 {
		alerts = append(alerts, fmt.Sprintf("memory pressure %.1f%% used", snapshot.MemoryUsedPercent))
	}
	if snapshot.SwapUsedPercent >= 20 || snapshot.SwapUsedBytes >= 1<<30 {
		alerts = append(alerts, fmt.Sprintf("swap pressure %s in use", domain.HumanBytes(int64(snapshot.SwapUsedBytes))))
	}
	if snapshot.LoadPerCPU >= 1.0 {
		alerts = append(alerts, fmt.Sprintf("cpu load %.2fx/core", snapshot.LoadPerCPU))
	}
	if snapshot.GPUUsagePercent >= 75 {
		alerts = append(alerts, fmt.Sprintf("gpu load %.0f%%", snapshot.GPUUsagePercent))
	}
	if snapshot.ThermalState != "" && strings.ToLower(snapshot.ThermalState) != "nominal" {
		label := "thermal " + strings.ToLower(snapshot.ThermalState)
		if snapshot.CPUTempCelsius > 0 {
			label += fmt.Sprintf(" %.1f°C", snapshot.CPUTempCelsius)
		}
		alerts = append(alerts, label)
	} else if snapshot.CPUTempCelsius >= 85 {
		alerts = append(alerts, fmt.Sprintf("cpu temp %.1f°C", snapshot.CPUTempCelsius))
	}
	if snapshot.Battery != nil {
		if snapshot.Battery.CapacityPercent > 0 && snapshot.Battery.CapacityPercent < 80 {
			alerts = append(alerts, fmt.Sprintf("battery health %d%%", snapshot.Battery.CapacityPercent))
		}
		if snapshot.Battery.Percent > 0 && snapshot.Battery.Percent <= 15 && strings.ToLower(snapshot.Battery.State) == "discharging" {
			alerts = append(alerts, fmt.Sprintf("battery low %.0f%%", snapshot.Battery.Percent))
		}
	}
	if n := len(snapshot.Warnings); n > 0 {
		warnWord := map[bool]string{true: "warning", false: "warnings"}[n == 1]
		alerts = append(alerts, fmt.Sprintf("%d live metric %s", n, warnWord))
	}
	if len(alerts) > 5 {
		return alerts[:5]
	}
	return alerts
}

func activeNetworkInterfaces(interfaces []netio.InterfaceStat) ([]string, int) {
	active := make([]string, 0, len(interfaces))
	count := 0
	for _, iface := range interfaces {
		if strings.TrimSpace(iface.Name) == "" {
			continue
		}
		count++
		flags := make(map[string]struct{}, len(iface.Flags))
		for _, flag := range iface.Flags {
			flags[strings.ToLower(strings.TrimSpace(flag))] = struct{}{}
		}
		if _, loopback := flags["loopback"]; loopback {
			continue
		}
		if _, up := flags["up"]; !up {
			continue
		}
		active = append(active, iface.Name)
	}
	sort.Strings(active)
	if len(active) > 4 {
		active = active[:4]
	}
	return active, count
}

func penalty(value float64, threshold float64, maxPenalty int, scale float64) int {
	if value <= threshold {
		return 0
	}
	raw := int((value - threshold) / scale)
	if raw > maxPenalty {
		return maxPenalty
	}
	return raw
}

func systemDiskRoot() string {
	if runtime.GOOS == "windows" {
		if drive := os.Getenv("SystemDrive"); drive != "" {
			return drive + `\`
		}
		return `C:\`
	}
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		return string(filepath.Separator)
	}
	return string(filepath.Separator)
}

