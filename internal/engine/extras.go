package engine

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/batuhanyuksel/sift/internal/domain"
)

var appVersion = "dev"

type managedCommand struct {
	Name string
	Args []string
}

func SetVersion(version string) {
	if strings.TrimSpace(version) == "" {
		return
	}
	appVersion = version
}

func currentVersion() string {
	if strings.TrimSpace(appVersion) == "" {
		return "dev"
	}
	return appVersion
}

func (s *Service) AvailableProtectedFamilies() []domain.ProtectedFamily {
	return availableProtectedFamilies(s.Adapter)
}

func (s *Service) currentExecutable() (string, error) {
	if s.Executable != nil {
		return s.Executable()
	}
	return os.Executable()
}

func (s *Service) resolveExecutable(name string) (string, error) {
	if s.LookPath != nil {
		return s.LookPath(name)
	}
	return exec.LookPath(name)
}

func (s *Service) execCommand(ctx context.Context, path string, args ...string) error {
	if s.RunCommand != nil {
		return s.RunCommand(ctx, path, args...)
	}
	return runManagedProcess(ctx, path, args...)
}
