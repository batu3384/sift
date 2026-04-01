//go:build darwin

package platform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func TestDarwinDiscoverRemnantsFindsBundleIDAndAliasPaths(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	bundle := filepath.Join(home, "Applications", "Example.app")
	if err := os.MkdirAll(filepath.Join(bundle, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.example.app</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	container := filepath.Join(home, "Library", "Containers", "com.example.app")
	appSupport := filepath.Join(home, "Library", "Application Support", "Example")
	if err := os.MkdirAll(container, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appSupport, 0o755); err != nil {
		t.Fatal(err)
	}
	preferencesDir := filepath.Join(home, "Library", "Preferences")
	if err := os.MkdirAll(preferencesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(preferencesDir, "com.example.app.plist"), []byte("plist"), 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := darwinAdapter{home: home}
	paths, warnings, err := adapter.DiscoverRemnants(context.Background(), domain.AppEntry{
		Name:        "example",
		DisplayName: "Example",
		BundlePath:  bundle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	expected := map[string]bool{
		container:  false,
		appSupport: false,
		filepath.Join(home, "Library", "Preferences", "com.example.app.plist"): false,
	}
	for _, path := range paths {
		if _, ok := expected[path]; ok {
			expected[path] = true
		}
	}
	for path, seen := range expected {
		if !seen {
			t.Fatalf("expected path %s in remnant discovery, got %v", path, paths)
		}
	}
}

func TestDarwinListAppsFindsUninstallHelper(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	appsDir := filepath.Join(home, "Applications")
	bundle := filepath.Join(appsDir, "Example.app")
	helper := filepath.Join(appsDir, "Uninstall Example.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(helper, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := darwinAdapter{home: home}
	apps, err := adapter.ListApps(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	var example *domain.AppEntry
	for _, app := range apps {
		if app.DisplayName == "Example" {
			appCopy := app
			example = &appCopy
			break
		}
	}
	if example == nil {
		t.Fatalf("expected Example app in listing, got %v", apps)
	}
	if example.UninstallCommand != helper {
		t.Fatalf("expected uninstall helper %s, got %s", helper, example.UninstallCommand)
	}
	if example.Origin != "user application" {
		t.Fatalf("expected user application origin, got %s", example.Origin)
	}
}

func TestDarwinThirdPartyFirewallPrefersKnownApps(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "Applications", "LuLu.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := darwinThirdPartyFirewall(home); got != "LuLu" {
		t.Fatalf("expected LuLu firewall detection, got %q", got)
	}
}

func TestDarwinLoginItemsDiagnosticForModeSkipsInCiSafeMode(t *testing.T) {
	t.Setenv(envSiftTestMode, "ci-safe")

	diag := darwinLoginItemsDiagnosticForMode(context.Background())
	if diag.Name != "login_items" || diag.Status != "ok" || !strings.Contains(diag.Message, "ci-safe") {
		t.Fatalf("expected ci-safe login item diagnostic, got %+v", diag)
	}
}

func TestDarwinCuratedRootsIncludePlaywrightAndXcodeDeviceSupport(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	for _, dir := range []string{
		filepath.Join(home, "Library", "Caches", "com.teamviewer.host"),
		filepath.Join(home, "Library", "Caches", "com.todesk.1"),
		filepath.Join(home, "Library", "Caches", "com.sunlogin.client"),
		filepath.Join(home, "Library", "Caches", "com.autodesk.autocad"),
		filepath.Join(home, "Library", "Caches", "com.sketchup.SketchUp"),
		filepath.Join(home, "Library", "Caches", "com.airmail.direct"),
		filepath.Join(home, "Library", "Caches", "com.any.do.mac"),
		filepath.Join(home, "Library", "Caches", "com.eudic.mac"),
		filepath.Join(home, "Library", "Caches", "com.douyu.live"),
		filepath.Join(home, "Library", "Caches", "com.huya.live"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	adapter := darwinAdapter{home: home}
	roots := adapter.CuratedRoots()

	expectedDeveloper := map[string]bool{
		filepath.Join(home, ".android", "cache"):                                           false,
		filepath.Join(home, ".android", "build-cache"):                                     false,
		filepath.Join(home, ".pub-cache"):                                                  false,
		filepath.Join(home, ".expo"):                                                       false,
		filepath.Join(home, ".expo-shared"):                                                false,
		filepath.Join(home, "flutter", "bin", "cache"):                                     false,
		filepath.Join(home, ".cache", "ms-playwright"):                                     false,
		filepath.Join(home, ".cache", "puppeteer"):                                         false,
		filepath.Join(home, ".cache", "deno"):                                              false,
		filepath.Join(home, ".cache", "bazel"):                                             false,
		filepath.Join(home, ".grafana", "cache"):                                           false,
		filepath.Join(home, ".prometheus", "data", "wal"):                                  false,
		filepath.Join(home, ".jenkins", "workspace"):                                       false,
		filepath.Join(home, "Library", "Developer", "Xcode", "iOS DeviceSupport"):          false,
		filepath.Join(home, "Library", "Caches", "deno"):                                   false,
		filepath.Join(home, "Library", "Caches", "bazel"):                                  false,
		filepath.Join(home, "Library", "Caches", "com.konghq.insomnia"):                    false,
		filepath.Join(home, "Library", "Caches", "com.tinyapp.TablePlus"):                  false,
		filepath.Join(home, "Library", "Caches", "com.getpaw.Paw"):                         false,
		filepath.Join(home, "Library", "Caches", "com.charlesproxy.charles"):               false,
		filepath.Join(home, "Library", "Caches", "com.proxyman.NSProxy"):                   false,
		filepath.Join(home, "Library", "Caches", "com.github.GitHubDesktop"):               false,
		filepath.Join(home, "Library", "Caches", "com.microsoft.VSCode"):                   false,
		filepath.Join(home, "Library", "Application Support", "Code", "DawnGraphiteCache"): false,
		filepath.Join(home, "Library", "Application Support", "Code", "DawnWebGPUCache"):   false,
	}
	for _, root := range roots.Developer {
		if _, ok := expectedDeveloper[root]; ok {
			expectedDeveloper[root] = true
		}
	}
	for root, seen := range expectedDeveloper {
		if !seen {
			t.Fatalf("expected developer root %s in %+v", root, roots.Developer)
		}
	}

	dockerLog := filepath.Join(home, "Library", "Containers", "com.docker.docker", "Data", "log")
	foundLog := false
	for _, root := range roots.Logs {
		if root == dockerLog {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Fatalf("expected docker log root %s in %+v", dockerLog, roots.Logs)
	}

	expectedTemp := map[string]bool{
		filepath.Join(home, "Library", "Caches", "com.openai.chat"):                                         false,
		filepath.Join(home, "Library", "Caches", "com.anthropic.claudefordesktop"):                          false,
		filepath.Join(home, "Library", "Application Support", "Figma", "GPUCache"):                          false,
		filepath.Join(home, "Library", "Application Support", "Figma", "Service Worker", "CacheStorage"):    false,
		filepath.Join(home, "Library", "Application Support", "Postman", "GPUCache"):                        false,
		filepath.Join(home, "Library", "Application Support", "Postman", "Service Worker", "CacheStorage"):  false,
		filepath.Join(home, "Library", "Application Support", "Zed", "GPUCache"):                            false,
		filepath.Join(home, "Library", "Application Support", "Zed", "Service Worker", "CacheStorage"):      false,
		filepath.Join(home, "Library", "Application Support", "Claude", "GPUCache"):                         false,
		filepath.Join(home, "Library", "Application Support", "Claude", "DawnGraphiteCache"):                false,
		filepath.Join(home, "Library", "Application Support", "Claude", "DawnWebGPUCache"):                  false,
		filepath.Join(home, "Library", "Application Support", "Claude", "Service Worker", "CacheStorage"):   false,
		filepath.Join(home, "Library", "Application Support", "ChatGPT", "GPUCache"):                        false,
		filepath.Join(home, "Library", "Application Support", "ChatGPT", "Service Worker", "CacheStorage"):  false,
		filepath.Join(home, "Library", "Application Support", "Cursor", "GPUCache"):                         false,
		filepath.Join(home, "Library", "Application Support", "Cursor", "Service Worker", "CacheStorage"):   false,
		filepath.Join(home, "Library", "Application Support", "VSCodium", "GPUCache"):                       false,
		filepath.Join(home, "Library", "Application Support", "VSCodium", "Service Worker", "CacheStorage"): false,
		filepath.Join(home, "Library", "Application Support", "Slack", "GPUCache"):                          false,
		filepath.Join(home, "Library", "Application Support", "Slack", "Service Worker", "CacheStorage"):    false,
		filepath.Join(home, "Library", "Application Support", "Discord", "GPUCache"):                        false,
		filepath.Join(home, "Library", "Application Support", "Discord", "Service Worker", "CacheStorage"):  false,
		filepath.Join(home, "Library", "Application Support", "Legcord", "GPUCache"):                        false,
		filepath.Join(home, "Library", "Application Support", "Microsoft", "Teams", "Cache"):                false,
		filepath.Join(home, "Library", "Application Support", "Microsoft", "Teams", "Code Cache"):           false,
		filepath.Join(home, "Library", "Application Support", "Microsoft", "Teams", "GPUCache"):             false,
		filepath.Join(home, "Library", "Application Support", "Steam", "htmlcache"):                         false,
		filepath.Join(home, "Library", "Application Support", "Steam", "appcache"):                          false,
		filepath.Join(home, "Library", "Application Support", "Steam", "depotcache"):                        false,
		filepath.Join(home, "Library", "Application Support", "Steam", "steamapps", "shadercache"):          false,
		filepath.Join(home, "Library", "Application Support", "Battle.net", "Cache"):                        false,
		filepath.Join(home, "Library", "Caches", "com.mongodb.compass"):                                     false,
		filepath.Join(home, "Library", "Application Support", "MongoDB Compass", "GPUCache"):                false,
		filepath.Join(home, "Library", "Application Support", "RedisInsight", "Code Cache"):                 false,
		filepath.Join(home, "Library", "Caches", "net.shinyfrog.bear"):                                      false,
		filepath.Join(home, "Library", "Application Support", "Logseq", "Service Worker", "CacheStorage"):   false,
		filepath.Join(home, "Library", "Application Support", "Riot Client", "GPUCache"):                    false,
		filepath.Join(home, "Library", "Application Support", "Lark", "Code Cache"):                         false,
		filepath.Join(home, "Library", "Application Support", "DingTalk", "GPUCache"):                       false,
		filepath.Join(home, "Library", "Caches", "com.anydesk.anydesk"):                                     false,
		filepath.Join(home, "Library", "Caches", "com.gog.galaxy"):                                          false,
		filepath.Join(home, "Library", "Caches", "com.ea.app"):                                              false,
		filepath.Join(home, "Library", "Caches", "com.tencent.meeting"):                                     false,
		filepath.Join(home, "Library", "Caches", "com.tencent.WeWorkMac"):                                   false,
		filepath.Join(home, "Library", "Caches", "com.teamviewer.host"):                                     false,
		filepath.Join(home, "Library", "Caches", "com.todesk.1"):                                            false,
		filepath.Join(home, "Library", "Caches", "com.sunlogin.client"):                                     false,
		filepath.Join(home, "Library", "Caches", "com.airmail.direct"):                                      false,
		filepath.Join(home, "Library", "Caches", "com.any.do.mac"):                                          false,
		filepath.Join(home, "Library", "Caches", "cx.c3.theunarchiver"):                                     false,
		filepath.Join(home, "Library", "Caches", "com.youdao.YoudaoDict"):                                   false,
		filepath.Join(home, "Library", "Caches", "com.eudic.mac"):                                           false,
		filepath.Join(home, "Library", "Caches", "com.bob-build.Bob"):                                       false,
		filepath.Join(home, "Library", "Caches", "com.tw93.MiaoYan"):                                        false,
		filepath.Join(home, "Library", "Caches", "com.flomoapp.mac"):                                        false,
		filepath.Join(home, "Library", "Application Support", "Quark", "Cache", "videoCache"):               false,
		filepath.Join(home, "Library", "Caches", "com.maxon.cinema4d"):                                      false,
		filepath.Join(home, "Library", "Caches", "com.autodesk.autocad"):                                    false,
		filepath.Join(home, "Library", "Caches", "com.sketchup.SketchUp"):                                   false,
		filepath.Join(home, "Library", "Caches", "com.netease.163music"):                                    false,
		filepath.Join(home, "Library", "Caches", "com.tencent.QQMusic"):                                     false,
		filepath.Join(home, "Library", "Caches", "com.kugou.mac"):                                           false,
		filepath.Join(home, "Library", "Caches", "com.kuwo.mac"):                                            false,
		filepath.Join(home, "Library", "Caches", "com.iqiyi.player"):                                        false,
		filepath.Join(home, "Library", "Caches", "com.tencent.tenvideo"):                                    false,
		filepath.Join(home, "Library", "Caches", "tv.danmaku.bili"):                                         false,
		filepath.Join(home, "Library", "Caches", "com.douyu.live"):                                          false,
		filepath.Join(home, "Library", "Caches", "com.huya.live"):                                           false,
		filepath.Join(home, "Library", "Caches", "com.reincubate.camo"):                                     false,
		filepath.Join(home, "Library", "Caches", "com.xnipapp.xnip"):                                        false,
		filepath.Join(home, "Library", "Caches", "org.m0k.transmission"):                                    false,
		filepath.Join(home, "Library", "Caches", "com.qbittorrent.qBittorrent"):                             false,
		filepath.Join(home, "Library", "Caches", "notion.id"):                                               false,
		filepath.Join(home, "Library", "Caches", "md.obsidian"):                                             false,
		filepath.Join(home, "Library", "Caches", "com.runningwithcrayons.Alfred"):                           false,
		filepath.Join(home, "Library", "Caches", "com.microsoft.teams2"):                                    false,
		filepath.Join(home, "Library", "Caches", "us.zoom.xos"):                                             false,
		filepath.Join(home, "Library", "Caches", "ru.keepcoder.Telegram"):                                   false,
		filepath.Join(home, "Library", "Caches", "com.tencent.xinWeChat"):                                   false,
		filepath.Join(home, "Library", "Caches", "com.skype.skype"):                                         false,
		filepath.Join(home, "Library", "Caches", "net.whatsapp.WhatsApp"):                                   false,
		filepath.Join(home, "Library", "Caches", "com.todoist.mac.Todoist"):                                 false,
		filepath.Join(home, "Library", "Caches", "com.valvesoftware.steam"):                                 false,
		filepath.Join(home, "Library", "Caches", "com.epicgames.EpicGamesLauncher"):                         false,
		filepath.Join(home, "Library", "Caches", "com.blizzard.Battle.net"):                                 false,
		filepath.Join(home, "Library", "Caches", "com.colliderli.iina"):                                     false,
		filepath.Join(home, "Library", "Caches", "org.videolan.vlc"):                                        false,
		filepath.Join(home, "Library", "Caches", "io.mpv"):                                                  false,
		filepath.Join(home, "Library", "Caches", "tv.plex.player.desktop"):                                  false,
		filepath.Join(home, "Library", "Caches", "com.readdle.smartemail-Mac"):                              false,
		filepath.Join(home, "Library", "Application Support", "com.bohemiancoding.sketch3", "cache"):        false,
		filepath.Join(home, "Library", "Application Support", "Adobe", "Common", "Media Cache Files"):       false,
	}
	for _, root := range roots.Temp {
		if _, ok := expectedTemp[root]; ok {
			expectedTemp[root] = true
		}
	}
	for root, seen := range expectedTemp {
		if !seen {
			t.Fatalf("expected temp root %s in %+v", root, roots.Temp)
		}
	}

	expectedLogs := map[string]bool{
		filepath.Join(home, "Library", "Logs", "Figma"):                                     false,
		filepath.Join(home, "Library", "Logs", "Postman"):                                   false,
		filepath.Join(home, "Library", "Logs", "Zed"):                                       false,
		filepath.Join(home, "Library", "Application Support", "Figma", "logs"):              false,
		filepath.Join(home, "Library", "Application Support", "Postman", "logs"):            false,
		filepath.Join(home, "Library", "Application Support", "Zed", "logs"):                false,
		filepath.Join(home, "Library", "Logs", "ChatGPT"):                                   false,
		filepath.Join(home, "Library", "Application Support", "Claude", "logs"):             false,
		filepath.Join(home, "Library", "Application Support", "ChatGPT", "logs"):            false,
		filepath.Join(home, "Library", "Application Support", "Cursor", "logs"):             false,
		filepath.Join(home, "Library", "Application Support", "VSCodium", "logs"):           false,
		filepath.Join(home, "Library", "Application Support", "Legcord", "logs"):            false,
		filepath.Join(home, "Library", "Application Support", "Logseq", "logs"):             false,
		filepath.Join(home, "Library", "Application Support", "Riot Client", "logs"):        false,
		filepath.Join(home, "Library", "Application Support", "minecraft", "logs"):          false,
		filepath.Join(home, "Library", "Application Support", "lunarclient", "logs"):        false,
		filepath.Join(home, "Library", "Application Support", "Lark", "logs"):               false,
		filepath.Join(home, "Library", "Application Support", "DingTalk", "logs"):           false,
		filepath.Join(home, "Library", "Application Support", "Microsoft", "Teams", "logs"): false,
		filepath.Join(home, "Library", "Application Support", "Steam", "logs"):              false,
	}
	for _, root := range roots.Logs {
		if _, ok := expectedLogs[root]; ok {
			expectedLogs[root] = true
		}
	}
	for root, seen := range expectedLogs {
		if !seen {
			t.Fatalf("expected log root %s in %+v", root, roots.Logs)
		}
	}
}

func TestDarwinCuratedRootsIncludeExpandedInstallerAndProtectionSurfaces(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	adapter := darwinAdapter{home: home}
	roots := adapter.CuratedRoots()
	for _, expected := range []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Public"),
		filepath.Join("/Users", "Shared"),
		filepath.Join("/Users", "Shared", "Downloads"),
		filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs", "Downloads"),
		filepath.Join(home, "Library", "Application Support", "Telegram Desktop"),
		filepath.Join(home, "Downloads", "Telegram Desktop"),
	} {
		found := false
		for _, root := range roots.Installer {
			if root == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected installer root %s in %+v", expected, roots.Installer)
		}
	}
	protected := adapter.ProtectedPaths()
	for _, expected := range []string{
		filepath.Join(home, "Library", "Application Support", "Raycast"),
		filepath.Join(home, "Library", "Application Support", "Alfred"),
		filepath.Join(home, "Library", "Containers", "com.raycast.macos.BrowserExtension"),
		filepath.Join(home, "Library", "Mobile Documents"),
	} {
		found := false
		for _, path := range protected {
			if path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected protected path %s in %+v", expected, protected)
		}
	}
}

func TestDarwinMaintenanceTasksIncludePremiumOptimizeTasks(t *testing.T) {
	t.Parallel()
	adapter := darwinAdapter{home: t.TempDir()}
	tasks := adapter.MaintenanceTasks(context.Background())
	seen := map[string]domain.MaintenanceTask{}
	for _, task := range tasks {
		seen[task.ID] = task
	}
	for _, id := range []string{
		"macos.optimize.preferences-temp",
		"macos.optimize.font-cache",
		"macos.optimize.route-cache",
		"macos.optimize.arp-cache",
	} {
		task, ok := seen[id]
		if !ok {
			t.Fatalf("expected optimize task %s in %+v", id, tasks)
		}
		if strings.TrimSpace(task.Phase) == "" || strings.TrimSpace(task.EstimatedImpact) == "" {
			t.Fatalf("expected optimize task %s to carry orchestration metadata, got %+v", id, task)
		}
	}
}

func TestDarwinMaintenanceTasksIncludeSQLiteVacuumTaskForEligibleDatabase(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not available")
	}
	darwinProcessRunningMu.Lock()
	original := darwinProcessRunning
	darwinProcessRunning = func(names ...string) bool { return false }
	darwinProcessRunningMu.Unlock()
	defer func() {
		darwinProcessRunningMu.Lock()
		darwinProcessRunning = original
		darwinProcessRunningMu.Unlock()
	}()

	home := t.TempDir()
	dbPath := filepath.Join(home, "Library", "Safari", "History.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, append([]byte("SQLite format 3\x00"), []byte("payload")...), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := darwinAdapter{home: home}
	tasks := adapter.MaintenanceTasks(context.Background())
	for _, task := range tasks {
		if task.ID != "macos.optimize.sqlite-safari-history" {
			continue
		}
		if task.Action != domain.ActionCommand {
			t.Fatalf("expected sqlite maintenance task to be managed command, got %+v", task)
		}
		if task.CommandArgs[0] != dbPath {
			t.Fatalf("expected sqlite task to target %s, got %+v", dbPath, task.CommandArgs)
		}
		return
	}
	t.Fatalf("expected sqlite vacuum task in %+v", tasks)
}

func TestDarwinCuratedRootsIncludeRecentItemsAndSystemUpdateSurfaces(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	adapter := darwinAdapter{home: home}
	roots := adapter.CuratedRoots()
	for _, expected := range []string{
		filepath.Join(home, "Library", "Application Support", "com.apple.sharedfilelist", "com.apple.LSSharedFileList.RecentApplications.sfl2"),
		filepath.Join(home, "Library", "Preferences", "com.apple.recentitems.plist"),
		filepath.Join(home, "Library", "Updates"),
		"/Library/Updates",
		"/Library/Apple/usr/share/rosetta/rosetta_update_bundle",
		filepath.Join(home, "Library", "Caches", "com.apple.rosetta.update"),
		filepath.Join(home, "Library", "Caches", "com.apple.amp.mediasevicesd"),
		filepath.Join(home, "Library", "Application Support", "Microsoft", "EdgeUpdater", "apps", "msedge-stable"),
	} {
		found := false
		for _, bucket := range [][]string{roots.RecentItems, roots.SystemUpdate, roots.Browser} {
			for _, root := range bucket {
				if root == expected {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Fatalf("expected curated root %s in recent/system/browser surfaces", expected)
		}
	}
}

func TestDarwinCuratedRootsIncludeAppleMediaMaintenanceSurfaces(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	adapter := darwinAdapter{home: home}
	roots := adapter.CuratedRoots()
	for _, expected := range []string{
		filepath.Join(home, "Library", "Application Support", "com.apple.wallpaper", "aerials", "videos"),
		filepath.Join(home, "Library", "Containers", "com.apple.wallpaper.agent", "Data", "Library", "Caches"),
		filepath.Join(home, "Library", "Application Support", "com.apple.idleassetsd"),
		"/Library/Application Support/com.apple.idleassetsd/Customer",
		filepath.Join(home, "Library", "Messages", "StickerCache"),
		filepath.Join(home, "Library", "Messages", "Caches", "Previews", "Attachments"),
		filepath.Join(home, "Library", "Messages", "Caches", "Previews", "StickerCache"),
		filepath.Join(home, "Library", "Caches", "com.apple.QuickLook.thumbnailcache"),
		filepath.Join(home, "Library", "Caches", "Quick Look"),
		filepath.Join(home, "Library", "Autosave Information"),
		filepath.Join(home, "Library", "IdentityCaches"),
		filepath.Join(home, "Library", "Suggestions"),
		filepath.Join(home, "Library", "Calendars", "Calendar Cache"),
	} {
		found := false
		for _, root := range roots.Temp {
			if root == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected temp root %s in %+v", expected, roots.Temp)
		}
	}
}

func TestDarwinDiagnosticParsers(t *testing.T) {
	t.Parallel()
	if diag := darwinFileVaultDiagnostic("FileVault is On.", nil); diag.Status != "ok" {
		t.Fatalf("expected filevault ok, got %+v", diag)
	}
	if diag := darwinGatekeeperDiagnostic("assessments disabled", nil); diag.Status != "warn" {
		t.Fatalf("expected gatekeeper warn, got %+v", diag)
	}
	if diag := darwinSIPDiagnostic("System Integrity Protection status: enabled.", nil); diag.Status != "ok" {
		t.Fatalf("expected sip ok, got %+v", diag)
	}
	if diag := darwinSoftwareUpdateDiagnostic("Software Update Tool\n\n* Label: macOS 15.4", nil); diag.Status != "warn" {
		t.Fatalf("expected macos update warn, got %+v", diag)
	}
	if diag := darwinHomebrewUpdatesDiagnostic("wget\nnode", nil); diag.Status != "warn" {
		t.Fatalf("expected brew update warn, got %+v", diag)
	}
	if diag := darwinHomebrewHealthDiagnostic("Your system is ready to brew.", nil); diag.Status != "ok" {
		t.Fatalf("expected brew health ok, got %+v", diag)
	}
	if diag := darwinHomebrewHealthDiagnostic("Warning: Some formulae are kegs only.", nil); diag.Status != "warn" {
		t.Fatalf("expected brew health warn, got %+v", diag)
	}
}

func TestDarwinMaintenanceTasksIncludeBrewCleanupAndModernLaunchServices(t *testing.T) {
	originalLookPath := execLookPathDarwin
	defer func() { execLookPathDarwin = originalLookPath }()
	execLookPathDarwin = func(file string) (string, error) {
		switch file {
		case "brew":
			return "/opt/homebrew/bin/brew", nil
		default:
			return "", exec.ErrNotFound
		}
	}

	adapter := darwinAdapter{home: t.TempDir()}
	tasks := adapter.dynamicMaintenanceTasks()

	foundBrewCleanup := false
	for _, task := range tasks {
		if task.ID == "macos.optimize.brew-cleanup" {
			foundBrewCleanup = true
			if task.CommandPath != "/opt/homebrew/bin/brew" || len(task.CommandArgs) != 1 || task.CommandArgs[0] != "cleanup" {
				t.Fatalf("unexpected brew cleanup task: %+v", task)
			}
		}
	}
	if !foundBrewCleanup {
		t.Fatal("expected brew cleanup maintenance task")
	}

	foundLaunchServices := false
	for _, task := range adapter.MaintenanceTasks(context.Background()) {
		if task.ID != "macos.optimize.launchservices" {
			continue
		}
		foundLaunchServices = true
		if slices.Contains(task.CommandArgs, "-kill") {
			t.Fatalf("expected launchservices task to avoid deprecated -kill flag, got %+v", task)
		}
		if !slices.Contains(task.CommandArgs, "-f") {
			t.Fatalf("expected launchservices task to force rebuild, got %+v", task)
		}
	}
	if !foundLaunchServices {
		t.Fatal("expected launchservices maintenance task")
	}
}
