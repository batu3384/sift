package platform

import (
	"os"
	"strings"
)

const (
	envSiftTestMode        = "SIFT_TEST_MODE"
	envSiftLiveIntegration = "SIFT_LIVE_INTEGRATION"
)

func TestModeEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envSiftTestMode))) {
	case "1", "true", "yes", "ci-safe", "contract":
		return true
	default:
		return false
	}
}

func LiveIntegrationEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envSiftLiveIntegration))) {
	case "1", "true", "yes", "live":
		return true
	default:
		return false
	}
}

func AllowDialogSensitiveActions() bool {
	return !TestModeEnabled() || LiveIntegrationEnabled()
}

func AllowDesktopIntegration() bool {
	if os.Getenv("CI") != "" && !LiveIntegrationEnabled() {
		return false
	}
	return AllowDialogSensitiveActions()
}
