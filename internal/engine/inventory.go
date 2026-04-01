package engine

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

func (s *Service) ListApps(ctx context.Context, allowAdmin bool) ([]domain.AppEntry, error) {
	apps, err := s.Adapter.ListApps(ctx, allowAdmin)
	if err != nil {
		return nil, err
	}
	apps = s.enrichAppInventory(ctx, apps)
	if s.Store != nil {
		_ = s.Store.SaveAppInventory(ctx, s.Adapter.Name(), apps)
	}
	return apps, nil
}

func (s *Service) CachedApps(ctx context.Context) ([]domain.AppEntry, time.Time, error) {
	if s.Store == nil {
		return nil, time.Time{}, nil
	}
	return s.Store.LoadAppInventory(ctx, s.Adapter.Name())
}

func (s *Service) enrichAppInventory(ctx context.Context, apps []domain.AppEntry) []domain.AppEntry {
	if len(apps) == 0 {
		return apps
	}
	out := make([]domain.AppEntry, len(apps))
	copy(out, apps)
	for i := range out {
		if out[i].ApproxBytes == 0 {
			out[i].ApproxBytes = s.approximateAppBytes(ctx, out[i])
		}
		out[i].FamilyMatches = s.appInventoryFamilyMatches(out[i])
		out[i].Sensitive = len(out[i].FamilyMatches) > 0
	}
	return out
}

func (s *Service) appInventoryFamilyMatches(app domain.AppEntry) []string {
	specs := familySpecs(s.Adapter)
	if len(specs) == 0 {
		return nil
	}
	paths := dedupePaths(append([]string{app.BundlePath}, app.SupportPaths...))
	if len(paths) == 0 {
		return nil
	}
	matches := make([]string, 0, 2)
	for _, spec := range specs {
		if spec.id == "safe_cache_domains" {
			continue
		}
		roots := normalizePolicyPaths(spec.roots(s.Adapter))
		for _, path := range paths {
			for _, root := range roots {
				if domain.HasPathPrefix(path, root) {
					matches = append(matches, spec.id)
					goto nextSpec
				}
			}
		}
	nextSpec:
	}
	return dedupeLower(matches)
}

func (s *Service) approximateAppBytes(ctx context.Context, app domain.AppEntry) int64 {
	candidates := dedupePaths(append([]string{app.BundlePath}, app.SupportPaths...))
	if len(candidates) == 0 {
		return 0
	}
	var total int64
	measured := 0
	for _, candidate := range candidates {
		if measured >= 2 {
			break
		}
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		measured++
		if !info.IsDir() {
			if info.Mode().IsRegular() {
				total += info.Size()
			}
			continue
		}
		measureCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		size, _, err := rulesMeasurePath(measureCtx, candidate)
		cancel()
		if size > 0 {
			total += size
		}
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			continue
		}
	}
	return total
}
