//go:build windows

package engine

import "os"

func fileIdentity(os.FileInfo) string {
	return ""
}
