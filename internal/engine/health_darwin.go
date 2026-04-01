//go:build darwin

package engine

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	darwinPowerCacheMu   sync.Mutex
	darwinPowerCachedAt  time.Time
	darwinPowerCachedRaw string

	darwinCoreTopoCacheMu  sync.Mutex
	darwinCoreTopoCachedAt time.Time
	darwinCoreTopoPCores   int
	darwinCoresTopoECores  int
)

func init() {
	enrichPlatformSnapshot = enrichDarwinSnapshot
}

func enrichDarwinSnapshot(ctx context.Context, snapshot *SystemSnapshot) {
	if perf, eff, err := darwinCoreTopology(ctx); err == nil && (perf > 0 || eff > 0) {
		snapshot.PerformanceCores = perf
		snapshot.EfficiencyCores = eff
	}
	if battery, powerSource, err := darwinBatterySnapshot(ctx); err == nil {
		snapshot.Battery = battery
		snapshot.PowerSource = powerSource
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "battery: "+err.Error())
	}
	if proxy, err := darwinProxySnapshot(ctx); err == nil && proxy != nil {
		snapshot.Proxy = proxy
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "proxy: "+err.Error())
	}
	if model, err := darwinHardwareModel(ctx); err == nil {
		snapshot.HardwareModel = model
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "hardware_model: "+err.Error())
	}
	if gpuModel, resolution, refreshRate, displayCount, err := darwinDisplayTelemetry(ctx); err == nil {
		snapshot.GPUModel = gpuModel
		snapshot.DisplayResolution = resolution
		snapshot.DisplayRefreshRate = refreshRate
		snapshot.DisplayCount = displayCount
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "display: "+err.Error())
	}
	if gpuUsage, rendererUsage, tilerUsage, err := darwinGPUUsageTelemetry(ctx); err == nil {
		snapshot.GPUUsagePercent = gpuUsage
		snapshot.GPURendererPercent = rendererUsage
		snapshot.GPUTilerPercent = tilerUsage
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "gpu_usage: "+err.Error())
	}
	if bluetoothPowered, bluetoothConnected, bluetoothDevices, err := darwinBluetoothTelemetry(ctx); err == nil {
		snapshot.BluetoothPowered = bluetoothPowered
		snapshot.BluetoothConnected = bluetoothConnected
		snapshot.BluetoothDevices = bluetoothDevices
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "bluetooth: "+err.Error())
	}
	if thermal, err := darwinThermalState(ctx); err == nil {
		snapshot.ThermalState = thermal
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "thermal: "+err.Error())
	}
	if cpuTemp, fanRPM, systemWatts, adapterWatts, batteryWatts, err := darwinPowerTelemetry(ctx); err == nil {
		snapshot.CPUTempCelsius = cpuTemp
		snapshot.FanSpeedRPM = fanRPM
		snapshot.SystemPowerWatts = systemWatts
		snapshot.AdapterPowerWatts = adapterWatts
		snapshot.BatteryPowerWatts = batteryWatts
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "power_metrics: "+err.Error())
	}
}

func darwinBatterySnapshot(ctx context.Context) (*BatterySnapshot, string, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/pmset", "-g", "batt").Output()
	if err != nil {
		return nil, "", err
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		return nil, "", nil
	}
	powerSource := ""
	if idx := strings.Index(lines[0], "'"); idx >= 0 {
		rest := lines[0][idx+1:]
		if end := strings.Index(rest, "'"); end >= 0 {
			powerSource = strings.TrimSpace(rest[:end])
		}
	}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "%") {
			continue
		}
		parts := strings.Split(line, "\t")
		payload := line
		if len(parts) > 1 {
			payload = parts[len(parts)-1]
		}
		fields := splitCSVLike(payload)
		battery := &BatterySnapshot{}
		for _, field := range fields {
			field = strings.TrimSpace(field)
			switch {
			case strings.HasSuffix(field, "%"):
				percent := strings.TrimSuffix(field, "%")
				if value, err := strconv.ParseFloat(percent, 64); err == nil {
					battery.Percent = value
				}
			case strings.Contains(field, "remaining"):
				if minutes := parseRemainingMinutes(field); minutes > 0 {
					battery.RemainingMinutes = minutes
				}
			case field == "charging" || field == "discharging" || field == "charged" || field == "finishing charge":
				battery.State = field
			}
		}
		if battery.State == "" {
			battery.State = "unknown"
		}
		if condition, cycles, capacity, err := darwinPowerHealthInfo(ctx); err == nil {
			battery.Condition = condition
			battery.CycleCount = cycles
			battery.CapacityPercent = capacity
		}
		return battery, powerSource, nil
	}
	return nil, powerSource, nil
}

