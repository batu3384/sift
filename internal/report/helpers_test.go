package report

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func TestRedactStringReplacesHomePathWithTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	value := filepath.Join(home, "Library", "Application Support", "sift")
	redacted := redactString(value)
	if redacted == value {
		t.Fatalf("expected redacted string to change, got %q", redacted)
	}
	if !strings.Contains(redacted, "~") {
		t.Fatalf("expected redacted string to include tilde, got %q", redacted)
	}
}

func TestRedactRawJSONRedactsNestedStrings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	raw := json.RawMessage(`{"path":"` + filepath.ToSlash(filepath.Join(home, "Secrets", "token")) + `","nested":["` + filepath.ToSlash(filepath.Join(home, "Projects", "keep")) + `"]}`)
	redacted := redactRawJSON(raw)
	if bytes.Contains(redacted, []byte(filepath.ToSlash(home))) {
		t.Fatalf("expected home path to be redacted from raw json, got %s", redacted)
	}
}

func TestRedactStatusSummaryRedactsNestedPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	summary := &store.StatusSummary{
		AuditLogPath: filepath.Join(home, "Library", "Caches", "sift", "audit.ndjson"),
		LastScan: &store.RecentScan{
			Warnings: []string{filepath.Join(home, "warning", "path")},
		},
		LastExecution: &store.ExecutionSummary{
			Warnings:         []string{filepath.Join(home, "execution", "warning")},
			FollowUpCommands: []string{`sift uninstall "` + filepath.Join(home, "follow-up", "target") + `"`},
		},
	}

	redacted := redactStatusSummary(summary)
	if redacted == nil {
		t.Fatal("expected redacted summary")
	}
	if strings.Contains(redacted.AuditLogPath, home) {
		t.Fatalf("expected audit log path to be redacted, got %q", redacted.AuditLogPath)
	}
	if strings.Contains(redacted.LastScan.Warnings[0], home) {
		t.Fatalf("expected last scan warning to be redacted, got %q", redacted.LastScan.Warnings[0])
	}
	if strings.Contains(redacted.LastExecution.FollowUpCommands[0], home) {
		t.Fatalf("expected follow-up command to be redacted, got %q", redacted.LastExecution.FollowUpCommands[0])
	}
}

func TestWriteJSONProducesIndentedPayload(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if err := writeJSON(writer, "sample.json", map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("expected one zip entry, got %d", len(reader.File))
	}
	rc, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	payload, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(payload, []byte("\n  \"status\": \"ok\"\n")) {
		t.Fatalf("expected indented json payload, got %s", payload)
	}
}

func TestRedactDiagnosticsRedactsMessages(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	diagnostics := []platform.Diagnostic{{
		Name:    "app_support",
		Status:  "warn",
		Message: filepath.Join(home, "Library", "Application Support", "sift"),
	}}
	redacted := redactDiagnostics(diagnostics)
	if strings.Contains(redacted[0].Message, home) {
		t.Fatalf("expected diagnostic message to be redacted, got %q", redacted[0].Message)
	}
}

func TestRedactConfigRedactsProtectedAndPurgePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := config.Default()
	cfg.ProtectedPaths = []string{filepath.Join(home, "Projects", "keep")}
	cfg.PurgeSearchPaths = []string{filepath.Join(home, "Projects")}

	redacted := redactConfig(cfg)
	if strings.Contains(redacted.ProtectedPaths[0], home) || strings.Contains(redacted.PurgeSearchPaths[0], home) {
		t.Fatalf("expected config paths to be redacted, got %+v", redacted)
	}
}

func TestRedactPlanRedactsNativeCommandAndPolicyPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	plan := domain.ExecutionPlan{
		Targets: []string{filepath.Join(home, "Downloads", "installer.dmg")},
		Policy: domain.ProtectionPolicy{
			ProtectedPaths: []string{filepath.Join(home, "Secrets")},
			AllowedRoots:   []string{filepath.Join(home, "Library", "Caches")},
		},
		Items: []domain.Finding{{
			Path:          filepath.Join(home, "Library", "Caches", "Example"),
			DisplayPath:   filepath.Join(home, "Library", "Caches", "Example"),
			NativeCommand: filepath.Join(home, "Applications", "Example.app", "Contents", "MacOS", "uninstall"),
		}},
	}

	redacted := redactPlan(plan)
	if strings.Contains(redacted.Targets[0], home) || strings.Contains(redacted.Policy.ProtectedPaths[0], home) || strings.Contains(redacted.Items[0].NativeCommand, home) {
		t.Fatalf("expected plan paths to be redacted, got %+v", redacted)
	}
}
