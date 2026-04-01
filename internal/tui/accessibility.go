package tui

import (
	"os"
	"strings"
)

const envSIFTReducedMotion = "SIFT_REDUCED_MOTION"

func ReducedMotionEnabled() bool {
	return truthyEnv(os.Getenv(envSIFTReducedMotion))
}

func truthyEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
