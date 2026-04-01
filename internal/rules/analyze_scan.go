package rules

import (
	"bytes"
	"container/heap"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/batuhanyuksel/sift/internal/analyze"
	"github.com/batuhanyuksel/sift/internal/domain"
)

func scanDiskUsage(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
	analyzeHooksMu.RLock()
	defer analyzeHooksMu.RUnlock()
	return cachedAnalyzeScan(ctx, "disk_usage", targets, analyzeDiskUsageLoader)
}

func scanDiskUsageFresh(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
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
			warnings = append(warnings, normalized+": symlink target skipped")
			continue
		}
		if !info.IsDir() {
			findings = append(findings, domain.Finding{
				ID:           uuid.NewString(),
				RuleID:       "analyze.disk_usage",
				Name:         filepath.Base(normalized),
				Category:     domain.CategoryDiskUsage,
				Path:         normalized,
				DisplayPath:  normalized,
				Risk:         domain.RiskReview,
				Bytes:        info.Size(),
				Action:       domain.ActionAdvisory,
				Recovery:     domain.RecoveryHint{Message: "Analyze is read-only."},
				Status:       domain.StatusAdvisory,
				LastModified: info.ModTime(),
				Fingerprint: domain.Fingerprint{
					Mode:    uint32(info.Mode()),
					Size:    info.Size(),
					ModTime: info.ModTime(),
				},
				Source: "Target file",
			})
			continue
		}
		entries, err := os.ReadDir(normalized)
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}
		childFindings, childWarnings, err := scanDiskUsageEntries(ctx, normalized, entries)
		if err != nil {
			return findings, warnings, err
		}
		findings = append(findings, childFindings...)
		warnings = append(warnings, childWarnings...)
	}
	slices.SortFunc(findings, func(a, b domain.Finding) int {
		if a.Bytes == b.Bytes {
			return strings.Compare(a.Path, b.Path)
		}
		if a.Bytes > b.Bytes {
			return -1
		}
		return 1
	})
	if len(findings) > maxAnalyzeDiskUsage {
		findings = findings[:maxAnalyzeDiskUsage]
		warnings = append(warnings, "disk usage analysis capped to top results")
	}
	return findings, capWarningsWithSummary("analyze", dedupeStrings(warnings), 10), nil
}

func scanDiskUsageEntries(ctx context.Context, root string, entries []fs.DirEntry) ([]domain.Finding, []string, error) {
	type job struct {
		entry fs.DirEntry
		path  string
	}
	type result struct {
		finding  *domain.Finding
		warning  string
		fatalErr error
	}

	workers := min(max(runtime.NumCPU(), 2), max(len(entries), 1))
	jobs := make(chan job, workers)
	results := make(chan result, len(entries))
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for task := range jobs {
			select {
			case <-ctx.Done():
				results <- result{fatalErr: ctx.Err()}
				return
			default:
			}
			entryInfo, err := task.entry.Info()
			if err != nil {
				results <- result{warning: task.path + ": " + err.Error()}
				continue
			}
			size := entryInfo.Size()
			modified := entryInfo.ModTime()
			findingPath := task.path
			findingName := task.entry.Name()
			if entryInfo.IsDir() {
				findingPath, findingName = foldAnalyzeDirectory(task.path)
				size, modified, err = MeasurePath(ctx, findingPath)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						results <- result{fatalErr: err}
						return
					}
					results <- result{warning: task.path + ": " + err.Error()}
					continue
				}
			}
			finding := domain.Finding{
				ID:           uuid.NewString(),
				RuleID:       "analyze.disk_usage",
				Name:         findingName,
				Category:     domain.CategoryDiskUsage,
				Path:         findingPath,
				DisplayPath:  findingPath,
				Risk:         domain.RiskReview,
				Bytes:        size,
				Action:       domain.ActionAdvisory,
				Recovery:     domain.RecoveryHint{Message: "Analyze is read-only."},
				Status:       domain.StatusAdvisory,
				LastModified: modified,
				Fingerprint: domain.Fingerprint{
					Mode:    uint32(entryInfo.Mode()),
					Size:    size,
					ModTime: modified,
				},
				Source: analyzeDiskUsageSource(root, task.path, findingPath),
			}
			results <- result{finding: &finding}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}
	for _, entry := range entries {
		child := domain.NormalizePath(filepath.Join(root, entry.Name()))
		if strings.HasPrefix(child, filepath.Join(root, ".Trash")) {
			continue
		}
		jobs <- job{entry: entry, path: child}
	}
	close(jobs)
	go func() {
		wg.Wait()
		close(results)
	}()

	var (
		findings []domain.Finding
		warnings []string
		fatalErr error
	)
	for item := range results {
		if item.fatalErr != nil {
			if fatalErr == nil {
				fatalErr = item.fatalErr
			}
			continue
		}
		if item.warning != "" {
			warnings = append(warnings, item.warning)
			continue
		}
		if item.finding != nil {
			findings = append(findings, *item.finding)
		}
	}
	return findings, warnings, fatalErr
}

