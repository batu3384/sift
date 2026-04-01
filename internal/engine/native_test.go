package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitCommandLinePreservesQuotedWindowsPaths(t *testing.T) {
	t.Parallel()
	fields, err := splitCommandLine(`"C:\Program Files\Example\uninstall.exe" /S`)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d (%v)", len(fields), fields)
	}
	if fields[0] != `C:\Program Files\Example\uninstall.exe` {
		t.Fatalf("unexpected executable %q", fields[0])
	}
	if fields[1] != "/S" {
		t.Fatalf("unexpected argument %q", fields[1])
	}
}

func TestParseNativeCommandRejectsShellMetacharacters(t *testing.T) {
	t.Parallel()
	if _, err := parseNativeCommand(`/usr/bin/open /Applications/Example.app ; cleanup-now`); err == nil {
		t.Fatal("expected shell metacharacters to be rejected")
	}
}

func TestParseNativeCommandRejectsUntrustedBareExecutable(t *testing.T) {
	t.Parallel()
	if _, err := parseNativeCommand(`cleanup-helper --silent`); err == nil {
		t.Fatal("expected untrusted bare executable to be rejected")
	}
}

func TestParseNativeCommandAllowsTrustedWindowsInstaller(t *testing.T) {
	t.Parallel()
	command, err := parseNativeCommand(`MsiExec.exe /X{ABC-123}`)
	if err != nil {
		t.Fatal(err)
	}
	if command.Path != "MsiExec.exe" {
		t.Fatalf("unexpected command path %q", command.Path)
	}
	if len(command.Args) != 1 || command.Args[0] != "/X{ABC-123}" {
		t.Fatalf("unexpected command args %+v", command.Args)
	}
}

func TestParseNativeCommandAllowsUnquotedAbsoluteExecutableWithSpaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	executable := filepath.Join(root, "Example App", "uninstall.exe")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	command, err := parseNativeCommand(executable + " /S")
	if err != nil {
		t.Fatal(err)
	}
	if command.Path != executable {
		t.Fatalf("unexpected command path %q", command.Path)
	}
	if len(command.Args) != 1 || command.Args[0] != "/S" {
		t.Fatalf("unexpected command args %+v", command.Args)
	}
}
