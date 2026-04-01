package analyze

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

const scanCacheTTL = 5 * time.Second

type ScanLoader func(context.Context, []string) ([]domain.Finding, []string, error)

type scanCacheEntry struct {
	findings  []domain.Finding
	warnings  []string
	expiresAt time.Time
}

type scanCall struct {
	done     chan struct{}
	findings []domain.Finding
	warnings []string
	err      error
}

var (
	scanMu       sync.Mutex
	scanCache    = map[string]scanCacheEntry{}
	scanInFlight = map[string]*scanCall{}
)

func CachedScan(ctx context.Context, kind string, targets []string, loader ScanLoader) ([]domain.Finding, []string, error) {
	key := scanKey(kind, targets)
	if key == kind {
		return loader(ctx, targets)
	}
	now := time.Now()

	scanMu.Lock()
	if cached, ok := scanCache[key]; ok && now.Before(cached.expiresAt) {
		findings := cloneFindings(cached.findings)
		warnings := append([]string{}, cached.warnings...)
		scanMu.Unlock()
		return findings, warnings, nil
	}
	if call, ok := scanInFlight[key]; ok {
		scanMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-call.done:
			return cloneFindings(call.findings), append([]string{}, call.warnings...), call.err
		}
	}
	call := &scanCall{done: make(chan struct{})}
	scanInFlight[key] = call
	scanMu.Unlock()

	findings, warnings, err := loader(ctx, targets)

	scanMu.Lock()
	delete(scanInFlight, key)
	call.findings = cloneFindings(findings)
	call.warnings = append([]string{}, warnings...)
	call.err = err
	if err == nil {
		scanCache[key] = scanCacheEntry{
			findings:  cloneFindings(findings),
			warnings:  append([]string{}, warnings...),
			expiresAt: time.Now().Add(scanCacheTTL),
		}
	}
	close(call.done)
	scanMu.Unlock()

	return findings, warnings, err
}

func cloneFindings(in []domain.Finding) []domain.Finding {
	out := append([]domain.Finding{}, in...)
	for i := range out {
		out[i].CommandArgs = append([]string{}, out[i].CommandArgs...)
		out[i].TaskVerify = append([]string{}, out[i].TaskVerify...)
		out[i].SuggestedBy = append([]string{}, out[i].SuggestedBy...)
	}
	return out
}

func scanKey(kind string, targets []string) string {
	var b strings.Builder
	b.WriteString(kind)
	for _, target := range targets {
		normalized := domain.NormalizePath(target)
		if normalized == "" {
			continue
		}
		b.WriteByte('|')
		b.WriteString(normalized)
		info, err := os.Lstat(normalized)
		switch {
		case err == nil:
			fmt.Fprintf(&b, "#%d#%d#%d", info.ModTime().UnixNano(), info.Size(), uint32(info.Mode()))
		case errors.Is(err, os.ErrNotExist):
			b.WriteString("#missing")
		default:
			b.WriteString("#err")
		}
	}
	return b.String()
}
