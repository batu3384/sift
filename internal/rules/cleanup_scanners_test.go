package rules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

// mockAdapter implements platform.Adapter for testing
type mockAdapter struct {
	roots platform.CuratedRoots
}

func (m mockAdapter) Name() string                                      { return "mock" }
func (m mockAdapter) CuratedRoots() platform.CuratedRoots               { return m.roots }
func (m mockAdapter) ProtectedPaths() []string                          { return nil }
func (m mockAdapter) ResolveTargets(in []string) []string               { return in }
func (m mockAdapter) ListApps(ctx context.Context, b bool) ([]domain.AppEntry, error) {
	return nil, nil
}
func (m mockAdapter) DiscoverRemnants(ctx context.Context, e domain.AppEntry) ([]string, []string, error) {
	return nil, nil, nil
}
func (m mockAdapter) MaintenanceTasks(ctx context.Context) []domain.MaintenanceTask { return nil }
func (m mockAdapter) Diagnostics(ctx context.Context) []platform.Diagnostic         { return nil }
func (m mockAdapter) IsAdminPath(s string) bool                                    { return false }
func (m mockAdapter) IsFileInUse(ctx context.Context, s string) bool              { return false }
func (m mockAdapter) IsProcessRunning(s ...string) bool                            { return false }

func TestScanRootEntries(t *testing.T) {
	// Create temp directory with some files
	tmpDir := t.TempDir()
	
	// Create test files
	testFiles := map[string]int64{
		"file1.txt":      100,
		"file2.log":      200,
		"subdir/file3":   300,
	}
	
	for path, size := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, make([]byte, size), 0644); err != nil {
			t.Fatal(err)
		}
	}

	adapter := mockAdapter{
		roots: platform.CuratedRoots{
			Temp: []string{tmpDir},
		},
	}

	ctx := context.Background()
	findings, warnings, err := scanRootEntries(ctx, adapter, []string{tmpDir}, domain.CategoryTempFiles, domain.RiskSafe, domain.ActionTrash, "test")
	
	if err != nil {
		t.Fatalf("scanRootEntries error: %v", err)
	}
	
	// Should find at least the root directory
	if len(findings) == 0 {
		t.Error("expected findings, got none")
	}
	
	// Check warnings
	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}
}

func TestScanRootEntriesEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	
	adapter := mockAdapter{
		roots: platform.CuratedRoots{
			Temp: []string{tmpDir},
		},
	}

	ctx := context.Background()
	findings, _, err := scanRootEntries(ctx, adapter, []string{tmpDir}, domain.CategoryTempFiles, domain.RiskSafe, domain.ActionTrash, "test")
	
	if err != nil {
		t.Fatalf("scanRootEntries error: %v", err)
	}
	
	// Empty dir should return empty findings
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty dir, got %d", len(findings))
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		left     []string
		right    []string
		expected int
	}{
		{"empty", []string{}, []string{}, 0},
		{"single_left", []string{"a", "b"}, []string{}, 2},
		{"single_right", []string{}, []string{"a", "b"}, 2},
		{"no_duplicate", []string{"a", "b"}, []string{"c", "d"}, 4},
		{"with_duplicate", []string{"a", "b"}, []string{"b", "c"}, 3},
		{"all_duplicate", []string{"a"}, []string{"a"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unique(tt.left, tt.right)
			if len(result) != tt.expected {
				t.Errorf("unique(%v, %v) = %d, want %d", tt.left, tt.right, len(result), tt.expected)
			}
		})
	}
}

func TestScanRootEntriesFindsFiles(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create test files
	regularDir := filepath.Join(tmpDir, "regular")
	if err := os.MkdirAll(regularDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regularDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := mockAdapter{
		roots: platform.CuratedRoots{
			Temp: []string{tmpDir},
		},
	}

	ctx := context.Background()
	findings, _, err := scanRootEntries(ctx, adapter, []string{tmpDir}, domain.CategoryTempFiles, domain.RiskSafe, domain.ActionTrash, "test")
	
	if err != nil {
		t.Fatalf("scanRootEntries error: %v", err)
	}
	
	// Should find the regular directory
	found := false
	for _, f := range findings {
		if f.Path == regularDir {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("should include regular paths in findings")
	}
}

func TestPlistStringValues(t *testing.T) {
	// Test plist parsing
	content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Program</key>
	<string>/usr/bin/test</string>
	<key>ProgramArguments</key>
	<array>
		<string>/usr/bin/test</string>
		<string>arg1</string>
	</array>
</dict>
</plist>`

	// Test Program key
	values := plistStringValues(content, "Program")
	if len(values) != 1 || values[0] != "/usr/bin/test" {
		t.Errorf("plistStringValues for Program = %v, want [/usr/bin/test]", values)
	}

	// Test ProgramArguments key
	values = plistStringValues(content, "ProgramArguments")
	if len(values) != 2 {
		t.Errorf("plistStringValues for ProgramArguments = %v, want 2 values", values)
	}

	// Test non-existent key
	values = plistStringValues(content, "NonExistent")
	if len(values) != 0 {
		t.Errorf("plistStringValues for NonExistent = %v, want []", values)
	}
}

func TestStaleLaunchAgentTarget(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantPath string
		wantMsg  string
	}{
		{
			name:     "missing_app",
			content:  `<key>Program</key><string>/Applications/Missing.app/Contents/MacOS/Missing</string>`,
			wantPath: "/Applications/Missing.app/Contents/MacOS/Missing",
			wantMsg:  "missing app/helper target",
		},
		{
			name:     "valid_path",
			content:  `<key>Program</key><string>/bin/bash</string>`,
			wantPath: "",
			wantMsg:  "",
		},
		{
			name:     "system_path",
			content:  `<key>Program</key><string>/usr/bin/launchd</string>`,
			wantPath: "",
			wantMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, msg := staleLaunchAgentTarget([]byte(tt.content))
			if path != tt.wantPath {
				t.Errorf("staleLaunchAgentTarget() path = %q, want %q", path, tt.wantPath)
			}
			if msg != tt.wantMsg {
				t.Errorf("staleLaunchAgentTarget() msg = %q, want %q", msg, tt.wantMsg)
			}
		})
	}
}