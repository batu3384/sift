//go:build darwin

package engine

import (
	"strings"
	"testing"
)

func TestParseDarwinDisplayInfo(t *testing.T) {
	t.Parallel()

	raw := `
Graphics/Displays:

    Apple M3 Pro:

      Chipset Model: Apple M3 Pro
      Type: GPU
      Displays:
        Color LCD:
          Resolution: 3024 x 1964 Retina
          UI Looks like: 1512 x 982 @ 120.00Hz

`

	gpuModel, resolution, refreshRate, displayCount := parseDarwinDisplayInfo(raw)
	if gpuModel != "Apple M3 Pro" {
		t.Fatalf("expected gpu model, got %q", gpuModel)
	}
	if resolution != "1512 x 982" {
		t.Fatalf("expected normalized resolution, got %q", resolution)
	}
	if refreshRate != "120Hz" {
		t.Fatalf("expected normalized refresh rate, got %q", refreshRate)
	}
	if displayCount != 1 {
		t.Fatalf("expected display count, got %d", displayCount)
	}
}

func TestNormalizeDisplayRefreshRatePreservesFractionalRates(t *testing.T) {
	t.Parallel()

	if got := normalizeDisplayRefreshRate("59.94Hz"); got != "59.94Hz" {
		t.Fatalf("expected 59.94Hz, got %q", got)
	}
}

func TestParseDarwinBluetoothInfo(t *testing.T) {
	t.Parallel()

	raw := `
Bluetooth:

      Bluetooth Controller:
          State: On
      Connected:
          Batuhan (AirPods Pro):
              RSSI: -56
          Keyboard:
              RSSI: -40
      Not Connected:
          Mouse:
              RSSI: -70
`

	powered, devices := parseDarwinBluetoothInfo(raw)
	if !powered {
		t.Fatal("expected bluetooth to be powered on")
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 parsed bluetooth devices, got %+v", devices)
	}
	if !devices[0].Connected || devices[0].Name != "Batuhan (AirPods Pro)" {
		t.Fatalf("expected first connected device, got %+v", devices[0])
	}
	if !devices[1].Connected || devices[1].Name != "Keyboard" {
		t.Fatalf("expected second connected device, got %+v", devices[1])
	}
	if devices[2].Connected || devices[2].Name != "Mouse" {
		t.Fatalf("expected disconnected mouse entry, got %+v", devices[2])
	}
}

func TestParseDarwinPowerInfo(t *testing.T) {
	t.Parallel()

	raw := `
Power:

    Battery Information:

      Model Information:
          Serial Number: D862345A123456
      Charge Information:
          State of Charge (%): 78
      Health Information:
          Cycle Count: 142
          Condition: Normal
          Maximum Capacity: 96%
`

	condition, cycles, capacity, err := parseDarwinPowerInfo(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if condition != "Normal" {
		t.Fatalf("expected condition Normal, got %q", condition)
	}
	if cycles != 142 {
		t.Fatalf("expected cycle count 142, got %d", cycles)
	}
	if capacity != 96 {
		t.Fatalf("expected max capacity 96, got %d", capacity)
	}
}

func TestParseDarwinFanSpeed(t *testing.T) {
	t.Parallel()

	raw := `
Power:
    Current Fan Speed: 2384
`

	if rpm := parseDarwinFanSpeed(raw); rpm != 2384 {
		t.Fatalf("expected fan speed 2384, got %d", rpm)
	}
}

func TestParseDarwinIORegFloat(t *testing.T) {
	t.Parallel()

	line := `"SystemPowerIn"=42000,`
	value, ok := parseDarwinIORegFloat(line, `"SystemPowerIn"=`)
	if !ok {
		t.Fatal("expected parser to match ioreg line")
	}
	if value != 42000 {
		t.Fatalf("expected 42000, got %f", value)
	}

	tempValue, ok := parseDarwinIORegFloat(`"Temperature" = 3055`, `"Temperature" = `)
	if !ok {
		t.Fatal("expected parser to match temperature line")
	}
	if tempValue != 3055 {
		t.Fatalf("expected 3055, got %f", tempValue)
	}
}

func TestParseDarwinIORegSignedPower(t *testing.T) {
	t.Parallel()

	value, ok := parseDarwinIORegSignedPower(`"BatteryPower"=18000,`, `"BatteryPower"=`)
	if !ok {
		t.Fatal("expected signed battery power parser to match positive line")
	}
	if value != 18000 {
		t.Fatalf("expected 18000, got %f", value)
	}

	negativeValue, ok := parseDarwinIORegSignedPower(`"BatteryPower"=18446744073709533616,`, `"BatteryPower"=`)
	if !ok {
		t.Fatal("expected signed battery power parser to match two's complement line")
	}
	if negativeValue != -18000 {
		t.Fatalf("expected -18000, got %f", negativeValue)
	}
}

func TestParseDarwinGPUUsage(t *testing.T) {
	t.Parallel()

	raw := `
+-o AGXAcceleratorG16G  <class AGXAcceleratorG16G, id 0x10000046b, registered>
  | {
  |   "PerformanceStatistics" = {"Tiler Utilization %"=57,"Renderer Utilization %"=56,"Device Utilization %"=61}
  | }
`

	device, renderer, tiler, found := parseDarwinGPUUsage(raw)
	if !found {
		t.Fatal("expected gpu usage parser to find utilization values")
	}
	if device != 61 || renderer != 56 || tiler != 57 {
		t.Fatalf("unexpected gpu usage values: device=%f renderer=%f tiler=%f", device, renderer, tiler)
	}
}

func TestSplitCSVLike(t *testing.T) {
	t.Parallel()

	got := splitCSVLike(" charging; 2:15 remaining; present: true ; ")
	if len(got) != 3 {
		t.Fatalf("expected 3 split fields, got %v", got)
	}
	if got[0] != "charging" || got[1] != "2:15 remaining" || got[2] != "present: true" {
		t.Fatalf("unexpected split fields: %v", got)
	}
}

func TestParseRemainingMinutes(t *testing.T) {
	t.Parallel()

	if got := parseRemainingMinutes("2:15 remaining present: true"); got != 135 {
		t.Fatalf("expected 135 minutes, got %d", got)
	}
	if got := parseRemainingMinutes("no estimate"); got != 0 {
		t.Fatalf("expected 0 minutes for missing estimate, got %d", got)
	}
}

func TestProxyFromEnvironmentPrefersSecureVariants(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "https://secure-proxy:8443")
	t.Setenv("https_proxy", "https://ignored-secure-proxy:9443")
	t.Setenv("HTTP_PROXY", "http://plain-proxy:8080")
	t.Setenv("http_proxy", "http://ignored-plain-proxy:9080")
	t.Setenv("NO_PROXY", "localhost,127.0.0.1")
	t.Setenv("no_proxy", "example.com")

	proxy := proxyFromEnvironment()
	if proxy == nil {
		t.Fatal("expected proxy to be built from environment")
	}
	if proxy.HTTPS != "https://secure-proxy:8443" {
		t.Fatalf("expected HTTPS proxy to prefer uppercase secure variant, got %+v", proxy)
	}
	if proxy.HTTP != "http://plain-proxy:8080" {
		t.Fatalf("expected HTTP proxy to prefer uppercase plain variant, got %+v", proxy)
	}
	if proxy.Bypass != "localhost,127.0.0.1" || !proxy.Enabled {
		t.Fatalf("unexpected proxy bypass/enabled state: %+v", proxy)
	}
}

