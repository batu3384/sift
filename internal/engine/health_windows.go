//go:build windows

package engine

import (
	"context"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	Reserved1           byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

var (
	kernel32                = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemPowerState = kernel32.NewProc("GetSystemPowerStatus")
)

func init() {
	enrichPlatformSnapshot = enrichWindowsSnapshot
}

func enrichWindowsSnapshot(_ context.Context, snapshot *SystemSnapshot) {
	if battery, powerSource, err := windowsBatterySnapshot(); err == nil {
		snapshot.Battery = battery
		snapshot.PowerSource = powerSource
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "battery: "+err.Error())
	}
	if proxy, err := windowsProxySnapshot(); err == nil && proxy != nil {
		snapshot.Proxy = proxy
	} else if err != nil {
		snapshot.Warnings = append(snapshot.Warnings, "proxy: "+err.Error())
	}
}

func windowsBatterySnapshot() (*BatterySnapshot, string, error) {
	var status systemPowerStatus
	ret, _, err := procGetSystemPowerState.Call(uintptr(unsafe.Pointer(&status)))
	if ret == 0 {
		return nil, "", err
	}
	powerSource := "unknown"
	switch status.ACLineStatus {
	case 0:
		powerSource = "battery"
	case 1:
		powerSource = "ac"
	}
	if status.BatteryLifePercent == 255 {
		return nil, powerSource, nil
	}
	state := "unknown"
	switch {
	case status.ACLineStatus == 1 && status.BatteryLifePercent >= 100:
		state = "charged"
	case status.ACLineStatus == 1:
		state = "charging"
	case status.ACLineStatus == 0:
		state = "discharging"
	}
	battery := &BatterySnapshot{
		Percent: float64(status.BatteryLifePercent),
		State:   state,
	}
	if status.BatteryLifeTime != 0 && status.BatteryLifeTime != 0xFFFFFFFF {
		battery.RemainingMinutes = int(status.BatteryLifeTime / 60)
	}
	return battery, powerSource, nil
}

func windowsProxySnapshot() (*ProxySnapshot, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.READ)
	if err != nil {
		return nil, err
	}
	defer key.Close()
	enabledValue, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		return nil, nil
	}
	server, _, _ := key.GetStringValue("ProxyServer")
	override, _, _ := key.GetStringValue("ProxyOverride")
	httpProxy, httpsProxy := parseWindowsProxyServer(server)
	if enabledValue == 0 && httpProxy == "" && httpsProxy == "" && strings.TrimSpace(override) == "" {
		return nil, nil
	}
	return &ProxySnapshot{
		Enabled: enabledValue != 0,
		HTTP:    httpProxy,
		HTTPS:   httpsProxy,
		Bypass:  strings.TrimSpace(override),
	}, nil
}

func parseWindowsProxyServer(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	httpProxy := ""
	httpsProxy := ""
	if !strings.Contains(value, "=") {
		return value, value
	}
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		target := strings.TrimSpace(kv[1])
		switch key {
		case "http":
			httpProxy = target
		case "https":
			httpsProxy = target
		}
	}
	if httpProxy == "" {
		httpProxy = httpsProxy
	}
	if httpsProxy == "" {
		httpsProxy = httpProxy
	}
	return httpProxy, httpsProxy
}
