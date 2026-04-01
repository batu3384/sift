package analyze

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

const previewCacheTTL = 20 * time.Second

type previewCacheEntry struct {
	preview   domain.DirectoryPreview
	expiresAt time.Time
}

type previewCall struct {
	done    chan struct{}
	preview domain.DirectoryPreview
	err     error
}

var (
	previewMu       sync.Mutex
	previewCache    = map[string]previewCacheEntry{}
	previewInFlight = map[string]*previewCall{}
)

func PreviewBatch(paths []string) map[string]domain.DirectoryPreview {
	previews := make(map[string]domain.DirectoryPreview, len(paths))
	for _, path := range paths {
		normalized := domain.NormalizePath(path)
		if normalized == "" {
			continue
		}
		preview, ok := Preview(normalized)
		if !ok {
			continue
		}
		previews[normalized] = preview
	}
	return previews
}

func Preview(path string) (domain.DirectoryPreview, bool) {
	key := domain.NormalizePath(path)
	if key == "" {
		return domain.DirectoryPreview{}, false
	}
	now := time.Now()

	previewMu.Lock()
	if cached, ok := previewCache[key]; ok && now.Before(cached.expiresAt) {
		preview := cloneDirectoryPreview(cached.preview)
		previewMu.Unlock()
		return preview, true
	}
	if call, ok := previewInFlight[key]; ok {
		previewMu.Unlock()
		<-call.done
		if call.err != nil {
			return domain.DirectoryPreview{}, false
		}
		return cloneDirectoryPreview(call.preview), true
	}
	call := &previewCall{done: make(chan struct{})}
	previewInFlight[key] = call
	previewMu.Unlock()

	preview, ok, err := loadDirectoryPreview(key)

	previewMu.Lock()
	delete(previewInFlight, key)
	call.err = err
	if ok && err == nil {
		call.preview = cloneDirectoryPreview(preview)
		previewCache[key] = previewCacheEntry{
			preview:   cloneDirectoryPreview(preview),
			expiresAt: time.Now().Add(previewCacheTTL),
		}
	}
	close(call.done)
	previewMu.Unlock()

	if err != nil || !ok {
		return domain.DirectoryPreview{}, false
	}
	return preview, true
}

func loadDirectoryPreview(path string) (domain.DirectoryPreview, bool, error) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return domain.DirectoryPreview{}, false, err
	}
	entries, err := os.ReadDir(path)
	preview := domain.DirectoryPreview{Path: path}
	if err != nil {
		preview.Unavailable = true
		return preview, true, nil
	}
	preview.Names = make([]string, 0, 3)
	preview.DirNames = make([]string, 0, 2)
	preview.FileSamples = make([]domain.DirectoryPreviewFile, 0, 4)
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		preview.Total++
		if entry.IsDir() {
			preview.Dirs++
			if len(preview.DirNames) < 2 {
				preview.DirNames = append(preview.DirNames, name)
			}
		} else {
			preview.Files++
			size := int64(0)
			if info, err := entry.Info(); err == nil {
				size = info.Size()
			}
			if len(preview.FileSamples) < 4 {
				preview.FileSamples = append(preview.FileSamples, domain.DirectoryPreviewFile{Name: name, Size: size})
			}
		}
		if len(preview.Names) < 3 {
			preview.Names = append(preview.Names, name)
		}
	}
	sort.SliceStable(preview.FileSamples, func(i, j int) bool {
		if preview.FileSamples[i].Size == preview.FileSamples[j].Size {
			return preview.FileSamples[i].Name < preview.FileSamples[j].Name
		}
		return preview.FileSamples[i].Size > preview.FileSamples[j].Size
	})
	return preview, true, nil
}

func cloneDirectoryPreview(preview domain.DirectoryPreview) domain.DirectoryPreview {
	out := preview
	out.Path = filepath.Clean(preview.Path)
	out.Names = append([]string{}, preview.Names...)
	out.DirNames = append([]string{}, preview.DirNames...)
	out.FileSamples = append([]domain.DirectoryPreviewFile{}, preview.FileSamples...)
	return out
}
