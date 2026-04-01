package rules

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/platform"
)

func scanTargets(ctx context.Context, targets []string, adapter platform.Adapter) ([]domain.Finding, []string, error) {
	var findings []domain.Finding
	var warnings []string
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		normalized := domain.NormalizePath(target)
		info, err := os.Lstat(normalized)
		if errors.Is(err, os.ErrNotExist) {
			warnings = append(warnings, normalized+": target not found")
			continue
		}
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			warnings = append(warnings, normalized+": symlink skipped")
			continue
		}
		size, newest, err := MeasurePath(ctx, normalized)
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}
		if size == 0 {
			continue
		}
		findings = append(findings, domain.Finding{
			ID:            uuid.NewString(),
			RuleID:        "target.path",
			Name:          filepath.Base(normalized),
			Category:      domain.CategorySystemClutter,
			Path:          normalized,
			DisplayPath:   normalized,
			Risk:          domain.RiskReview,
			Bytes:         size,
			RequiresAdmin: adapter.IsAdminPath(normalized),
			Action:        domain.ActionTrash,
			Recovery: domain.RecoveryHint{
				Message:  "Recover from Trash/Recycle Bin if needed.",
				Location: "system trash",
			},
			Status:       domain.StatusPlanned,
			LastModified: newest,
			Fingerprint: domain.Fingerprint{
				Mode:    uint32(info.Mode()),
				Size:    size,
				ModTime: newest,
			},
			Source: "User supplied target",
		})
	}
	sortFindings(findings)
	return findings, dedupeStrings(warnings), nil
}
