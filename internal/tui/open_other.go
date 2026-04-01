//go:build !darwin && !windows

package tui

import "fmt"

func OpenPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	return fmt.Errorf("open/reveal is unsupported on this platform")
}

func RevealPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	return fmt.Errorf("open/reveal is unsupported on this platform")
}
