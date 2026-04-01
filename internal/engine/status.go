package engine

import (
	"context"

	"github.com/batuhanyuksel/sift/internal/analyze"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/store"
)

func (s *Service) AnalyzePreviews(paths []string) map[string]domain.DirectoryPreview {
	return analyze.PreviewBatch(paths)
}

func (s *Service) MaintenanceTasks(ctx context.Context) []domain.MaintenanceTask {
	return s.Adapter.MaintenanceTasks(ctx)
}

func (s *Service) RecentScans(ctx context.Context, limit int) ([]store.RecentScan, error) {
	if s.Store == nil {
		return nil, nil
	}
	return s.Store.RecentScans(ctx, limit)
}

func (s *Service) StatusSummary(ctx context.Context, limit int) (store.StatusSummary, error) {
	if s.Store == nil {
		return store.StatusSummary{Platform: s.Adapter.Name()}, nil
	}
	return s.Store.BuildStatusSummary(ctx, s.Adapter.Name(), limit)
}

func (s *Service) StatusReport(ctx context.Context, limit int) (StatusReport, error) {
	summary, err := s.StatusSummary(ctx, limit)
	if err != nil {
		return StatusReport{}, err
	}
	live, err := Snapshot(ctx)
	if err != nil {
		return StatusReport{StatusSummary: summary}, nil
	}
	return StatusReport{
		StatusSummary: summary,
		Live:          live,
	}, nil
}
