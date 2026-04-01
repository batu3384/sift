package engine

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/Bios-Marcel/wastebasket/v2"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/platform"
	"github.com/batuhanyuksel/sift/internal/store"
)

type Service struct {
	Adapter             platform.Adapter
	Config              config.Config
	Store               *store.Store
	LookPath            func(string) (string, error)
	Executable          func() (string, error)
	RunCommand          func(context.Context, string, ...string) error
	MoveToTrash         func(string) error
	ReadFile            func(string) ([]byte, error)
	WriteFile           func(string, []byte, os.FileMode) error
	TouchIDPAMPath      string
	TouchIDLocalPAMPath string
}

// CategoryScanProgress callback is called after each category scan with the rule name and findings
type CategoryScanProgress func(ruleID string, ruleName string, itemsFound int, bytesFound int64)

// CategoryScanResult holds the result of scanning a single category
type CategoryScanResult struct {
	RuleID    string
	RuleName  string
	Items     []domain.Finding
	Bytes     int64
	Warnings  []string
}

type ScanOptions struct {
	Command    string
	Profile    string
	Targets    []string
	RuleIDs    []string
	DryRun     bool
	AllowAdmin bool
	// CategoryCallback is called after each category scan with progress info
	CategoryCallback CategoryScanProgress
}

type StatusReport struct {
	store.StatusSummary
	Live *SystemSnapshot `json:"live,omitempty"`
}

type ExecuteOptions struct {
	Permanent       bool
	NativeUninstall bool
}

func NewService(cfg config.Config, st *store.Store) *Service {
	return &Service{
		Adapter:    platform.Current(),
		Config:     cfg,
		Store:      st,
		LookPath:   exec.LookPath,
		Executable: os.Executable,
		RunCommand: runManagedProcess,
		MoveToTrash: func(path string) error {
			return wastebasket.Trash(path)
		},
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
	}
}

// IsProcessRunning checks if any of the given processes are running
func (s *Service) IsProcessRunning(names ...string) bool {
	return s.Adapter.IsProcessRunning(names...)
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range in {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func trimmedValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
