package rules

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/batu3384/sift/internal/analyze"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

type stubAdapter struct {
	name  string
	roots platform.CuratedRoots
}

func (s stubAdapter) Name() string                        { return s.name }
func (s stubAdapter) CuratedRoots() platform.CuratedRoots { return s.roots }
func (s stubAdapter) ProtectedPaths() []string            { return nil }
func (s stubAdapter) ResolveTargets(in []string) []string { return in }
func (s stubAdapter) ListApps(context.Context, bool) ([]domain.AppEntry, error) {
	return []domain.AppEntry{{DisplayName: "KeepApp"}}, nil
}
func (s stubAdapter) DiscoverRemnants(context.Context, domain.AppEntry) ([]string, []string, error) {
	return nil, nil, nil
}
func (s stubAdapter) MaintenanceTasks(context.Context) []domain.MaintenanceTask { return nil }
func (s stubAdapter) Diagnostics(context.Context) []platform.Diagnostic         { return nil }
func (s stubAdapter) IsAdminPath(string) bool                                   { return false }
func (s stubAdapter) IsFileInUse(context.Context, string) bool                  { return false }
func (s stubAdapter) IsProcessRunning(...string) bool                           { return false }

func resetAnalyzeScanState() {
	analyze.ResetCachesForTests()
}

func TestMeasurePathIgnoresSymlinkLoop(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	unicodeDir := filepath.Join(root, "ünicode")
	if err := os.MkdirAll(unicodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(unicodeDir, "cache.bin")
	if err := os.WriteFile(file, []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(unicodeDir, "loop")); err != nil {
		t.Fatal(err)
	}
	size, _, err := MeasurePath(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if size != 10 {
		t.Fatalf("expected size 10, got %d", size)
	}
}

func TestInlineWalkSkipsSymlinks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file.dat"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(sub, "loop")); err != nil {
		t.Fatal(err)
	}
	var count int
	inlineWalk(context.Background(), root, func(info os.FileInfo) {
		count++
	})
	// only sub/ and file.dat should be visited; loop symlink must be skipped
	if count != 2 {
		t.Fatalf("expected 2 entries (dir + file), got %d", count)
	}
}

func TestInlineWalkRespectsContextCancellation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// create enough files that cancellation has a chance to cut in
	for i := 0; i < 20; i++ {
		name := filepath.Join(root, fmt.Sprintf("f%d.bin", i))
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before walking
	var count int
	inlineWalk(ctx, root, func(os.FileInfo) { count++ })
	// With an already-cancelled context the walk should return without visiting any entries
	if count != 0 {
		t.Fatalf("expected 0 entries after context cancel, got %d", count)
	}
}

func TestMeasurePathCancelledContextReturnsEarly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Populate enough files to make the worker pool start before we cancel
	for i := 0; i < 50; i++ {
		sub := filepath.Join(root, fmt.Sprintf("d%d", i))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sub, "data"), make([]byte, 1024), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Should not hang — context is already cancelled
	done := make(chan struct{})
	go func() {
		MeasurePath(ctx, root) //nolint:errcheck
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("MeasurePath did not return within 5 seconds with a cancelled context")
	}
}

