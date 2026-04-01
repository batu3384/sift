package engine

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	netio "github.com/shirou/gopsutil/v4/net"
)

func TestDeriveHealthHealthy(t *testing.T) {
	t.Parallel()
	score, label := deriveHealth(&SystemSnapshot{
		CPUPercent:        18,
		MemoryUsedPercent: 42,
		DiskUsedPercent:   51,
	})
	if score < 85 {
		t.Fatalf("expected healthy score, got %d", score)
	}
	if label != "healthy" {
		t.Fatalf("expected healthy label, got %s", label)
	}
}

func TestDeriveHealthCritical(t *testing.T) {
	t.Parallel()
	score, label := deriveHealth(&SystemSnapshot{
		CPUPercent:        96,
		MemoryUsedPercent: 94,
		DiskUsedPercent:   97,
		Warnings:          []string{"cpu", "memory", "disk"},
	})
	if score >= 40 {
		t.Fatalf("expected critical score, got %d", score)
	}
	if label != "critical" {
		t.Fatalf("expected critical label, got %s", label)
	}
}

func TestDeriveHighlights(t *testing.T) {
	t.Parallel()
	highlights := deriveHighlights(&SystemSnapshot{
		MemoryUsedPercent:    84,
		SwapUsedPercent:      12,
		SwapUsedBytes:        700 << 20,
		DiskUsedPercent:      91,
		LoadPerCPU:           1.2,
		HardwareModel:        "MacBookPro18,3",
		CPUModel:             "Apple M1 Pro",
		GPUUsagePercent:      57,
		GPURendererPercent:   56,
		GPUTilerPercent:      54,
		ThermalState:         "warm",
		CPUTempCelsius:       61.5,
		Battery:              &BatterySnapshot{Percent: 84, State: "charging"},
		PowerSource:          "ac",
		Proxy:                &ProxySnapshot{Enabled: true, HTTP: "proxy.local:8080"},
		TopProcesses:         []ProcessSnapshot{{Name: "Code", MemoryRSSBytes: 512 << 20}},
		VirtualizationSystem: "docker",
		VirtualizationRole:   "guest",
	})
	if len(highlights) == 0 {
		t.Fatal("expected highlights")
	}
	joined := strings.Join(highlights, " | ")
	for _, needle := range []string{"Memory pressure high", "Swap in use", "Disk pressure high", "Load is", "Hardware: MacBookPro18,3", "CPU: Apple M1 Pro", "GPU load: 57%", "Thermal state: warm", "CPU temp: 61.5°C"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected %q in highlights, got %s", needle, joined)
		}
	}
}

func TestDeriveOperatorAlerts(t *testing.T) {
	t.Parallel()
	alerts := deriveOperatorAlerts(&SystemSnapshot{
		MemoryUsedPercent: 88,
		SwapUsedBytes:     2 << 30,
		SwapUsedPercent:   24,
		DiskUsedPercent:   93,
		LoadPerCPU:        1.25,
		GPUUsagePercent:   84,
		ThermalState:      "warm",
		CPUTempCelsius:    88.4,
		Battery:           &BatterySnapshot{Percent: 12, State: "discharging", CapacityPercent: 78},
		Warnings:          []string{"cpu", "memory"},
	})
	joined := strings.Join(alerts, " | ")
	for _, needle := range []string{
		"disk pressure 93.0% used",
		"memory pressure 88.0% used",
		"swap pressure 2.0 GB in use",
		"cpu load 1.25x/core",
		"gpu load 84%",
	} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected %q in operator alerts, got %s", needle, joined)
		}
	}
	if len(alerts) > 5 {
		t.Fatalf("expected operator alerts to be capped, got %v", alerts)
	}
}

func TestDeriveOperatorAlertsIncludesBatterySignals(t *testing.T) {
	t.Parallel()

	alerts := deriveOperatorAlerts(&SystemSnapshot{
		Battery: &BatterySnapshot{
			Percent:         14,
			State:           "discharging",
			CapacityPercent: 79,
		},
	})
	joined := strings.Join(alerts, " | ")
	for _, needle := range []string{"battery health 79%", "battery low 14%"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected %q in operator alerts, got %s", needle, joined)
		}
	}
}

func TestDeriveHighlightsCapsAtTwelve(t *testing.T) {
	t.Parallel()

	highlights := deriveHighlights(&SystemSnapshot{
		MemoryUsedPercent:    91,
		SwapUsedPercent:      33,
		SwapUsedBytes:        3 << 30,
		DiskUsedPercent:      92,
		LoadPerCPU:           1.4,
		TopProcesses:         []ProcessSnapshot{{Name: "Xcode", MemoryRSSBytes: 2 << 30}},
		VirtualizationSystem: "docker",
		VirtualizationRole:   "guest",
		HardwareModel:        "Mac14,6",
		CPUModel:             "Apple M3 Max",
		GPUUsagePercent:      81,
		GPURendererPercent:   72,
		GPUTilerPercent:      70,
		ThermalState:         "serious",
		CPUTempCelsius:       91.2,
		Battery:              &BatterySnapshot{Percent: 42, State: "charging"},
		PowerSource:          "ac",
		Proxy:                &ProxySnapshot{Enabled: true, HTTPS: "proxy.local:443"},
		Warnings:             []string{"cpu", "gpu", "network"},
	})
	if len(highlights) != 12 {
		t.Fatalf("expected highlights to be capped at 12, got %d (%v)", len(highlights), highlights)
	}
}

func TestSnapshotIncludesArchitecture(t *testing.T) {
	t.Parallel()
	snapshot := &SystemSnapshot{Architecture: runtime.GOARCH}
	if snapshot.Architecture == "" {
		t.Fatal("expected architecture to be set")
	}
}

func TestActiveNetworkInterfacesFiltersLoopbackAndDown(t *testing.T) {
	t.Parallel()
	active, count := activeNetworkInterfaces([]netio.InterfaceStat{
		{Name: "lo0", Flags: []string{"up", "loopback"}},
		{Name: "en0", Flags: []string{"up", "broadcast"}},
		{Name: "utun4", Flags: []string{"up", "pointtopoint"}},
		{Name: "en5", Flags: []string{"broadcast"}},
	})
	if count != 4 {
		t.Fatalf("expected 4 counted interfaces, got %d", count)
	}
	joined := strings.Join(active, ",")
	if joined != "en0,utun4" {
		t.Fatalf("expected active interfaces en0,utun4, got %s", joined)
	}
}

func TestSafeCPUPercentWithContextRecoversFromPanic(t *testing.T) {
	t.Parallel()

	original := cpuPercentWithContext
	cpuPercentWithContext = func(context.Context, time.Duration, bool) ([]float64, error) {
		panic("boom")
	}
	defer func() { cpuPercentWithContext = original }()

	values, err := safeCPUPercentWithContext(context.Background(), true)
	if err == nil {
		t.Fatal("expected panic to convert into error")
	}
	if values != nil {
		t.Fatalf("expected nil values on panic recovery, got %+v", values)
	}
	if !strings.Contains(err.Error(), "cpu sampling panic") {
		t.Fatalf("expected panic error message, got %v", err)
	}
}