func darwinPowerHealthInfo(ctx context.Context) (string, int, int, error) {
	raw, err := darwinPowerProfile(ctx)
	if err != nil {
		return "", 0, 0, err
	}
	return parseDarwinPowerInfo(raw)
}

func darwinPowerProfile(ctx context.Context) (string, error) {
	now := time.Now()
	darwinPowerCacheMu.Lock()
	if darwinPowerCachedRaw != "" && now.Sub(darwinPowerCachedAt) < 30*time.Second {
		raw := darwinPowerCachedRaw
		darwinPowerCacheMu.Unlock()
		return raw, nil
	}
	darwinPowerCacheMu.Unlock()

	out, err := exec.CommandContext(ctx, "/usr/sbin/system_profiler", "SPPowerDataType", "-detailLevel", "mini").Output()
	if err != nil {
		return "", err
	}
	raw := string(out)

	darwinPowerCacheMu.Lock()
	darwinPowerCachedRaw = raw
	darwinPowerCachedAt = now
	darwinPowerCacheMu.Unlock()
	return raw, nil
}

func parseDarwinPowerInfo(raw string) (string, int, int, error) {
	condition := ""
	cycleCount := 0
	capacityPercent := 0

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "Condition:"):
			condition = strings.TrimSpace(strings.TrimPrefix(line, "Condition:"))
		case strings.HasPrefix(line, "Cycle Count:"):
			value := strings.TrimSpace(strings.TrimPrefix(line, "Cycle Count:"))
			if parsed, err := strconv.Atoi(value); err == nil {
				cycleCount = parsed
			}
		case strings.HasPrefix(line, "Maximum Capacity:"):
			value := strings.TrimSpace(strings.TrimPrefix(line, "Maximum Capacity:"))
			value = strings.TrimSuffix(value, "%")
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
				capacityPercent = parsed
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", 0, 0, err
	}
	return condition, cycleCount, capacityPercent, nil
}

func darwinPowerTelemetry(ctx context.Context) (float64, int, float64, float64, float64, error) {
	raw, err := darwinPowerProfile(ctx)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	fanRPM := parseDarwinFanSpeed(raw)
	cpuTemp, systemWatts, adapterWatts, batteryWatts := darwinRealtimePowerMetrics(ctx)
	return cpuTemp, fanRPM, systemWatts, adapterWatts, batteryWatts, nil
}

func parseDarwinFanSpeed(raw string) int {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "fan") || !strings.Contains(lower, "speed") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.TrimSpace(parts[1])
		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}
		if parsed, err := strconv.Atoi(fields[0]); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func darwinRealtimePowerMetrics(ctx context.Context) (float64, float64, float64, float64) {
	out, err := exec.CommandContext(ctx, "/usr/sbin/ioreg", "-rn", "AppleSmartBattery").Output()
	if err != nil {
		return 0, 0, 0, 0
	}
	cpuTemp := 0.0
	systemWatts := 0.0
	adapterWatts := 0.0
	batteryWatts := 0.0
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if value, ok := parseDarwinIORegFloat(line, "\"Temperature\" = "); ok && value > 0 {
			cpuTemp = value / 100.0
			continue
		}
		if value, ok := parseDarwinIORegFloat(line, "\"SystemPowerIn\"="); ok {
			if value >= 0 && value < 1000000 {
				systemWatts = value / 1000.0
			}
			continue
		}
		if strings.Contains(line, "\"AdapterDetails\"") && strings.Contains(line, "\"Watts\"=") {
			if value, ok := parseDarwinIORegFloat(line, "\"Watts\"="); ok && value > 0 {
				adapterWatts = value
			}
			continue
		}
		if value, ok := parseDarwinIORegSignedPower(line, "\"BatteryPower\"="); ok {
			if value > -200000 && value < 200000 {
				batteryWatts = value / 1000.0
			}
		}
	}
	return cpuTemp, systemWatts, adapterWatts, batteryWatts
}

