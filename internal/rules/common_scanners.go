package rules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func scanImmediateChildren(ctx context.Context, root string, category domain.Category, risk domain.Risk, action domain.Action, source string, curated []string) ([]domain.Finding, []string, error) {
	normalized := domain.NormalizePath(root)
	if normalized == "" {
		return nil, nil, nil
	}
	info, err := os.Lstat(normalized)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var (
		findings []domain.Finding
		warnings []string
	)
	if info.Mode()&fs.ModeSymlink != 0 {
		return nil, []string{normalized + ": symlink skipped"}, nil
	}
	if !info.IsDir() {
		size, newest, err := measureEntry(ctx, normalized, info)
		if err != nil || size == 0 {
			return nil, warnings, err
		}
		return []domain.Finding{newFinding(filepath.Base(normalized), normalized, category, risk, action, size, newest, info.Mode(), source)}, nil, nil
	}
	entries, err := os.ReadDir(normalized)
	if err != nil {
		return nil, nil, err
	}
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}
		child := domain.NormalizePath(filepath.Join(normalized, entry.Name()))
		if shouldSkipCuratedOverlap(child, normalized, curated) || isSIFTOwnedPath(child) {
			continue
		}
		entryInfo, err := os.Lstat(child)
		if err != nil {
			warnings = append(warnings, child+": "+err.Error())
			continue
		}
		if entryInfo.Mode()&fs.ModeSymlink != 0 {
			warnings = append(warnings, child+": symlink skipped")
			continue
		}
		size, newest, err := measureEntry(ctx, child, entryInfo)
		if err != nil {
			warnings = append(warnings, child+": "+err.Error())
			continue
		}
		if size == 0 {
			continue
		}
		findings = append(findings, newFinding(entry.Name(), child, category, risk, action, size, newest, entryInfo.Mode(), source))
	}
	sortFindings(findings)
	if len(findings) > maxFindingsPerRoot {
		findings = findings[:maxFindingsPerRoot]
		warnings = append(warnings, normalized+": capped to largest immediate children")
	}
	return findings, warnings, nil
}

func scanRootAsFinding(ctx context.Context, root string, category domain.Category, risk domain.Risk, action domain.Action, source string) (domain.Finding, []string, error) {
	normalized := domain.NormalizePath(root)
	if normalized == "" {
		return domain.Finding{}, nil, nil
	}
	info, err := os.Lstat(normalized)
	if errors.Is(err, os.ErrNotExist) {
		return domain.Finding{}, nil, nil
	}
	if err != nil {
		return domain.Finding{}, nil, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return domain.Finding{}, []string{normalized + ": symlink skipped"}, nil
	}
	size, newest, err := measureEntry(ctx, normalized, info)
	if err != nil || size == 0 {
		return domain.Finding{}, nil, err
	}
	name := filepath.Base(normalized)
	if source != "" {
		name = source
	}
	return newFinding(name, normalized, category, risk, action, size, newest, info.Mode(), source), nil, nil
}

func measureEntry(ctx context.Context, path string, info fs.FileInfo) (int64, time.Time, error) {
	if info.IsDir() {
		child, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return MeasurePath(child, path)
	}
	return info.Size(), info.ModTime(), nil
}

func newFinding(name, path string, category domain.Category, risk domain.Risk, action domain.Action, size int64, newest time.Time, mode fs.FileMode, source string) domain.Finding {
	return domain.Finding{
		ID:          uuid.NewString(),
		RuleID:      string(category),
		Name:        name,
		Category:    category,
		Path:        path,
		DisplayPath: path,
		Risk:        risk,
		Bytes:       size,
		Action:      action,
		Recovery: domain.RecoveryHint{
			Message:  "Recover from Trash/Recycle Bin if needed.",
			Location: "system trash",
		},
		Status:       domain.StatusPlanned,
		LastModified: newest,
		Fingerprint: domain.Fingerprint{
			Mode:    uint32(mode),
			Size:    size,
			ModTime: newest,
		},
		Source: source,
	}
}

func MeasurePath(ctx context.Context, root string) (int64, time.Time, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, time.Time{}, nil
	}
	if err != nil {
		return 0, time.Time{}, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return 0, info.ModTime(), nil
	}
	if !info.IsDir() {
		size := int64(0)
		if info.Mode().IsRegular() {
			size = info.Size()
		}
		return size, info.ModTime(), nil
	}

	type accumulator struct {
		total  int64
		newest time.Time
	}
	var (
		mu       sync.Mutex
		acc      = accumulator{newest: info.ModTime()}
		firstErr error
		errOnce  sync.Once
	)
	setErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
		})
	}
	updateEntry := func(info fs.FileInfo) {
		mu.Lock()
		defer mu.Unlock()
		if info.ModTime().After(acc.newest) {
			acc.newest = info.ModTime()
		}
		if info.Mode().IsRegular() {
			acc.total += info.Size()
		}
	}

	jobs := make(chan string, max(runtime.NumCPU()*2, 8))
	var dirWG sync.WaitGroup
	var workerWG sync.WaitGroup

	workers := max(min(runtime.NumCPU(), 12), 2)
	worker := func() {
		defer workerWG.Done()
		for dir := range jobs {
			if ctx.Err() != nil {
				setErr(ctx.Err())
				dirWG.Done()
				continue
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					setErr(err)
				}
				dirWG.Done()
				continue
			}
			for _, entry := range entries {
				if ctx.Err() != nil {
					setErr(ctx.Err())
					break
				}
				if entry.Type()&fs.ModeSymlink != 0 {
					continue
				}
				childPath := filepath.Join(dir, entry.Name())
				info, err := entry.Info()
				if err != nil {
					continue
				}
				updateEntry(info)
				if info.IsDir() {
					dirWG.Add(1)
					select {
					case jobs <- childPath:
					default:
						// Channel full — walk inline to prevent deadlock
						// when all workers try to enqueue simultaneously.
						inlineWalk(ctx, childPath, updateEntry)
						dirWG.Done()
					}
				}
			}
			dirWG.Done()
		}
	}

	workerWG.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}
	dirWG.Add(1)
	jobs <- root
	go func() {
		dirWG.Wait()
		close(jobs)
	}()
	workerWG.Wait()

	if errors.Is(firstErr, os.ErrNotExist) {
		return 0, acc.newest, nil
	}
	return acc.total, acc.newest, firstErr
}

// inlineWalk recursively walks a directory on the calling goroutine,
// used as a fallback when the jobs channel is full to prevent deadlock.
func inlineWalk(ctx context.Context, dir string, update func(fs.FileInfo)) {
	if ctx.Err() != nil {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		update(info)
		if info.IsDir() {
			inlineWalk(ctx, filepath.Join(dir, entry.Name()), update)
		}
	}
}

func unique(left, right []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range append(append([]string{}, left...), right...) {
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

func sortFindings(findings []domain.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Bytes == findings[j].Bytes {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Bytes > findings[j].Bytes
	})
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func capWarningsWithSummary(label string, warnings []string, limit int) []string {
	if len(warnings) <= limit || limit <= 0 {
		return warnings
	}
	out := append([]string{}, warnings[:limit]...)
	suppressed := len(warnings) - limit
	warnWord := map[bool]string{true: "warning", false: "warnings"}[suppressed == 1]
	out = append(out, fmt.Sprintf("%s: %d additional %s suppressed", label, suppressed, warnWord))
	return out
}