func TestInstallerScannerOnlyReturnsOldArchives(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	oldFile := filepath.Join(root, "old-installer.dmg")
	if err := os.WriteFile(oldFile, []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-20 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	newFile := filepath.Join(root, "new-installer.zip")
	if err := os.WriteFile(newFile, []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := stubAdapter{
		name: "test",
		roots: platform.CuratedRoots{
			Installer: []string{root},
		},
	}
	defs := ByIDs([]string{"installer_leftovers"})
	if len(defs) != 1 {
		t.Fatalf("expected installer definition")
	}
	findings, _, err := defs[0].Scanner(context.Background(), adapter, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Path != oldFile {
		t.Fatalf("expected %s, got %s", oldFile, findings[0].Path)
	}
}

func TestInstallerScannerIncludesExpandedArchiveTypesAndIncompleteDownloads(t *testing.T) {
	root := t.TempDir()
	// Create a Downloads subdirectory since scanIncompleteDownloads only scans paths containing "downloads"
	downloadsDir := filepath.Join(root, "Downloads")
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-20 * 24 * time.Hour)
	files := []string{
		filepath.Join(downloadsDir, "archive.mpkg"),
		filepath.Join(downloadsDir, "archive.xip"),
	}
	for _, path := range files {
		if err := os.WriteFile(path, []byte("1234"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}
	incomplete := filepath.Join(downloadsDir, "video.crdownload")
	if err := os.WriteFile(incomplete, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := currentAdapterIsFileInUse
	defer func() { currentAdapterIsFileInUse = original }()
	currentAdapterIsFileInUse = func(ctx context.Context, path string) bool {
		return false
	}
	findings, warnings, err := scanInstallerFiles(context.Background(), stubAdapter{
		name: "darwin",
		roots: platform.CuratedRoots{
			Installer: []string{root},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	want := map[string]bool{
		files[0]:   false,
		files[1]:   false,
		incomplete: false,
	}
	for _, finding := range findings {
		if _, ok := want[finding.Path]; ok {
			want[finding.Path] = true
		}
	}
	for path, seen := range want {
		if !seen {
			t.Fatalf("expected installer finding for %s, got %+v", path, findings)
		}
	}
}

func TestInstallerScannerSkipsActiveIncompleteDownloads(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, "active.download")
	if err := os.WriteFile(active, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := currentAdapterIsFileInUse
	defer func() { currentAdapterIsFileInUse = original }()
	currentAdapterIsFileInUse = func(ctx context.Context, path string) bool {
		return path == active
	}
	findings, warnings, err := scanInstallerFiles(context.Background(), stubAdapter{
		name: "darwin",
		roots: platform.CuratedRoots{
			Installer: []string{root},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected active incomplete download to be skipped, got %+v", findings)
	}
	if len(warnings) == 0 || !strings.Contains(strings.Join(warnings, " | "), "skipped active download") {
		t.Fatalf("expected active download warning, got %v", warnings)
	}
}

func TestAnalysisScannerReturnsLargestChildren(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "small.txt"), []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	largeDir := filepath.Join(root, "large")
	if err := os.MkdirAll(largeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(largeDir, "blob.bin"), make([]byte, 32), 0o644); err != nil {
		t.Fatal(err)
	}
	defs := AnalysisDefinitions([]string{root})
	findings, _, err := defs[0].Scanner(context.Background(), stubAdapter{name: "test"}, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].Name != "large" {
		t.Fatalf("expected largest child first, got %s", findings[0].Name)
	}
	if findings[0].Action != domain.ActionAdvisory || findings[0].Status != domain.StatusAdvisory {
		t.Fatalf("expected advisory analysis item, got %+v", findings[0])
	}
}

func TestLargeFileScannerReturnsLargestNestedFiles(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	large := filepath.Join(nested, "huge.bin")
	if err := os.WriteFile(large, make([]byte, analyzeLargeFileMinBytes+4096), 0o644); err != nil {
		t.Fatal(err)
	}
	small := filepath.Join(root, "small.txt")
	if err := os.WriteFile(small, []byte("tiny"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanLargeFiles(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one large-file finding, got %d", len(findings))
	}
	if findings[0].Category != domain.CategoryLargeFiles {
		t.Fatalf("expected large_files category, got %s", findings[0].Category)
	}
	if findings[0].Path != large {
		t.Fatalf("expected %s, got %s", large, findings[0].Path)
	}
	if findings[0].Action != domain.ActionAdvisory || findings[0].Status != domain.StatusAdvisory {
		t.Fatalf("expected advisory large-file finding, got %+v", findings[0])
	}
}

func TestLargeFileScannerCapsResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < maxAnalyzeLargeFiles+5; i++ {
		name := filepath.Join(root, fmt.Sprintf("large-%02d.bin", i))
		if err := os.WriteFile(name, make([]byte, analyzeLargeFileMinBytes+i+1), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	findings, warnings, err := scanLargeFiles(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != maxAnalyzeLargeFiles {
		t.Fatalf("expected capped results %d, got %d", maxAnalyzeLargeFiles, len(findings))
	}
	if len(warnings) == 0 || warnings[len(warnings)-1] != "large file analysis capped to top results" {
		t.Fatalf("expected capping warning, got %v", warnings)
	}
	if findings[0].Bytes < findings[len(findings)-1].Bytes {
		t.Fatalf("expected descending findings, got %+v", findings)
	}
}

func TestAnalysisScannerCapsResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < maxAnalyzeDiskUsage+5; i++ {
		name := filepath.Join(root, fmt.Sprintf("item-%02d.bin", i))
		if err := os.WriteFile(name, make([]byte, i+1), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	findings, warnings, err := scanDiskUsage(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != maxAnalyzeDiskUsage {
		t.Fatalf("expected capped results %d, got %d", maxAnalyzeDiskUsage, len(findings))
	}
	if len(warnings) == 0 || warnings[len(warnings)-1] != "disk usage analysis capped to top results" {
		t.Fatalf("expected capping warning, got %v", warnings)
	}
}

func TestScanDiskUsageFoldsSingleChildDirectories(t *testing.T) {
	root := t.TempDir()
	folded := filepath.Join(root, "apps", "web", "cache")
	if err := os.MkdirAll(folded, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folded, "blob.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanDiskUsage(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one folded finding, got %+v", findings)
	}
	if findings[0].Path != folded {
		t.Fatalf("expected folded path %s, got %s", folded, findings[0].Path)
	}
	if findings[0].Name != filepath.Join("apps", "web", "cache") {
		t.Fatalf("expected folded name, got %q", findings[0].Name)
	}
	if !strings.Contains(findings[0].Source, "folded") {
		t.Fatalf("expected folded source label, got %q", findings[0].Source)
	}
}

func TestScanDiskUsagePreservesBranchingDirectories(t *testing.T) {
	root := t.TempDir()
	apps := filepath.Join(root, "apps")
	if err := os.MkdirAll(filepath.Join(apps, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(apps, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apps, "web", "blob.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanDiskUsage(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one immediate child finding, got %+v", findings)
	}
	if findings[0].Path != apps {
		t.Fatalf("expected branching directory to stay at %s, got %s", apps, findings[0].Path)
	}
	if findings[0].Name != "apps" {
		t.Fatalf("expected immediate child name, got %q", findings[0].Name)
	}
}

func TestCleanupSourceLabelRecognizesPlaywrightAndXcodeDeviceSupport(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/Users/example/Library/Caches/com.konghq.insomnia", want: "Insomnia cache"},
		{path: "/Users/example/Library/Caches/com.tinyapp.TablePlus", want: "TablePlus cache"},
		{path: "/Users/example/Library/Caches/com.getpaw.Paw", want: "Paw API cache"},
		{path: "/Users/example/Library/Caches/com.charlesproxy.charles", want: "Charles Proxy cache"},
		{path: "/Users/example/Library/Caches/com.proxyman.NSProxy", want: "Proxyman cache"},
		{path: "/Users/example/Library/Caches/com.github.GitHubDesktop", want: "GitHub Desktop cache"},
		{path: "/Users/example/Library/Caches/com.microsoft.VSCode", want: "VS Code cache"},
		{path: "/Users/example/Library/Caches/com.bohemiancoding.sketch3", want: "Sketch cache"},
		{path: "/Users/example/Library/Application Support/com.bohemiancoding.sketch3/cache", want: "Sketch app cache"},
		{path: "/Users/example/Library/Caches/Adobe", want: "Adobe cache"},
		{path: "/Users/example/Library/Application Support/Adobe/Common/Media Cache Files", want: "Adobe media cache"},
		{path: "/Users/example/Library/Caches/com.adobe.PremierePro.24", want: "Premiere Pro cache"},
		{path: "/Users/example/Library/Caches/com.apple.FinalCut", want: "Final Cut Pro cache"},
		{path: "/Users/example/Library/Caches/com.blackmagic-design.DaVinciResolve", want: "DaVinci Resolve cache"},
		{path: "/Users/example/Library/Caches/net.telestream.screenflow10", want: "ScreenFlow cache"},
		{path: "/Users/example/Library/Caches/org.blenderfoundation.blender", want: "Blender cache"},
		{path: "/Users/example/Library/Caches/SentryCrash", want: "Sentry crash reports"},
		{path: "/Users/example/Library/Caches/KSCrash", want: "KSCrash reports"},
		{path: "/Users/example/Library/Caches/com.crashlytics.data", want: "Crashlytics data"},
		{path: "/Users/example/Library/Application Support/Figma/GPUCache", want: "Figma GPU cache"},
		{path: "/Users/example/Library/Application Support/Postman/Service Worker/CacheStorage", want: "Postman service worker cache"},
		{path: "/Users/example/Library/Application Support/Zed/CachedData", want: "Zed data cache"},
		{path: "/Users/example/Library/Caches/com.anthropic.claudefordesktop", want: "Claude desktop cache"},
		{path: "/Users/example/Library/Application Support/Claude/DawnGraphiteCache", want: "Claude Dawn cache"},
		{path: "/Users/example/Library/Application Support/Claude/DawnWebGPUCache", want: "Claude WebGPU cache"},
		{path: "/Users/example/AppData/Roaming/Figma/logs", want: "Figma logs"},
		{path: "/Users/example/AppData/Roaming/Postman/GPUCache", want: "Postman GPU cache"},
		{path: "/Users/example/AppData/Roaming/Zed/Code Cache", want: "Zed code cache"},
		{path: "/Users/example/Library/Application Support/Claude/GPUCache", want: "Claude GPU cache"},
		{path: "/Users/example/Library/Application Support/Claude/Service Worker/CacheStorage", want: "Claude service worker cache"},
		{path: "/Users/example/Library/Caches/com.openai.chat", want: "ChatGPT cache"},
		{path: "/Users/example/Library/Application Support/ChatGPT/CachedData", want: "ChatGPT data cache"},
		{path: "/Users/example/AppData/Roaming/Claude/logs", want: "Claude logs"},
		{path: "/Users/example/AppData/Roaming/ChatGPT/GPUCache", want: "ChatGPT GPU cache"},
		{path: "/Users/example/Library/Application Support/Code/DawnGraphiteCache", want: "VS Code Dawn cache"},
		{path: "/Users/example/Library/Application Support/Code/DawnWebGPUCache", want: "VS Code WebGPU cache"},
		{path: "/Users/example/Library/Application Support/Cursor/GPUCache", want: "Cursor GPU cache"},
		{path: "/Users/example/Library/Application Support/Cursor/Service Worker/CacheStorage", want: "Cursor service worker cache"},
		{path: "/Users/example/AppData/Roaming/VSCodium/GPUCache", want: "VSCodium GPU cache"},
		{path: "/Users/example/AppData/Roaming/VSCodium/logs", want: "VSCodium logs"},
		{path: "/Users/example/Library/Application Support/Slack/GPUCache", want: "Slack GPU cache"},
		{path: "/Users/example/Library/Application Support/Slack/Service Worker/CacheStorage", want: "Slack service worker cache"},
		{path: "/Users/example/AppData/Roaming/discord/GPUCache", want: "Discord GPU cache"},
		{path: "/Users/example/Library/Application Support/Microsoft/Teams/Cache", want: "Teams cache"},
		{path: "/Users/example/Library/Application Support/Microsoft/Teams/Code Cache", want: "Teams code cache"},
		{path: "/Users/example/Library/Application Support/Microsoft/Teams/GPUCache", want: "Teams GPU cache"},
		{path: "/Users/example/Library/Application Support/Legcord/GPUCache", want: "Legcord GPU cache"},
		{path: "/Users/example/Library/Application Support/Legcord/logs", want: "Legcord logs"},
		{path: "/Users/example/Library/Application Support/Steam/htmlcache", want: "Steam web cache"},
		{path: "/Users/example/Library/Application Support/Steam/appcache", want: "Steam app cache"},
		{path: "/Users/example/Library/Application Support/Steam/depotcache", want: "Steam depot cache"},
		{path: "/Users/example/Library/Application Support/Steam/steamapps/shadercache", want: "Steam shader cache"},
		{path: "/Users/example/Library/Application Support/Steam/logs", want: "Steam logs"},
		{path: "/Users/example/Library/Application Support/Battle.net/Cache", want: "Battle.net app cache"},
		{path: "/Users/example/.android/cache", want: "Android cache"},
		{path: "/Users/example/.android/build-cache", want: "Android build cache"},
		{path: "/Users/example/.pub-cache", want: "Dart pub cache"},
		{path: "/Users/example/flutter/bin/cache", want: "Flutter SDK cache"},
		{path: "/Users/example/.expo", want: "Expo cache"},
		{path: "/Users/example/.expo-shared", want: "Expo shared cache"},
		{path: "/Users/example/.cache/deno", want: "Deno cache"},
		{path: "/Users/example/Library/Caches/bazel", want: "Bazel cache"},
		{path: "/Users/example/.grafana/cache", want: "Grafana cache"},
		{path: "/Users/example/.prometheus/data/wal", want: "Prometheus WAL"},
		{path: "/Users/example/.jenkins/workspace/demo/target", want: "Jenkins workspace artifacts"},
		{path: "/Users/example/Library/Caches/com.mongodb.compass", want: "MongoDB Compass cache"},
		{path: "/Users/example/Library/Application Support/RedisInsight/Code Cache", want: "Redis Insight code cache"},
		{path: "/Users/example/Library/Caches/net.shinyfrog.bear", want: "Bear cache"},
		{path: "/Users/example/Library/Application Support/Logseq/Service Worker/CacheStorage", want: "Logseq service worker cache"},
		{path: "/Users/example/Library/Application Support/Riot Client/GPUCache", want: "Riot Client GPU cache"},
		{path: "/Users/example/Library/Application Support/Lark/Code Cache", want: "Feishu code cache"},
		{path: "/Users/example/Library/Application Support/DingTalk/GPUCache", want: "DingTalk GPU cache"},
		{path: "/Users/example/Library/Caches/com.anydesk.anydesk", want: "AnyDesk cache"},
		{path: "/Users/example/Library/Caches/com.gog.galaxy", want: "GOG Galaxy cache"},
		{path: "/Users/example/Library/Caches/com.ea.app", want: "EA app cache"},
		{path: "/Users/example/Library/Caches/com.tencent.meeting", want: "Tencent Meeting cache"},
		{path: "/Users/example/Library/Caches/com.tencent.WeWorkMac", want: "WeCom cache"},
		{path: "/Users/example/Library/Caches/com.teamviewer.host", want: "TeamViewer cache"},
		{path: "/Users/example/Library/Caches/com.todesk.app", want: "ToDesk cache"},
		{path: "/Users/example/Library/Caches/com.sunlogin.client", want: "Sunlogin cache"},
		{path: "/Users/example/Library/Caches/com.airmail.direct", want: "Airmail cache"},
		{path: "/Users/example/Library/Caches/com.any.do.mac", want: "Any.do cache"},
		{path: "/Users/example/Library/Caches/cx.c3.theunarchiver", want: "The Unarchiver cache"},
		{path: "/Users/example/Library/Caches/com.youdao.YoudaoDict", want: "Youdao Dictionary cache"},
		{path: "/Users/example/Library/Caches/com.eudic.mac", want: "Eudict cache"},
		{path: "/Users/example/Library/Caches/com.bob-build.Bob", want: "Bob Translation cache"},
		{path: "/Users/example/Library/Caches/com.tw93.MiaoYan", want: "MiaoYan cache"},
		{path: "/Users/example/Library/Caches/com.flomoapp.mac", want: "Flomo cache"},
		{path: "/Users/example/Library/Application Support/Quark/Cache/videoCache", want: "Quark video cache"},
		{path: "/Users/example/Library/Caches/com.maxon.cinema4d", want: "Cinema 4D cache"},
		{path: "/Users/example/Library/Caches/com.autodesk.autocad", want: "Autodesk cache"},
		{path: "/Users/example/Library/Caches/com.sketchup.SketchUp", want: "SketchUp cache"},
		{path: "/Users/example/Library/Caches/com.netease.163music", want: "NetEase Music cache"},
		{path: "/Users/example/Library/Caches/com.tencent.QQMusic", want: "QQ Music cache"},
		{path: "/Users/example/Library/Caches/com.kugou.mac", want: "Kugou Music cache"},
		{path: "/Users/example/Library/Caches/com.kuwo.mac", want: "Kuwo Music cache"},
		{path: "/Users/example/Library/Caches/com.iqiyi.player", want: "iQIYI cache"},
		{path: "/Users/example/Library/Caches/com.tencent.tenvideo", want: "Tencent Video cache"},
		{path: "/Users/example/Library/Caches/tv.danmaku.bili", want: "Bilibili cache"},
		{path: "/Users/example/Library/Caches/com.douyu.live", want: "Douyu cache"},
		{path: "/Users/example/Library/Caches/com.huya.live", want: "Huya cache"},
		{path: "/Users/example/Library/Caches/com.reincubate.camo", want: "Camo cache"},
		{path: "/Users/example/Library/Caches/com.xnipapp.xnip", want: "Xnip cache"},
		{path: "/Users/example/Library/Caches/org.m0k.transmission", want: "Transmission cache"},
		{path: "/Users/example/Library/Caches/com.qbittorrent.qBittorrent", want: "qBittorrent cache"},
		{path: "/Users/example/Library/Caches/notion.id", want: "Notion cache"},
		{path: "/Users/example/Library/Caches/md.obsidian", want: "Obsidian cache"},
		{path: "/Users/example/Library/Caches/com.runningwithcrayons.Alfred", want: "Alfred cache"},
		{path: "/Users/example/Library/Caches/com.microsoft.teams2", want: "Teams cache"},
		{path: "/Users/example/Library/Caches/us.zoom.xos", want: "Zoom cache"},
		{path: "/Users/example/Library/Caches/ru.keepcoder.Telegram", want: "Telegram cache"},
		{path: "/Users/example/Library/Caches/com.tencent.xinWeChat", want: "WeChat cache"},
		{path: "/Users/example/Library/Caches/com.skype.skype", want: "Skype cache"},
		{path: "/Users/example/Library/Caches/net.whatsapp.WhatsApp", want: "WhatsApp cache"},
		{path: "/Users/example/Library/Caches/com.todoist.mac.Todoist", want: "Todoist cache"},
		{path: "/Users/example/Library/Caches/com.valvesoftware.steam", want: "Steam cache"},
		{path: "/Users/example/Library/Caches/com.epicgames.EpicGamesLauncher", want: "Epic Games cache"},
		{path: "/Users/example/Library/Caches/com.blizzard.Battle.net", want: "Battle.net cache"},
		{path: "/Users/example/Library/Caches/com.colliderli.iina", want: "IINA cache"},
		{path: "/Users/example/Library/Caches/org.videolan.vlc", want: "VLC cache"},
		{path: "/Users/example/Library/Caches/io.mpv", want: "MPV cache"},
		{path: "/Users/example/Library/Caches/tv.plex.player.desktop", want: "Plex cache"},
		{path: "/Users/example/Library/Caches/com.readdle.smartemail-Mac", want: "Spark cache"},
		{path: "/Users/example/AppData/Roaming/Code/GPUCache", want: "VS Code GPU cache"},
		{path: "/Users/example/Library/Caches/ms-playwright", want: "Playwright browser cache"},
		{path: "/Users/example/.cache/puppeteer", want: "Puppeteer browser cache"},
		{path: "/Users/example/Library/Developer/Xcode/iOS DeviceSupport", want: "Xcode iOS device support"},
		{path: "/Users/example/Library/Containers/com.docker.docker/Data/log", want: "Docker Desktop logs"},
	} {
		if got := cleanupSourceLabel(tc.path, "fallback"); got != tc.want {
			t.Fatalf("expected %q for %s, got %q", tc.want, tc.path, got)
		}
	}
}

func TestShouldTreatRootAsLeafForNewDeveloperCaches(t *testing.T) {
	t.Parallel()
	for _, root := range []string{
		"/Users/example/.android/cache",
		"/Users/example/.android/build-cache",
		"/Users/example/.pub-cache",
		"/Users/example/flutter/bin/cache",
		"/Users/example/.expo",
		"/Users/example/.expo-shared",
		"/Users/example/.cache/ms-playwright",
		"/Users/example/.cache/puppeteer",
		"/Users/example/Library/Developer/Xcode/iOS DeviceSupport",
		"/Users/example/Library/Application Support/com.bohemiancoding.sketch3/cache",
		"/Users/example/Library/Application Support/Adobe/Common/Media Cache Files",
		"/Users/example/Library/Application Support/Code/DawnGraphiteCache",
		"/Users/example/Library/Application Support/Code/DawnWebGPUCache",
		"/Users/example/Library/Application Support/Claude/DawnGraphiteCache",
		"/Users/example/Library/Application Support/Claude/DawnWebGPUCache",
		"/Users/example/Library/Application Support/Figma/Service Worker/CacheStorage",
		"/Users/example/Library/Application Support/Postman/Service Worker/CacheStorage",
		"/Users/example/Library/Application Support/Zed/Service Worker/CacheStorage",
		"/Users/example/Library/Application Support/Claude/Service Worker/CacheStorage",
		"/Users/example/Library/Application Support/Cursor/Service Worker/CacheStorage",
		"/Users/example/Library/Application Support/Slack/Service Worker/CacheStorage",
		"/Users/example/Library/Application Support/Microsoft/Teams/Cache",
		"/Users/example/Library/Application Support/Microsoft/Teams/Code Cache",
		"/Users/example/Library/Application Support/Microsoft/Teams/GPUCache",
		"/Users/example/Library/Application Support/Legcord/GPUCache",
		"/Users/example/Library/Application Support/Legcord/logs",
		"/Users/example/Library/Application Support/Steam/htmlcache",
		"/Users/example/Library/Application Support/Steam/appcache",
		"/Users/example/Library/Application Support/Steam/depotcache",
		"/Users/example/Library/Application Support/Steam/steamapps/shadercache",
		"/Users/example/Library/Application Support/Steam/logs",
		"/Users/example/Library/Application Support/Battle.net/Cache",
		"/Users/example/.cache/deno",
		"/Users/example/Library/Caches/bazel",
		"/Users/example/Library/Caches/com.mongodb.compass",
		"/Users/example/Library/Application Support/RedisInsight/Code Cache",
		"/Users/example/Library/Caches/net.shinyfrog.bear",
		"/Users/example/Library/Application Support/Logseq/Service Worker/CacheStorage",
		"/Users/example/Library/Application Support/Logseq/logs",
		"/Users/example/Library/Application Support/Riot Client/GPUCache",
		"/Users/example/Library/Application Support/Riot Client/logs",
		"/Users/example/Library/Application Support/Lark/Code Cache",
		"/Users/example/Library/Application Support/Lark/logs",
		"/Users/example/Library/Application Support/DingTalk/GPUCache",
		"/Users/example/Library/Application Support/DingTalk/logs",
		"/Users/example/Library/Caches/com.anydesk.anydesk",
		"/Users/example/Library/Caches/com.gog.galaxy",
		"/Users/example/Library/Caches/com.ea.app",
		"/Users/example/Library/Caches/notion.id",
		"/Users/example/Library/Caches/md.obsidian",
		"/Users/example/Library/Caches/com.runningwithcrayons.Alfred",
		"/Users/example/Library/Caches/com.microsoft.teams2",
		"/Users/example/Library/Caches/us.zoom.xos",
		"/Users/example/Library/Caches/ru.keepcoder.Telegram",
		"/Users/example/Library/Caches/com.tencent.xinWeChat",
		"/Users/example/Library/Caches/com.skype.skype",
		"/Users/example/Library/Caches/net.whatsapp.WhatsApp",
		"/Users/example/Library/Caches/com.todoist.mac.Todoist",
		"/Users/example/Library/Caches/com.valvesoftware.steam",
		"/Users/example/Library/Caches/com.epicgames.EpicGamesLauncher",
		"/Users/example/Library/Caches/com.blizzard.Battle.net",
		"/Users/example/Library/Caches/com.colliderli.iina",
		"/Users/example/Library/Caches/org.videolan.vlc",
		"/Users/example/Library/Caches/io.mpv",
		"/Users/example/Library/Caches/tv.plex.player.desktop",
		"/Users/example/Library/Caches/com.readdle.smartemail-Mac",
	} {
		if !shouldTreatRootAsLeaf(root, domain.CategoryDeveloperCaches) {
			t.Fatalf("expected root %s to be treated as a leaf", root)
		}
	}
}

func TestLargeFileScannerUsesSpotlightAssistWithoutDuplicateResults(t *testing.T) {
	root := t.TempDir()
	large := filepath.Join(root, "huge.bin")
	if err := os.WriteFile(large, make([]byte, analyzeLargeFileMinBytes+8192), 0o644); err != nil {
		t.Fatal(err)
	}
	original := spotlightLargeFileSearch
	t.Cleanup(func() {
		spotlightLargeFileSearch = original
	})
	called := 0
	spotlightLargeFileSearch = func(ctx context.Context, target string, minBytes int64) ([]string, error) {
		called++
		if target != root {
			t.Fatalf("unexpected spotlight target: %s", target)
		}
		if minBytes != analyzeLargeFileMinBytes {
			t.Fatalf("unexpected spotlight threshold: %d", minBytes)
		}
		return []string{large}, nil
	}

	findings, warnings, err := scanLargeFiles(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if called != 1 {
		t.Fatalf("expected spotlight helper to be called once, got %d", called)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one deduplicated large-file finding, got %+v", findings)
	}
	if findings[0].Path != large {
		t.Fatalf("expected spotlight-discovered file, got %+v", findings[0])
	}
}

func TestLargeFileScannerCachesImmediateRepeat(t *testing.T) {
	root := t.TempDir()
	resetAnalyzeScanState()
	original := analyzeLargeFilesLoader
	defer func() {
		analyzeLargeFilesLoader = original
		resetAnalyzeScanState()
	}()

	calls := 0
	analyzeLargeFilesLoader = func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		calls++
		return []domain.Finding{{
			ID:          "cached",
			RuleID:      "analyze.large_files",
			Name:        "huge.bin",
			Category:    domain.CategoryLargeFiles,
			Path:        filepath.Join(root, "huge.bin"),
			DisplayPath: filepath.Join(root, "huge.bin"),
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Status:      domain.StatusAdvisory,
		}}, nil, nil
	}

	first, _, err := scanLargeFiles(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := scanLargeFiles(context.Background(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected cached repeat to reuse one loader call, got %d", calls)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected cached findings, got %+v and %+v", first, second)
	}
}

func TestDiskUsageScannerCoalescesConcurrentRequests(t *testing.T) {
	root := t.TempDir()
	resetAnalyzeScanState()
	original := analyzeDiskUsageLoader
	defer func() {
		analyzeDiskUsageLoader = original
		resetAnalyzeScanState()
	}()

	var calls atomic.Int32
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	analyzeDiskUsageLoader = func(ctx context.Context, targets []string) ([]domain.Finding, []string, error) {
		if calls.Add(1) == 1 {
			started <- struct{}{}
		}
		<-release
		return []domain.Finding{{
			ID:          "coalesced",
			RuleID:      "analyze.disk_usage",
			Name:        filepath.Base(root),
			Category:    domain.CategoryDiskUsage,
			Path:        root,
			DisplayPath: root,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Status:      domain.StatusAdvisory,
		}}, nil, nil
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, err := scanDiskUsage(context.Background(), []string{root})
		errs <- err
	}()
	<-started
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, err := scanDiskUsage(context.Background(), []string{root})
		errs <- err
	}()
	time.Sleep(25 * time.Millisecond)
	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected concurrent requests to coalesce into one loader call, got %d", got)
	}
}

func TestScanImmediateChildrenSkipsSymlinkBaseDirectories(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(root, "linked")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanImmediateChildren(context.Background(), linkDir, domain.CategoryTempFiles, domain.RiskSafe, domain.ActionTrash, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected symlinked base dir to be skipped, got %d findings", len(findings))
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for symlinked base dir")
	}
}

func TestScanRootEntriesReturnsImmediateChildrenInsteadOfRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "Caches")
	if err := os.MkdirAll(filepath.Join(cacheRoot, "AppA"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cacheRoot, "AppB"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "AppA", "a.bin"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "AppB", "b.bin"), []byte("123"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanRootEntries(context.Background(), stubAdapter{name: "test"}, []string{cacheRoot}, domain.CategorySystemClutter, domain.RiskSafe, domain.ActionTrash, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 2 {
		t.Fatalf("expected immediate children only, got %d findings", len(findings))
	}
	if findings[0].Path == cacheRoot || findings[1].Path == cacheRoot {
		t.Fatal("expected child findings, not the cache root itself")
	}
	if findings[0].Name != "AppA" {
		t.Fatalf("expected larger child first, got %s", findings[0].Name)
	}
}

func TestScanRootEntriesTreatsLeafCacheRootsAsSingleFinding(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	slackCache := filepath.Join(root, "Library", "Application Support", "Slack", "Cache")
	if err := os.MkdirAll(slackCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(slackCache, "blob.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := stubAdapter{
		name: "test",
		roots: platform.CuratedRoots{
			Developer: []string{slackCache},
		},
	}
	findings, warnings, err := scanRootEntries(context.Background(), adapter, adapter.CuratedRoots().Developer, domain.CategoryDeveloperCaches, domain.RiskReview, domain.ActionTrash, "Developer cache")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected leaf root to become a single finding, got %+v", findings)
	}
	if findings[0].Path != slackCache {
		t.Fatalf("expected finding path %s, got %s", slackCache, findings[0].Path)
	}
	if findings[0].Name != "Slack cache" {
		t.Fatalf("expected friendly source name, got %s", findings[0].Name)
	}
}

func TestScanDeveloperRootsDiscoversCoreSimulatorDeviceCaches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	devicesRoot := filepath.Join(root, "Library", "Developer", "CoreSimulator", "Devices")
	deviceCacheRoot := filepath.Join(devicesRoot, "DEVICE-1", "data", "Library", "Caches")
	appCache := filepath.Join(deviceCacheRoot, "com.example.sim")
	if err := os.MkdirAll(appCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appCache, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanDeveloperRoots(context.Background(), stubAdapter{
		name: "darwin",
		roots: platform.CuratedRoots{
			Developer: []string{devicesRoot},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one discovered CoreSimulator cache finding, got %+v", findings)
	}
	if findings[0].Path != appCache {
		t.Fatalf("expected device cache child %s, got %+v", appCache, findings[0])
	}
	if findings[0].Source != "CoreSimulator device cache" {
		t.Fatalf("expected CoreSimulator source label, got %+v", findings[0])
	}
}

func TestScanRootEntriesSkipsBroadChildrenCoveredBySpecificCuratedRoots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cachesRoot := filepath.Join(root, "Library", "Caches")
	googleRoot := filepath.Join(cachesRoot, "Google")
	chromeCache := filepath.Join(googleRoot, "Chrome", "Default", "Code Cache")
	homebrewRoot := filepath.Join(cachesRoot, "Homebrew")
	homebrewDownloads := filepath.Join(homebrewRoot, "Downloads")
	if err := os.MkdirAll(chromeCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homebrewDownloads, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chromeCache, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homebrewDownloads, "archive.tgz"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := stubAdapter{
		name: "test",
		roots: platform.CuratedRoots{
			Temp:           []string{cachesRoot},
			Browser:        []string{chromeCache},
			PackageManager: []string{homebrewRoot, homebrewDownloads},
		},
	}
	findings, warnings, err := scanRootEntries(context.Background(), adapter, adapter.CuratedRoots().Temp, domain.CategoryTempFiles, domain.RiskSafe, domain.ActionTrash, "OS temporary data")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 0 {
		t.Fatalf("expected broad overlap roots to be skipped, got %+v", findings)
	}
}

func TestScanRootEntriesSkipsSIFTOwnedState(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "Library", "Caches")
	siftRoot := filepath.Join(cacheRoot, "sift")
	otherRoot := filepath.Join(cacheRoot, "OtherApp")
	if err := os.MkdirAll(siftRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siftRoot, "state.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherRoot, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := stubAdapter{
		name: "test",
		roots: platform.CuratedRoots{
			Temp: []string{cacheRoot},
		},
	}
	findings, _, err := scanRootEntries(context.Background(), adapter, adapter.CuratedRoots().Temp, domain.CategoryTempFiles, domain.RiskSafe, domain.ActionTrash, "OS temporary data")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Path != otherRoot {
		t.Fatalf("expected only non-SIFT cache root, got %+v", findings)
	}
}

func TestScanPurgeTargetsAllowsKnownProjectArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "app")
	artifact := filepath.Join(project, "node_modules")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeTargets(context.Background(), []string{artifact}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one purge finding, got %d", len(findings))
	}
	if findings[0].Category != domain.CategoryProjectArtifacts {
		t.Fatalf("expected project artifact category, got %s", findings[0].Category)
	}
}

func TestScanPurgeTargetsRejectsShallowArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Setenv("HOME", oldHome)
	}()
	findings, warnings, err := scanPurgeTargets(context.Background(), []string{filepath.Join(home, "node_modules")}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected shallow artifact to be rejected, got %d findings", len(findings))
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for shallow artifact")
	}
}

func TestScanPurgeTargetsMarksRecentArtifactsHighRisk(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "app")
	artifact := filepath.Join(project, "dist")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(artifact, "bundle.js")
	if err := os.WriteFile(file, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(file, now, now); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeTargets(context.Background(), []string{artifact}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one purge finding, got %d", len(findings))
	}
	if findings[0].Risk != domain.RiskHigh {
		t.Fatalf("expected high-risk recent artifact, got %s", findings[0].Risk)
	}
}

func TestScanPurgeTargetsAllowsWorkspaceMarkers(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := filepath.Join(root, "repo")
	artifact := filepath.Join(workspace, "node_modules")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "pnpm-workspace.yaml"), []byte("packages:\n  - apps/*\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeTargets(context.Background(), []string{artifact}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one purge finding, got %d", len(findings))
	}
	if !strings.Contains(findings[0].Source, "workspace") {
		t.Fatalf("expected workspace source label, got %q", findings[0].Source)
	}
}

func TestScanPurgeTargetsAllowsDartArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "mobile")
	artifact := filepath.Join(project, ".dart_tool")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "pubspec.yaml"), []byte("name: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeTargets(context.Background(), []string{artifact}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].Source, "Dart project") {
		t.Fatalf("expected Dart purge finding, got %+v", findings)
	}
}

func TestScanPurgeTargetsAllowsZigArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "native")
	artifact := filepath.Join(project, "zig-out")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "build.zig"), []byte("const std = @import(\"std\");\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "app"), []byte("payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeTargets(context.Background(), []string{artifact}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].Source, "Zig project") {
		t.Fatalf("expected Zig purge finding, got %+v", findings)
	}
}

func TestScanPurgeTargetsRejectsVendorNestedArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "repo")
	artifact := filepath.Join(project, "vendor", "demo", "node_modules")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeTargets(context.Background(), []string{artifact}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected vendor nested artifact to be rejected, got %+v", findings)
	}
	if len(warnings) == 0 || !strings.Contains(warnings[0], "protected vendor") {
		t.Fatalf("expected protected vendor warning, got %v", warnings)
	}
}

func TestScanPurgeDiscoveryFindsKnownArtifactsUnderSearchRoots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project := filepath.Join(root, "workspace", "app")
	artifact := filepath.Join(project, "node_modules")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeDiscovery(context.Background(), []string{filepath.Join(root, "workspace")}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one purge discovery finding, got %+v", findings)
	}
	if findings[0].Path != artifact {
		t.Fatalf("expected discovered artifact %s, got %+v", artifact, findings[0])
	}
}