func foldAnalyzeDirectory(path string) (string, string) {
	current := domain.NormalizePath(path)
	parts := []string{filepath.Base(current)}
	for depth := 0; depth < analyzeFoldedDirMaxDepth; depth++ {
		entries, err := os.ReadDir(current)
		if err != nil {
			break
		}
		nextName := ""
		fileSeen := false
		dirCount := 0
		for _, entry := range entries {
			if entry.Type()&fs.ModeSymlink != 0 {
				fileSeen = true
				break
			}
			if entry.IsDir() {
				dirCount++
				nextName = entry.Name()
				continue
			}
			if _, ok := analyzeFoldIgnoredFiles[entry.Name()]; ok {
				continue
			}
			fileSeen = true
			break
		}
		if fileSeen || dirCount != 1 {
			break
		}
		current = domain.NormalizePath(filepath.Join(current, nextName))
		parts = append(parts, nextName)
	}
	return current, strings.Join(parts, string(filepath.Separator))
}

func analyzeDiskUsageSource(root string, originalPath string, foldedPath string) string {
	source := "Immediate child of " + root
	if originalPath != foldedPath {
		source += " • folded"
	}
	return source
}

func scanLargeFiles(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
	analyzeHooksMu.RLock()
	defer analyzeHooksMu.RUnlock()
	return cachedAnalyzeScan(ctx, "large_files", targets, analyzeLargeFilesLoader)
}

func scanLargeFilesFresh(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
	filesHeap := &largeAnalyzeFindingHeap{}
	heap.Init(filesHeap)
	totalMatches := 0
	var warnings []string
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return nil, warnings, ctx.Err()
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
			warnings = append(warnings, normalized+": symlink target skipped")
			continue
		}
		if !info.IsDir() {
			if info.Mode().IsRegular() && info.Size() >= analyzeLargeFileMinBytes {
				totalMatches++
				pushLargeAnalyzeFinding(filesHeap, largeFileFinding(normalized, info, normalized), maxAnalyzeLargeFiles)
			}
			continue
		}
		seeded := map[string]struct{}{}
		spotlightPaths, err := spotlightLargeFileSearch(ctx, normalized, analyzeLargeFileMinBytes)
		if err != nil {
			warnings = append(warnings, normalized+": spotlight assist unavailable: "+err.Error())
		}
		for _, candidate := range spotlightPaths {
			if _, ok := seeded[candidate]; ok {
				continue
			}
			if !domain.HasPathPrefix(candidate, normalized) {
				continue
			}
			candidateInfo, err := os.Lstat(candidate)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				warnings = append(warnings, candidate+": "+err.Error())
				continue
			}
			if candidateInfo.Mode()&fs.ModeSymlink != 0 || !candidateInfo.Mode().IsRegular() || candidateInfo.Size() < analyzeLargeFileMinBytes {
				continue
			}
			seeded[candidate] = struct{}{}
			totalMatches++
			pushLargeAnalyzeFinding(filesHeap, largeFileFinding(candidate, candidateInfo, normalized), maxAnalyzeLargeFiles)
		}
		err = filepath.WalkDir(normalized, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				warnings = append(warnings, path+": "+walkErr.Error())
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				base := filepath.Base(path)
				if base == ".Trash" || strings.EqualFold(base, "$Recycle.Bin") {
					return filepath.SkipDir
				}
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				warnings = append(warnings, path+": "+err.Error())
				return nil
			}
			if !info.Mode().IsRegular() || info.Size() < analyzeLargeFileMinBytes {
				return nil
			}
			if _, ok := seeded[path]; ok {
				return nil
			}
			totalMatches++
			pushLargeAnalyzeFinding(filesHeap, largeFileFinding(path, info, normalized), maxAnalyzeLargeFiles)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			warnings = append(warnings, normalized+": "+err.Error())
		}
	}
	findings := filesHeap.sorted()
	if totalMatches > maxAnalyzeLargeFiles {
		warnings = append(warnings, "large file analysis capped to top results")
	}
	return findings, capWarningsWithSummary("analyze", dedupeStrings(warnings), 10), nil
}