func parseDarwinIORegFloat(line string, token string) (float64, bool) {
	_, after, found := strings.Cut(line, token)
	if !found {
		return 0, false
	}
	value := strings.TrimSpace(after)
	for _, sep := range []string{",", "}"} {
		if idx := strings.Index(value, sep); idx >= 0 {
			value = value[:idx]
		}
	}
	value = strings.TrimSpace(value)
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func parseDarwinIORegSignedPower(line string, token string) (float64, bool) {
	_, after, found := strings.Cut(line, token)
	if !found {
		return 0, false
	}
	value := strings.TrimSpace(after)
	for _, sep := range []string{",", "}"} {
		if idx := strings.Index(value, sep); idx >= 0 {
			value = value[:idx]
		}
	}
	value = strings.TrimSpace(value)
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return float64(parsed), true
	}
	if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
		return float64(int64(parsed)), true
	}
	return 0, false
}

func darwinProxySnapshot(ctx context.Context) (*ProxySnapshot, error) {
	proxy := proxyFromEnvironment()
	out, err := exec.CommandContext(ctx, "/usr/sbin/scutil", "--proxy").Output()
	if err != nil {
		if proxy == nil {
			return nil, err
		}
		return proxy, nil
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "HTTPEnable":
			if value == "1" {
				proxy = ensureProxy(proxy)
				proxy.Enabled = true
			}
		case "HTTPProxy":
			proxy = ensureProxy(proxy)
			proxy.HTTP = value
		case "HTTPPort":
			proxy = ensureProxy(proxy)
			if proxy.HTTP != "" && !strings.Contains(proxy.HTTP, ":") {
				proxy.HTTP += ":" + value
			}
		case "HTTPSEnable":
			if value == "1" {
				proxy = ensureProxy(proxy)
				proxy.Enabled = true
			}
		case "HTTPSProxy":
			proxy = ensureProxy(proxy)
			proxy.HTTPS = value
		case "HTTPSPort":
			proxy = ensureProxy(proxy)
			if proxy.HTTPS != "" && !strings.Contains(proxy.HTTPS, ":") {
				proxy.HTTPS += ":" + value
			}
		case "ExceptionsList":
			if proxy == nil {
				proxy = &ProxySnapshot{}
			}
			if proxy.Bypass == "" {
				proxy.Bypass = os.Getenv("NO_PROXY")
			}
		}
	}
	if proxy != nil && (proxy.HTTP != "" || proxy.HTTPS != "" || proxy.Bypass != "") {
		proxy.Enabled = true
	}
	return proxy, scanner.Err()
}