func TestScanPurgeDiscoverySkipsShallowHomeArtifact(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)

	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "workspace", "demo")
	artifact := filepath.Join(project, "node_modules")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeDiscovery(context.Background(), []string{root}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected only nested project artifact, got %d findings (%v)", len(findings), warnings)
	}
	if findings[0].Path != artifact {
		t.Fatalf("unexpected discovered artifact: %+v", findings[0])
	}
}

func TestScanPurgeDiscoverySkipsProtectedVendorContainers(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	project := filepath.Join(workspace, "app")
	artifact := filepath.Join(project, "node_modules")
	vendorArtifact := filepath.Join(workspace, "vendor", "pkg", "node_modules")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(vendorArtifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "vendor", "pkg", "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorArtifact, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanPurgeDiscovery(context.Background(), []string{workspace}, stubAdapter{name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected only one non-vendor artifact, got %+v", findings)
	}
	if findings[0].Path != artifact {
		t.Fatalf("expected artifact %s, got %+v", artifact, findings[0])
	}
}

func TestScanAppSupportRootsNormalizesInstalledAppNames(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	supportRoot := filepath.Join(root, "Application Support")
	if err := os.MkdirAll(filepath.Join(supportRoot, "Keep-App"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(supportRoot, "Keep-App", "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(supportRoot, "Keep-App", "cache.bin"), oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanAppSupportRoots(context.Background(), stubAdapter{
		name: "test",
		roots: platform.CuratedRoots{
			AppSupport: []string{supportRoot},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 0 {
		t.Fatalf("expected installed app support root to be ignored, got %+v", findings)
	}
}

func TestScanAppSupportRootsSkipsSIFTOwnedState(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	supportRoot := filepath.Join(root, "Application Support")
	if err := os.MkdirAll(filepath.Join(supportRoot, "sift"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(supportRoot, "sift", "state.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(supportRoot, "sift", "state.bin"), oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanAppSupportRoots(context.Background(), stubAdapter{
		name: "test",
		roots: platform.CuratedRoots{
			AppSupport: []string{supportRoot},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 0 {
		t.Fatalf("expected SIFT-owned app support root to be ignored, got %+v", findings)
	}
}

func TestScanBrowserRootsOnlyReturnsSafeFirefoxCaches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	profiles := filepath.Join(root, "Profiles")
	cacheDir := filepath.Join(profiles, "default-release", "cache2")
	protectedLikeDir := filepath.Join(profiles, "default-release", "storage")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(protectedLikeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "entry.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protectedLikeDir, "identity.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanBrowserRoots(context.Background(), stubAdapter{name: "test"}, []string{profiles})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 {
		t.Fatalf("expected only safe Firefox cache findings, got %+v", findings)
	}
	if !strings.Contains(findings[0].Path, "cache2") {
		t.Fatalf("expected cache2 finding, got %+v", findings[0])
	}
}

func TestZipLooksLikeInstaller(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	archive := filepath.Join(root, "setup.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("Setup.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	ok, warnings := zipLooksLikeInstaller(archive)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if !ok {
		t.Fatal("expected installer-like zip to be detected")
	}
}

func TestZipLooksLikeInstallerRejectsGenericArchives(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	archive := filepath.Join(root, "docs.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	ok, warnings := zipLooksLikeInstaller(archive)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if ok {
		t.Fatal("expected generic zip to be rejected")
	}
}

func TestCapWarningsWithSummary(t *testing.T) {
	t.Parallel()
	warnings := []string{"a", "b", "c", "d"}
	capped := capWarningsWithSummary("root", warnings, 2)
	if len(capped) != 3 {
		t.Fatalf("expected 3 warnings after capping, got %d", len(capped))
	}
	if capped[2] != "root: 2 additional warnings suppressed" {
		t.Fatalf("unexpected summary warning: %v", capped)
	}
}

func TestScanSystemUpdateArtifactsFindsRosettaCache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rosettaCache := filepath.Join(root, "Library", "Caches", "com.apple.rosetta.update")
	if err := os.MkdirAll(rosettaCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rosettaCache, "cache.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanSystemUpdateArtifacts(context.Background(), stubAdapter{
		name: "test",
		roots: platform.CuratedRoots{
			SystemUpdate: []string{rosettaCache},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].Source, "Rosetta 2 user cache") {
		t.Fatalf("expected rosetta system update artifact, got %+v", findings)
	}
}

func TestCleanupSourceLabelCoversAppleMediaMaintenanceSurfaces(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"/Users/test/Library/Application Support/com.apple.wallpaper/aerials/videos":    "Aerial wallpaper videos",
		"/Users/test/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches":  "Wallpaper agent cache",
		"/Users/test/Library/Application Support/com.apple.idleassetsd":                 "Idle assets cache",
		"/Users/test/Library/Messages/Caches/Previews/Attachments":                      "Messages preview attachment cache",
		"/Users/test/Library/Caches/com.apple.QuickLook.thumbnailcache":                 "Quick Look thumbnail cache",
		"/Users/test/Library/Application Support/AddressBook/Sources/1234/Photos.cache": "Address Book photo cache",
	}

	for path, want := range cases {
		if got := cleanupSourceLabel(path, "fallback"); got != want {
			t.Fatalf("expected %q for %s, got %q", want, path, got)
		}
	}
}

func TestAnalyzePreviewsCachesDirectorySnapshots(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "cache")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "blob.bin"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	first := AnalyzePreviews([]string{dir})
	preview, ok := first[domain.NormalizePath(dir)]
	if !ok {
		t.Fatalf("expected preview for %s", dir)
	}
	if preview.Total != 2 || preview.Dirs != 1 || preview.Files != 1 {
		t.Fatalf("unexpected preview %+v", preview)
	}

	if err := os.WriteFile(filepath.Join(dir, "later.bin"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	second := AnalyzePreviews([]string{dir})
	preview = second[domain.NormalizePath(dir)]
	if preview.Total != 2 {
		t.Fatalf("expected cached preview to stay stable within ttl, got %+v", preview)
	}
}

func TestScanStaleLoginItemsFindsMissingHelperTarget(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific login item hint")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	launchAgents := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgents, 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>ProgramArguments</key><array><string>/Applications/Example.app/Contents/MacOS/helper</string></array></dict></plist>`
	path := filepath.Join(launchAgents, "com.example.stale.plist")
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, warnings, err := scanStaleLoginItems(context.Background(), stubAdapter{name: "darwin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 || findings[0].Path != path {
		t.Fatalf("expected stale login item finding, got %+v", findings)
	}
}

func TestScanOrphanedServicesFindsStaleLaunchDaemonAndHelper(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific service cleanup")
	}
	root := t.TempDir()
	launchDaemons := filepath.Join(root, "LaunchDaemons")
	helperRoot := filepath.Join(root, "PrivilegedHelperTools")
	if err := os.MkdirAll(launchDaemons, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(helperRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	plistPath := filepath.Join(launchDaemons, "com.example.helper.plist")
	plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>Program</key><string>/Applications/Example.app/Contents/Library/LaunchServices/helper</string></dict></plist>`
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	helperPath := filepath.Join(helperRoot, "com.example.helper")
	if err := os.WriteFile(helperPath, []byte("payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	originalRoots := systemServiceRoots
	originalHelpers := privilegedHelperToolsRoot
	systemServiceRoots = func() []string { return []string{launchDaemons} }
	privilegedHelperToolsRoot = func() string { return helperRoot }
	t.Cleanup(func() {
		systemServiceRoots = originalRoots
		privilegedHelperToolsRoot = originalHelpers
	})
	findings, warnings, err := scanOrphanedServices(context.Background(), stubAdapter{name: "darwin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 2 {
		t.Fatalf("expected plist and helper findings, got %+v", findings)
	}
}

func TestScanBrowserRootsFindsEdgeUpdaterOldVersions(t *testing.T) {
	original := currentAdapterIsProcessRunning
	currentAdapterIsProcessRunning = func(...string) bool { return false }
	defer func() { currentAdapterIsProcessRunning = original }()

	root := filepath.Join(t.TempDir(), "Library", "Application Support", "Microsoft", "EdgeUpdater", "apps", "msedge-stable")
	oldVersion := filepath.Join(root, "124.0.0")
	latestVersion := filepath.Join(root, "125.0.0")
	for _, dir := range []string{oldVersion, latestVersion} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "payload.bin"), []byte("payload"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	findings, warnings, err := scanBrowserRoots(context.Background(), stubAdapter{name: "darwin"}, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(findings) != 1 || findings[0].Path != oldVersion {
		t.Fatalf("expected only old edge updater version to be returned, got %+v", findings)
	}
}