func cachedAnalyzeScan(ctx context.Context, kind string, targets []string, loader func(context.Context, []string) ([]domain.Finding, []string, error)) ([]domain.Finding, []string, error) {
	return analyze.CachedScan(ctx, kind, targets, loader)
}

func AnalyzePreviews(paths []string) map[string]domain.DirectoryPreview {
	return analyze.PreviewBatch(paths)
}

func discoverSpotlightLargeFiles(ctx context.Context, root string, minBytes int64) ([]string, error) {
	if runtime.GOOS != "darwin" || strings.TrimSpace(root) == "" {
		return nil, nil
	}
	const mdfindPath = "/usr/bin/mdfind"
	if _, err := os.Stat(mdfindPath); err != nil {
		return nil, nil
	}
	query := fmt.Sprintf("kMDItemFSSize >= %d", minBytes)
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return nil, err
	}
	defer stdin.Close()
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer readPipe.Close()
	proc, err := os.StartProcess(mdfindPath, []string{"mdfind", "-0", "-onlyin", root, query}, &os.ProcAttr{
		Files: []*os.File{stdin, writePipe, writePipe},
	})
	if err != nil {
		_ = writePipe.Close()
		return nil, err
	}
	_ = writePipe.Close()
	done := make(chan struct {
		out []byte
		err error
	}, 1)
	go func() {
		output, readErr := io.ReadAll(readPipe)
		done <- struct {
			out []byte
			err error
		}{out: output, err: readErr}
	}()
	select {
	case <-ctx.Done():
		_ = proc.Kill()
		_, _ = proc.Wait()
		return nil, ctx.Err()
	case result := <-done:
		state, waitErr := proc.Wait()
		if result.err != nil {
			return nil, result.err
		}
		if waitErr != nil {
			return nil, waitErr
		}
		if state != nil && !state.Success() {
			return nil, nil
		}
		raw := bytes.Split(result.out, []byte{0})
		paths := make([]string, 0, len(raw))
		for _, entry := range raw {
			path := domain.NormalizePath(strings.TrimSpace(string(entry)))
			if path == "" {
				continue
			}
			paths = append(paths, path)
		}
		return dedupeStrings(paths), nil
	}
}

type largeAnalyzeFindingHeap []domain.Finding

func (h largeAnalyzeFindingHeap) Len() int { return len(h) }

func (h largeAnalyzeFindingHeap) Less(i, j int) bool {
	if h[i].Bytes == h[j].Bytes {
		return h[i].Path > h[j].Path
	}
	return h[i].Bytes < h[j].Bytes
}

func (h largeAnalyzeFindingHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *largeAnalyzeFindingHeap) Push(x any) {
	*h = append(*h, x.(domain.Finding))
}