func splitCSVLike(value string) []string {
	parts := strings.Split(value, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseRemainingMinutes(value string) int {
	fields := strings.Fields(value)
	for _, field := range fields {
		if strings.Count(field, ":") != 1 {
			continue
		}
		hm := strings.SplitN(field, ":", 2)
		hours, errH := strconv.Atoi(hm[0])
		minutes, errM := strconv.Atoi(hm[1])
		if errH == nil && errM == nil {
			return hours*60 + minutes
		}
	}
	return 0
}

func proxyFromEnvironment() *ProxySnapshot {
	httpProxy := strings.TrimSpace(firstNonEmpty(os.Getenv("HTTPS_PROXY"), os.Getenv("https_proxy")))
	plainHTTP := strings.TrimSpace(firstNonEmpty(os.Getenv("HTTP_PROXY"), os.Getenv("http_proxy")))
	bypass := strings.TrimSpace(firstNonEmpty(os.Getenv("NO_PROXY"), os.Getenv("no_proxy")))
	if httpProxy == "" && plainHTTP == "" && bypass == "" {
		return nil
	}
	return &ProxySnapshot{
		Enabled: httpProxy != "" || plainHTTP != "" || bypass != "",
		HTTP:    plainHTTP,
		HTTPS:   httpProxy,
		Bypass:  bypass,
	}
}

func ensureProxy(proxy *ProxySnapshot) *ProxySnapshot {
	if proxy != nil {
		return proxy
	}
	return &ProxySnapshot{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// darwinCoreTopology reads the number of Performance and Efficiency cores on
// Apple Silicon Macs via sysctl. The values are cached for 10 minutes since
// they never change at runtime.
func darwinCoreTopology(ctx context.Context) (perf int, eff int, err error) {
	darwinCoreTopoCacheMu.Lock()
	defer darwinCoreTopoCacheMu.Unlock()
	if !darwinCoreTopoCachedAt.IsZero() && time.Since(darwinCoreTopoCachedAt) < 10*time.Minute {
		return darwinCoreTopoPCores, darwinCoresTopoECores, nil
	}
	out, err := exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "hw.perflevel0.physicalcpu").Output()
	if err != nil {
		// Not an Apple Silicon Mac — not an error we surface.
		darwinCoreTopoCachedAt = time.Now()
		return 0, 0, nil
	}
	if p, parseErr := strconv.Atoi(strings.TrimSpace(string(out))); parseErr == nil {
		perf = p
	}
	if out2, err2 := exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "hw.perflevel1.physicalcpu").Output(); err2 == nil {
		if e, parseErr := strconv.Atoi(strings.TrimSpace(string(out2))); parseErr == nil {
			eff = e
		}
	}
	darwinCoreTopoPCores = perf
	darwinCoresTopoECores = eff
	darwinCoreTopoCachedAt = time.Now()
	return perf, eff, nil
}

func darwinHardwareModel(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "hw.model").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func darwinDisplayTelemetry(ctx context.Context) (string, string, string, int, error) {
	out, err := exec.CommandContext(ctx, "/usr/sbin/system_profiler", "SPDisplaysDataType", "-detailLevel", "mini").Output()
	if err != nil {
		return "", "", "", 0, err
	}
	gpuModel, resolution, refreshRate, displayCount := parseDarwinDisplayInfo(string(out))
	return gpuModel, resolution, refreshRate, displayCount, nil
}

func darwinGPUUsageTelemetry(ctx context.Context) (float64, float64, float64, error) {
	var lastErr error
	for _, className := range []string{"AGXAccelerator", "IOAccelerator", "IOGPU"} {
		out, err := exec.CommandContext(ctx, "/usr/sbin/ioreg", "-r", "-c", className, "-l").Output()
		if err != nil {
			lastErr = err
			continue
		}
		device, renderer, tiler, found := parseDarwinGPUUsage(string(out))
		if found {
			return device, renderer, tiler, nil
		}
	}
	if lastErr != nil {
		return 0, 0, 0, lastErr
	}
	return 0, 0, 0, nil
}

func parseDarwinDisplayInfo(raw string) (string, string, string, int) {
	var gpuModel string
	var resolution string
	var refreshRate string
	displayCount := 0
	inDisplays := false

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		indent := len(rawLine) - len(strings.TrimLeft(rawLine, " "))
		switch {
		case line == "Displays:":
			inDisplays = true
			continue
		case gpuModel == "" && (strings.HasPrefix(line, "Chipset Model:") || strings.HasPrefix(line, "GPU:")):
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				gpuModel = strings.TrimSpace(parts[1])
			}
		case inDisplays && indent == 8 && strings.HasSuffix(line, ":"):
			displayCount++
		case inDisplays && resolution == "" && strings.HasPrefix(line, "Resolution:"):
			resolution = normalizeDisplayResolution(strings.TrimSpace(strings.TrimPrefix(line, "Resolution:")))
		case inDisplays && refreshRate == "" && strings.HasPrefix(line, "Refresh Rate:"):
			refreshRate = normalizeDisplayRefreshRate(strings.TrimSpace(strings.TrimPrefix(line, "Refresh Rate:")))
		case inDisplays && strings.HasPrefix(line, "UI Looks like:"):
			uiValue := strings.TrimSpace(strings.TrimPrefix(line, "UI Looks like:"))
			resolution = normalizeDisplayResolution(uiValue)
			if refreshRate == "" {
				if idx := strings.LastIndex(uiValue, "@"); idx >= 0 {
					refreshRate = normalizeDisplayRefreshRate(uiValue[idx+1:])
				}
			}
		case inDisplays && refreshRate == "" && strings.Contains(line, "@") && strings.Contains(strings.ToLower(line), "hz"):
			idx := strings.LastIndex(line, "@")
			if idx >= 0 {
				refreshRate = normalizeDisplayRefreshRate(line[idx+1:])
			}
		}
	}
	return strings.TrimSpace(gpuModel), resolution, refreshRate, displayCount
}