func TestNormalizeDisplayResolutionAndRefreshRate(t *testing.T) {
	t.Parallel()

	if got := normalizeDisplayResolution("1512 x 982 @ 120.00Hz (Retina)"); got != "1512 x 982" {
		t.Fatalf("expected stripped display resolution, got %q", got)
	}
	if got := normalizeDisplayRefreshRate("@120.00Hz)"); got != "120Hz" {
		t.Fatalf("expected normalized refresh rate, got %q", got)
	}
}

func TestParseDarwinBluetoothInfoIncludesBatteryLevel(t *testing.T) {
	t.Parallel()

	raw := `
Bluetooth:

      Bluetooth Controller:
          State: On
      Connected:
          Batuhan Keyboard:
              Battery Level: 64%
              RSSI: -40
`

	powered, devices := parseDarwinBluetoothInfo(raw)
	if !powered {
		t.Fatal("expected bluetooth to be powered on")
	}
	if len(devices) != 1 || devices[0].Battery != "64%" {
		t.Fatalf("expected connected device battery level, got %+v", devices)
	}
}

func TestEnsureProxyReusesExistingInstance(t *testing.T) {
	t.Parallel()

	proxy := &ProxySnapshot{Enabled: true}
	if got := ensureProxy(proxy); got != proxy {
		t.Fatal("expected ensureProxy to return the existing pointer")
	}
	if fresh := ensureProxy(nil); fresh == nil || fresh == proxy {
		t.Fatalf("expected ensureProxy to allocate a fresh proxy, got %+v", fresh)
	}
}

func TestParseDarwinDisplayInfoCountsMultipleDisplays(t *testing.T) {
	t.Parallel()

	raw := `
Graphics/Displays:

    Apple M4:
      Chipset Model: Apple M4
      Displays:
        Studio Display:
          Resolution: 5120 x 2880
        LG UltraFine:
          UI Looks like: 2560 x 1440 @ 59.94Hz
`

	gpuModel, resolution, refreshRate, displayCount := parseDarwinDisplayInfo(raw)
	if gpuModel != "Apple M4" {
		t.Fatalf("expected gpu model, got %q", gpuModel)
	}
	if displayCount != 2 {
		t.Fatalf("expected two displays, got %d", displayCount)
	}
	if resolution != "5120 x 2880" && !strings.Contains(resolution, "2560 x 1440") {
		t.Fatalf("expected a parsed resolution, got %q", resolution)
	}
	if refreshRate != "" && refreshRate != "59.94Hz" {
		t.Fatalf("unexpected refresh rate %q", refreshRate)
	}
}