func (h *largeAnalyzeFindingHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func (h largeAnalyzeFindingHeap) sorted() []domain.Finding {
	out := append([]domain.Finding{}, h...)
	sortFindings(out)
	return out
}

func pushLargeAnalyzeFinding(h *largeAnalyzeFindingHeap, finding domain.Finding, limit int) {
	if limit <= 0 {
		return
	}
	if h.Len() < limit {
		heap.Push(h, finding)
		return
	}
	current := (*h)[0]
	if finding.Bytes > current.Bytes || (finding.Bytes == current.Bytes && finding.Path < current.Path) {
		heap.Pop(h)
		heap.Push(h, finding)
	}
}

func largeFileFinding(path string, info fs.FileInfo, root string) domain.Finding {
	return domain.Finding{
		ID:           uuid.NewString(),
		RuleID:       "analyze.large_files",
		Name:         filepath.Base(path),
		Category:     domain.CategoryLargeFiles,
		Path:         path,
		DisplayPath:  path,
		Risk:         domain.RiskReview,
		Bytes:        info.Size(),
		Action:       domain.ActionAdvisory,
		Recovery:     domain.RecoveryHint{Message: "Analyze is read-only."},
		Status:       domain.StatusAdvisory,
		LastModified: info.ModTime(),
		Fingerprint: domain.Fingerprint{
			Mode:    uint32(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		},
		Source: "Large file under " + root,
	}
}

// scanDuplicates finds duplicate files by content hash
func scanDuplicates(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
	analyzeHooksMu.RLock()
	defer analyzeHooksMu.RUnlock()
	return cachedAnalyzeScan(ctx, "duplicates", targets, analyzeDuplicatesLoader)
}

func analyzeDuplicatesLoader(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
	var findings []domain.Finding
	var warnings []string

	// Size -> []paths with that size
	sizeMap := make(map[int64][]string)
	var totalSize int64

	for _, target := range targets {
		select {
		case <-ctx.Done():
			return findings, warnings, ctx.Err()
		default:
		}

		normalized := domain.NormalizePath(target)
		info, err := os.Lstat(normalized)
		if err != nil {
			warnings = append(warnings, normalized+": "+err.Error())
			continue
		}

		if info.IsDir() {
			// Walk directory to find all files
			err := filepath.Walk(normalized, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.Mode().IsRegular() {
					return nil
				}
				// Skip small files (< 1KB) to save time
				if info.Size() < 1024 {
					return nil
				}
				sizeMap[info.Size()] = append(sizeMap[info.Size()], path)
				totalSize += info.Size()
				return nil
			})
			if err != nil {
				warnings = append(warnings, normalized+": "+err.Error())
			}
		}
	}

	// Find duplicates (files with same size)
	var wg sync.WaitGroup
	var mu sync.Mutex
	hashChan := make(chan []string, 100)

	// Start workers to hash files
	workerCount := runtime.NumCPU()
	if workerCount > 8 {
		workerCount = 8
	}

	results := make(map[string][]string) // hash -> paths

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for paths := range hashChan {
				if len(paths) < 2 {
					continue
				}
				// Hash first 64KB of each file to find potential duplicates
				hashes := make(map[string][]string)
				for _, path := range paths {
					hash, err := quickHash(path)
					if err != nil {
						continue
					}
					hashes[hash] = append(hashes[hash], path)
				}
				mu.Lock()
				for h, p := range hashes {
					if len(p) >= 2 {
						results[h] = append(results[h], p...)
					}
				}
				mu.Unlock()
			}
		}()
	}

	// Send paths to workers
	go func() {
		for size, paths := range sizeMap {
			if size < 1024 || len(paths) < 2 {
				continue
			}
			hashChan <- paths
		}
		close(hashChan)
	}()

	wg.Wait()

	// Convert to findings
	for hash, paths := range results {
		if len(paths) < 2 {
			continue
		}
		var totalDupSize int64
		info, _ := os.Lstat(paths[0])
		if info != nil {
			totalDupSize = info.Size() * int64(len(paths)-1)
		}

		findings = append(findings, domain.Finding{
			ID:           uuid.NewString(),
			RuleID:       "analyze.duplicates",
			Name:         filepath.Base(paths[0]),
			Category:     "duplicates",
			Path:         paths[0],
			DisplayPath:  fmt.Sprintf("%d duplicate files", len(paths)),
			Risk:         domain.RiskReview,
			Bytes:        totalDupSize,
			Action:       domain.ActionAdvisory,
			Recovery:     domain.RecoveryHint{Message: fmt.Sprintf("Found %d identical files", len(paths))},
			Status:       domain.StatusAdvisory,
			Fingerprint:  domain.Fingerprint{ContentHash: hash},
			Source:       "Duplicate files found",
		})
	}

	return findings, warnings, nil
}

// quickHash returns a hash of the first 64KB of a file
func quickHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Read first 64KB
	buf := make([]byte, 65536)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	buf = buf[:n]

	// Simple hash using stdlib
	h := sha256.New()
	h.Write(buf)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