func parseDarwinGPUUsage(raw string) (float64, float64, float64, bool) {
	device := 0.0
	renderer := 0.0
	tiler := 0.0
	found := false
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, "Utilization %") {
			continue
		}
		if value, ok := parseDarwinIORegFloat(line, "\"Device Utilization %\"="); ok {
			if value > device {
				device = value
			}
			found = true
		}
		if value, ok := parseDarwinIORegFloat(line, "\"Renderer Utilization %\"="); ok {
			if value > renderer {
				renderer = value
			}
			found = true
		}
		if value, ok := parseDarwinIORegFloat(line, "\"Tiler Utilization %\"="); ok {
			if value > tiler {
				tiler = value
			}
			found = true
		}
	}
	return device, renderer, tiler, found
}

func normalizeDisplayRefreshRate(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "@")
	value = strings.TrimSuffix(value, ")")
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(value), "hz") {
		number := strings.TrimSpace(value[:len(value)-2])
		if parsed, err := strconv.ParseFloat(number, 64); err == nil {
			formatted := strconv.FormatFloat(parsed, 'f', -1, 64)
			return formatted + "Hz"
		}
	}
	return value
}

func normalizeDisplayResolution(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "("); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if idx := strings.Index(value, "@"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}

func darwinBluetoothTelemetry(ctx context.Context) (bool, int, []BluetoothDeviceSnapshot, error) {
	out, err := exec.CommandContext(ctx, "/usr/sbin/system_profiler", "SPBluetoothDataType", "-detailLevel", "mini").Output()
	if err != nil {
		return false, 0, nil, err
	}
	powered, devices := parseDarwinBluetoothInfo(string(out))
	connected := 0
	for _, device := range devices {
		if device.Connected {
			connected++
		}
	}
	return powered, connected, devices, nil
}

func parseDarwinBluetoothInfo(raw string) (bool, []BluetoothDeviceSnapshot) {
	powered := false
	inConnected := false
	inDisconnected := false
	var current *BluetoothDeviceSnapshot
	devices := make([]BluetoothDeviceSnapshot, 0, 4)

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		indent := len(rawLine) - len(strings.TrimLeft(rawLine, " "))
		switch {
		case strings.HasPrefix(line, "State:"):
			powered = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "State:")), "on")
		case line == "Connected:":
			inConnected = true
			inDisconnected = false
			if current != nil {
				devices = append(devices, *current)
				current = nil
			}
		case line == "Not Connected:":
			inConnected = false
			inDisconnected = true
			if current != nil {
				devices = append(devices, *current)
				current = nil
			}
		case strings.HasSuffix(line, ":") && indent <= 6:
			inConnected = false
			inDisconnected = false
		case (inConnected || inDisconnected) && indent == 10 && strings.HasSuffix(line, ":"):
			if current != nil {
				devices = append(devices, *current)
			}
			current = &BluetoothDeviceSnapshot{
				Name:      strings.TrimSuffix(line, ":"),
				Connected: inConnected,
			}
		case current != nil && strings.HasPrefix(line, "Battery Level:"):
			current.Battery = strings.TrimSpace(strings.TrimPrefix(line, "Battery Level:"))
		}
	}
	if current != nil {
		devices = append(devices, *current)
	}
	return powered, devices
}

func darwinThermalState(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/pmset", "-g", "therm").Output()
	if err != nil {
		return "", err
	}
	thermalLevel := -1
	cpuLimit := 100
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "ThermalLevel"):
			if value, ok := parseKeyValueInt(line); ok {
				thermalLevel = value
			}
		case strings.HasPrefix(line, "CPU_Speed_Limit"):
			if value, ok := parseKeyValueInt(line); ok {
				cpuLimit = value
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	switch {
	case thermalLevel >= 2 || cpuLimit <= 75:
		return "critical", nil
	case thermalLevel == 1 || cpuLimit < 100:
		return "warm", nil
	case thermalLevel == 0 || cpuLimit == 100:
		return "nominal", nil
	default:
		return "", nil
	}
}

func parseKeyValueInt(line string) (int, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, false
	}
	return value, true
}
