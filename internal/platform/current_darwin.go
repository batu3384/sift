//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/batu3384/sift/internal/domain"
)

type darwinAdapter struct {
	home string
}

var execLookPathDarwin = exec.LookPath

func Current() Adapter {
	home, _ := os.UserHomeDir()
	return darwinAdapter{home: home}
}

func (d darwinAdapter) Name() string { return "darwin" }

func (d darwinAdapter) IsFileInUse(ctx context.Context, path string) bool {
	lsofPath, err := execLookPathDarwin("lsof")
	if err != nil {
		// Log warning but assume file is not in use (safe default)
		fmt.Fprintf(os.Stderr, "warning: lsof not found, cannot check if file is in use: %v\n", err)
		return false
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return exec.CommandContext(checkCtx, lsofPath, "-F", "n", "--", path).Run() == nil
}

func (d darwinAdapter) ListApps(_ context.Context, allowAdmin bool) ([]domain.AppEntry, error) {
	homebrewCasks := d.homebrewCaskNames()
	setappRoot := filepath.Join(d.home, "Library", "Application Support", "Setapp", "Applications")
	roots := []struct {
		path          string
		origin        string
		requiresAdmin bool
	}{
		{path: "/Applications", origin: "system application", requiresAdmin: true},
		{path: filepath.Join(d.home, "Applications"), origin: "user application", requiresAdmin: false},
		{path: setappRoot, origin: "setapp", requiresAdmin: false},
	}

	// Filter roots based on admin permissions
	if !allowAdmin {
		var filtered []struct {
			path          string
			origin        string
			requiresAdmin bool
		}
		for _, root := range roots {
			if !root.requiresAdmin {
				filtered = append(filtered, root)
			}
		}
		roots = filtered
	}

	var apps []domain.AppEntry
	seen := map[string]struct{}{}

	// Helper function to scan a directory for .app bundles
	scanDir := func(dirPath string, origin string, requiresAdmin bool) {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".app") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".app")
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			bundleOrigin := origin
			if dirPath != setappRoot {
				if _, ok := homebrewCasks[strings.ToLower(name)]; ok {
					bundleOrigin = "homebrew cask"
				}
			}
			bundlePath := filepath.Join(dirPath, entry.Name())
			apps = append(apps, domain.AppEntry{
				Name:        strings.ToLower(name),
				DisplayName: name,
				BundlePath:  bundlePath,
				SupportPaths: []string{
					filepath.Join(d.home, "Library", "Application Support", name),
					filepath.Join(d.home, "Library", "Caches", name),
					filepath.Join(d.home, "Library", "Logs", name),
					filepath.Join(d.home, "Library", "Preferences", "com."+strings.ToLower(strings.ReplaceAll(name, " ", ""))+".plist"),
				},
				Origin:           bundleOrigin,
				RequiresAdmin:    requiresAdmin,
				UninstallHint:    "Review bundle and support files before deletion.",
				UninstallCommand: d.findUninstallHelper(bundlePath, name),
				LastModified:     info.ModTime(),
			})
		}
	}

	// Scan each root directory
	for _, root := range roots {
		// First scan the root directory itself
		scanDir(root.path, root.origin, root.requiresAdmin)

		// Also scan subdirectories (e.g., /Applications/Python 3.13/IDLE.app)
		// This handles apps installed in subfolders like Python, Microsoft Office, etc.
		if entries, err := os.ReadDir(root.path); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				subDir := filepath.Join(root.path, entry.Name())
				// Check if subdirectory might contain .app files
				if subEntries, err := os.ReadDir(subDir); err == nil {
					for _, subEntry := range subEntries {
						if subEntry.IsDir() && strings.HasSuffix(subEntry.Name(), ".app") {
							// This is a nested app - scan it
							scanDir(subDir, root.origin, root.requiresAdmin)
						}
					}
				}
			}
		}
	}
	return apps, nil
}

func (d darwinAdapter) homebrewCaskNames() map[string]struct{} {
	roots := []string{"/opt/homebrew/Caskroom", "/usr/local/Caskroom"}
	names := map[string]struct{}{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			names[strings.ToLower(strings.TrimSpace(entry.Name()))] = struct{}{}
		}
	}
	return names
}

func (d darwinAdapter) DiscoverRemnants(ctx context.Context, app domain.AppEntry) ([]string, []string, error) {
	aliases := aliasCandidates(app)
	var candidates []string
	candidates = append(candidates, app.SupportPaths...)

	bundleID := d.bundleIdentifier(ctx, app.BundlePath)
	if bundleID != "" {
		candidates = append(candidates,
			filepath.Join(d.home, "Library", "Preferences", bundleID+".plist"),
			filepath.Join(d.home, "Library", "Containers", bundleID),
			filepath.Join(d.home, "Library", "Saved Application State", bundleID+".savedState"),
			filepath.Join(d.home, "Library", "HTTPStorages", bundleID),
			filepath.Join(d.home, "Library", "WebKit", bundleID),
			filepath.Join(d.home, "Library", "Application Scripts", bundleID),
			filepath.Join(d.home, "Library", "Group Containers", "group."+bundleID),
			filepath.Join(d.home, "Library", "Caches", bundleID),
			filepath.Join(d.home, "Library", "Logs", bundleID),
			filepath.Join(d.home, "Library", "Application Support", bundleID),
		)
	}

	searchRoots := []string{
		filepath.Join(d.home, "Library", "Application Support"),
		filepath.Join(d.home, "Library", "Caches"),
		filepath.Join(d.home, "Library", "Logs"),
		filepath.Join(d.home, "Library", "Containers"),
		filepath.Join(d.home, "Library", "Saved Application State"),
	}
	var warnings []string
	for _, root := range searchRoots {
		matches, rootWarnings := exactNameMatches(root, aliases)
		candidates = append(candidates, matches...)
		warnings = append(warnings, rootWarnings...)
	}
	if bundleID != "" {
		for _, root := range []string{
			filepath.Join(d.home, "Library", "LaunchAgents"),
			"/Library/LaunchAgents",
			"/Library/LaunchDaemons",
			filepath.Join(d.home, "Library", "Preferences", "ByHost"),
		} {
			matches, rootWarnings := prefixMatches(root, bundleID)
			candidates = append(candidates, matches...)
			warnings = append(warnings, rootWarnings...)
		}
	}
	return existingUniquePaths(candidates), warnings, nil
}

func (d darwinAdapter) MaintenanceTasks(_ context.Context) []domain.MaintenanceTask {
	tasks := []domain.MaintenanceTask{
		{
			ID:              "macos.optimize.dns-flush",
			Title:           "Flush DNS cache",
			Description:     "Refreshes cached DNS resolution data using the system DNS cache tool.",
			Risk:            domain.RiskReview,
			Capability:      "dscacheutil available",
			Phase:           "repair",
			EstimatedImpact: "Refreshes resolver state without touching user data.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/sudo",
			CommandArgs:     []string{"/usr/bin/dscacheutil", "-flushcache"},
			TimeoutSeconds:  10,
			RequiresAdmin:   true,
			Verification:    []string{"Retry the affected hostname lookup", "If discovery is still stale, continue with mDNS reload"},
			Steps: []string{
				"DNS cache refreshes immediately",
				"Use when hostname resolution feels stale",
			},
		},
		{
			ID:              "macos.optimize.mdns-reload",
			Title:           "Reload mDNSResponder",
			Description:     "Signals the mDNSResponder service to refresh Bonjour and resolver state.",
			Risk:            domain.RiskReview,
			Capability:      "mDNSResponder present",
			Phase:           "repair",
			EstimatedImpact: "Refreshes Bonjour and resolver helpers without a reboot.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/sudo",
			CommandArgs:     []string{"/usr/bin/killall", "-HUP", "mDNSResponder"},
			TimeoutSeconds:  10,
			RequiresAdmin:   true,
			Verification:    []string{"Retry local discovery or AirDrop device lookup", "If DNS is still stale, inspect network routing cache"},
			Steps: []string{
				"Resolver state reloads without a reboot",
				"Use after DNS cache flushes or stale Bonjour discovery",
			},
		},
		{
			ID:              "macos.optimize.quicklook",
			Title:           "Reset Quick Look thumbnail cache",
			Description:     "Removes stale Quick Look previews so macOS rebuilds thumbnails on demand.",
			Risk:            domain.RiskSafe,
			Phase:           "cleanup",
			EstimatedImpact: "Reclaims stale preview cache and forces fresh thumbnails.",
			Action:          domain.ActionTrash,
			Paths: []string{
				filepath.Join(d.home, "Library", "Caches", "com.apple.QuickLook.thumbnailcache"),
			},
			Verification: []string{"Reopen Finder and confirm thumbnails rebuild", "Run Quick Look rebuild if previews still look stale"},
			Steps: []string{
				"Preview thumbnails will rebuild automatically",
				"Reopen Finder windows if previews look stale",
			},
		},
		{
			ID:              "macos.optimize.quicklook-rebuild",
			Title:           "Rebuild Quick Look services",
			Description:     "Asks Quick Look to refresh its cache and service state without deleting user data.",
			Risk:            domain.RiskSafe,
			Capability:      "qlmanage available",
			Phase:           "verify",
			EstimatedImpact: "Refreshes preview helpers in place after cache cleanup.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/qlmanage",
			CommandArgs:     []string{"-r", "cache"},
			TimeoutSeconds:  10,
			Verification:    []string{"Select a file in Finder and tap space to validate preview", "If preview remains stale, reload Quick Look daemons"},
			Steps: []string{
				"Quick Look cache rebuilds in place",
				"Use with stale previews or thumbnail generation issues",
			},
		},
		{
			ID:             "macos.optimize.quicklook-reload",
			Title:          "Reload Quick Look daemons",
			Description:    "Reloads Quick Look helper services after cache resets.",
			Risk:           domain.RiskSafe,
			Capability:     "qlmanage available",
			Action:         domain.ActionCommand,
			CommandPath:    "/usr/bin/qlmanage",
			CommandArgs:    []string{"-r"},
			TimeoutSeconds: 10,
			Steps: []string{
				"Quick Look helper processes reload automatically",
				"Use after cache resets if previews still look stale",
			},
		},
		{
			ID:              "macos.optimize.iconservices",
			Title:           "Reset Finder icon services cache",
			Description:     "Clears stale icon cache files so Finder can regenerate them cleanly.",
			Risk:            domain.RiskSafe,
			Phase:           "cleanup",
			EstimatedImpact: "Forces Finder icons to rebuild from source metadata.",
			Action:          domain.ActionTrash,
			Paths: []string{
				filepath.Join(d.home, "Library", "Caches", "com.apple.iconservices"),
				filepath.Join(d.home, "Library", "Caches", "com.apple.iconservices.store"),
			},
			Verification: []string{"Refresh Finder windows", "Restart Dock if stale icons remain"},
			Steps: []string{
				"Finder icons rebuild after reopen or sign out",
				"Use if file or app icons look stale",
			},
		},
		{
			ID:                "macos.optimize.saved-state",
			Title:             "Clear saved application state",
			Description:       "Removes saved state bundles for apps that can safely regenerate them on next launch.",
			Risk:              domain.RiskReview,
			Phase:             "cleanup",
			EstimatedImpact:   "Reclaims stale reopen/session state from crashed or sticky apps.",
			SuggestedByChecks: []string{"check.disk_pressure"},
			Action:            domain.ActionTrash,
			PathGlobs: []string{
				filepath.Join(d.home, "Library", "Saved Application State", "*.savedState"),
			},
			Verification: []string{"Relaunch the affected app", "Confirm stale windows or sessions no longer reopen"},
			Steps: []string{
				"Apps will recreate saved state on next launch",
				"Use when windows reopen incorrectly or stale sessions persist",
			},
		},
		{
			ID:                "macos.optimize.spotlight",
			Title:             "Reset Spotlight helper cache",
			Description:       "Clears stale Spotlight UI cache data without touching the search index itself.",
			Risk:              domain.RiskSafe,
			Phase:             "cleanup",
			EstimatedImpact:   "Refreshes Spotlight UI caches while keeping the index intact.",
			SuggestedByChecks: []string{"check.disk_pressure"},
			Action:            domain.ActionTrash,
			Paths: []string{
				filepath.Join(d.home, "Library", "Caches", "com.apple.Spotlight"),
			},
			Verification: []string{"Open Spotlight and repeat the stale search", "If results remain stale, escalate to Spotlight rebuild"},
			Steps: []string{
				"Spotlight cache data rebuilds automatically",
				"Use when Spotlight panels look stale or sluggish",
			},
		},
		{
			ID:              "macos.optimize.launchservices",
			Title:           "Repair LaunchServices registry",
			Description:     "Refreshes LaunchServices so removed or moved apps stop appearing in Open With and Launchpad.",
			Risk:            domain.RiskReview,
			Capability:      "lsregister available",
			Phase:           "repair",
			EstimatedImpact: "Rebuilds app registration metadata for Open With and Launchpad.",
			Action:          domain.ActionCommand,
			CommandPath:     "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister",
			CommandArgs:     []string{"-r", "-f", "-domain", "local", "-domain", "system", "-domain", "user"},
			TimeoutSeconds:  20,
			Verification:    []string{"Retry Open With on the affected file", "Check Launchpad for removed app entries"},
			Steps: []string{
				"LaunchServices refreshes app registration metadata",
				"Use after uninstalling apps that still appear in Open With or Launchpad",
			},
		},
		{
			ID:              "macos.optimize.dock-refresh",
			Title:           "Refresh Dock",
			Description:     "Restarts the Dock process so stale icons, recents and pinned apps refresh immediately.",
			Risk:            domain.RiskSafe,
			Capability:      "Dock process available",
			Phase:           "verify",
			EstimatedImpact: "Refreshes Dock surfaces after app or icon cleanup.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/killall",
			CommandArgs:     []string{"Dock"},
			TimeoutSeconds:  10,
			Verification:    []string{"Confirm stale Dock icons or recents are gone"},
			Steps: []string{
				"Dock restarts automatically",
				"Use when app icons or recents remain stale after cleanup",
			},
		},
		{
			ID:                "macos.optimize.spotlight-rebuild",
			Title:             "Rebuild Spotlight index",
			Description:       "Requests a full Spotlight reindex for the startup volume when search remains slow or stale.",
			Risk:              domain.RiskHigh,
			Capability:        "mdutil available",
			Phase:             "repair",
			EstimatedImpact:   "Starts a full reindex pass for persistent Spotlight drift.",
			Action:            domain.ActionCommand,
			CommandPath:       "/usr/bin/sudo",
			CommandArgs:       []string{"/usr/bin/mdutil", "-E", "/"},
			TimeoutSeconds:    20,
			RequiresAdmin:     true,
			SuggestedByChecks: []string{"check.disk_pressure", "check.health_score"},
			Verification:      []string{"Run `mdutil -s /` to confirm indexing resumes", "Expect indexing to continue in the background"},
			Steps: []string{
				"Spotlight indexing restarts in the background",
				"Search may remain busy for up to several hours on large disks",
			},
		},
		{
			ID:              "macos.optimize.permissions",
			Title:           "Repair user directory permissions",
			Description:     "Runs diskutil resetUserPermissions for the current user when file ownership or access looks stale.",
			Risk:            domain.RiskHigh,
			Capability:      "diskutil available",
			Phase:           "repair",
			EstimatedImpact: "Repairs permission drift across the user home directory.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/sudo",
			CommandArgs:     []string{"/usr/sbin/diskutil", "resetUserPermissions", "/", strconv.Itoa(os.Getuid())},
			TimeoutSeconds:  30,
			RequiresAdmin:   true,
			Verification:    []string{"Retry the file access or save failure that triggered the repair"},
			Steps: []string{
				"Repairs ownership and permission drift in the user home directory",
				"Use when file access problems persist after app or cache cleanup",
			},
		},
		{
			ID:              "macos.optimize.bluetooth-reset",
			Title:           "Restart Bluetooth daemon",
			Description:     "Restarts bluetoothd so stale device and pairing state can recover without a reboot.",
			Risk:            domain.RiskReview,
			Capability:      "bluetoothd available",
			Phase:           "repair",
			EstimatedImpact: "Forces Bluetooth pairing and transport state to re-establish.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/sudo",
			CommandArgs:     []string{"/usr/bin/pkill", "-TERM", "bluetoothd"},
			TimeoutSeconds:  10,
			RequiresAdmin:   true,
			Verification:    []string{"Wait for devices to reconnect", "Re-check Bluetooth device health in status"},
			Steps: []string{
				"Bluetooth restarts automatically and devices can reconnect",
				"Use for persistent pairing or connection issues",
			},
		},
		{
			ID:              "macos.optimize.helpd",
			Title:           "Reset Help Viewer cache",
			Description:     "Removes stale help viewer cache data so in-app help content can rebuild cleanly.",
			Risk:            domain.RiskSafe,
			Phase:           "cleanup",
			EstimatedImpact: "Refreshes stale help search and article caches.",
			Action:          domain.ActionTrash,
			Paths: []string{
				filepath.Join(d.home, "Library", "Caches", "com.apple.helpd"),
			},
			Verification: []string{"Reopen an app help article and confirm search results refresh"},
			Steps: []string{
				"Help viewer cache rebuilds automatically",
				"Use if app help search or articles look stale",
			},
		},
		{
			ID:                "macos.review.login-items",
			Title:             "Review login items",
			Description:       "Audit apps that launch at login before trimming support files.",
			Risk:              domain.RiskReview,
			Phase:             "preflight",
			EstimatedImpact:   "Highlights startup apps that should be trimmed before cleanup.",
			SuggestedByChecks: []string{"check.login_items"},
			Verification:      []string{"Confirm unused login items are disabled in System Settings"},
			Steps: []string{
				"System Settings > General > Login Items",
				"Disable unused background items first",
			},
		},
		{
			ID:                "macos.review.homebrew",
			Title:             "Review Homebrew cache growth",
			Description:       "Homebrew downloads can be large; SIFT scans them but does not auto-run brew cleanup.",
			Risk:              domain.RiskSafe,
			Phase:             "preflight",
			EstimatedImpact:   "Surfaces package cache growth before package-level cleanup.",
			RequiresApp:       []string{"Homebrew"},
			SuggestedByChecks: []string{"check.brew_updates", "check.disk_pressure"},
			Verification:      []string{"Run `brew cleanup` manually if package cache remains oversized"},
			Steps: []string{
				"Inspect ~/Library/Caches/Homebrew",
				"Run brew cleanup manually if you want package-level cleanup",
			},
		},
	}
	return append(tasks, d.dynamicMaintenanceTasks()...)
}

func (d darwinAdapter) dynamicMaintenanceTasks() []domain.MaintenanceTask {
	tasks := []domain.MaintenanceTask{
		{
			ID:              "macos.optimize.preferences-temp",
			Title:           "Clear stale preference temp files",
			Description:     "Removes abandoned preference lockfiles and temp files that can block clean preference writes.",
			Risk:            domain.RiskSafe,
			Phase:           "cleanup",
			EstimatedImpact: "Clears transient preference debris left by crashed writes.",
			Action:          domain.ActionTrash,
			PathGlobs: []string{
				filepath.Join(d.home, "Library", "Preferences", "*.lockfile"),
				filepath.Join(d.home, "Library", "Preferences", "*.tmp"),
				filepath.Join(d.home, "Library", "Preferences", "ByHost", "*.lockfile"),
				filepath.Join(d.home, "Library", "Preferences", "ByHost", "*.tmp"),
			},
			Verification: []string{"Retry the settings change that previously failed"},
			Steps: []string{
				"macOS recreates transient preference temp files automatically",
				"Use when preference writes or app settings feel stuck after crashes",
			},
		},
	}
	if _, err := os.Stat("/sbin/route"); err == nil {
		tasks = append(tasks, domain.MaintenanceTask{
			ID:              "macos.optimize.route-cache",
			Title:           "Refresh network routing cache",
			Description:     "Flushes stale route entries so macOS rebuilds active network paths cleanly.",
			Risk:            domain.RiskReview,
			Capability:      "route available",
			Phase:           "repair",
			EstimatedImpact: "Rebuilds kernel routing choices for active interfaces.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/sudo",
			CommandArgs:     []string{"/sbin/route", "-n", "flush"},
			TimeoutSeconds:  15,
			RequiresAdmin:   true,
			Verification:    []string{"Retry the failing network path or VPN route", "If stale neighbors remain, clear ARP cache next"},
			Steps: []string{
				"Active network routes are rebuilt immediately",
				"Use when path selection or gateway state remains stale after DNS fixes",
			},
		})
	}
	if _, err := os.Stat("/usr/sbin/arp"); err == nil {
		tasks = append(tasks, domain.MaintenanceTask{
			ID:              "macos.optimize.arp-cache",
			Title:           "Clear ARP neighbor cache",
			Description:     "Clears stale ARP entries so local network neighbors can repopulate cleanly.",
			Risk:            domain.RiskReview,
			Capability:      "arp available",
			Phase:           "repair",
			EstimatedImpact: "Refreshes local network neighbor discovery state.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/sudo",
			CommandArgs:     []string{"/usr/sbin/arp", "-a", "-d"},
			TimeoutSeconds:  15,
			RequiresAdmin:   true,
			Verification:    []string{"Retry connecting to the local gateway or peer host"},
			Steps: []string{
				"Local ARP neighbors repopulate automatically on the next connection",
				"Use for stale local networking or gateway discovery issues",
			},
		})
	}
	if _, err := os.Stat("/usr/bin/atsutil"); err == nil {
		tasks = append(tasks, domain.MaintenanceTask{
			ID:              "macos.optimize.font-cache",
			Title:           "Rebuild font cache",
			Description:     "Clears ATS font databases so macOS can rebuild font metadata cleanly.",
			Risk:            domain.RiskReview,
			Capability:      "atsutil available",
			Phase:           "repair",
			EstimatedImpact: "Forces font metadata to rebuild after glyph or activation drift.",
			Action:          domain.ActionCommand,
			CommandPath:     "/usr/bin/sudo",
			CommandArgs:     []string{"/usr/bin/atsutil", "databases", "-remove"},
			TimeoutSeconds:  20,
			RequiresAdmin:   true,
			Verification:    []string{"Reopen the app with broken fonts and confirm glyphs render correctly"},
			Steps: []string{
				"Font databases rebuild automatically after the next app launch",
				"Use when glyph rendering or font activation remains stale",
			},
		})
	}
	if purgePath, err := execLookPathDarwin("purge"); err == nil {
		tasks = append(tasks, domain.MaintenanceTask{
			ID:                "macos.optimize.memory-relief",
			Title:             "Release inactive memory",
			Description:       "Runs purge to drop reclaimable inactive memory pages when the system feels stuck under pressure.",
			Risk:              domain.RiskReview,
			Capability:        "purge available",
			Phase:             "repair",
			EstimatedImpact:   "Drops reclaimable inactive pages to relieve memory pressure.",
			Action:            domain.ActionCommand,
			CommandPath:       "/usr/bin/sudo",
			CommandArgs:       []string{purgePath},
			TimeoutSeconds:    20,
			RequiresAdmin:     true,
			SuggestedByChecks: []string{"check.memory_pressure", "check.swap_pressure"},
			Verification:      []string{"Check memory pressure again in `sift status`"},
			Steps: []string{
				"Inactive memory is reclaimed without killing active apps",
				"Use when memory pressure remains high after closing large workloads",
			},
		})
	}
	if nixCollectGarbage, err := execLookPathDarwin("nix-collect-garbage"); err == nil {
		tasks = append(tasks, domain.MaintenanceTask{
			ID:                "macos.optimize.nix-gc",
			Title:             "Collect old Nix generations",
			Description:       "Runs nix-collect-garbage to trim stale Nix store generations and release disk space.",
			Risk:              domain.RiskReview,
			Capability:        "nix-collect-garbage available",
			Phase:             "cleanup",
			EstimatedImpact:   "Reclaims disk from stale Nix generations while keeping current profiles.",
			Action:            domain.ActionCommand,
			CommandPath:       nixCollectGarbage,
			CommandArgs:       []string{"-d"},
			TimeoutSeconds:    120,
			SuggestedByChecks: []string{"check.disk_pressure"},
			Verification:      []string{"Re-check free disk space after garbage collection"},
			Steps: []string{
				"Old Nix generations are removed while current profiles stay intact",
				"Use when /nix/store growth is the main disk pressure source",
			},
		})
	}
	if brewPath, err := execLookPathDarwin("brew"); err == nil {
		tasks = append(tasks, domain.MaintenanceTask{
			ID:                "macos.optimize.brew-cleanup",
			Title:             "Run Homebrew cleanup",
			Description:       "Prunes stale Homebrew downloads, old package versions, and cached artifacts that remain after upgrades.",
			Risk:              domain.RiskReview,
			Capability:        "brew available",
			Phase:             "cleanup",
			EstimatedImpact:   "Reclaims package-manager cache and cellar disk usage without removing active formulas.",
			Action:            domain.ActionCommand,
			CommandPath:       brewPath,
			CommandArgs:       []string{"cleanup"},
			TimeoutSeconds:    180,
			SuggestedByChecks: []string{"check.brew_updates", "check.brew_health", "check.disk_pressure"},
			Verification:      []string{"Re-check Homebrew cache size or run `brew doctor` if warnings remain"},
			Steps: []string{
				"Old Homebrew downloads and outdated package versions are pruned",
				"Use when Homebrew caches or cellar growth are driving disk pressure",
			},
		})
	}
	if sqlitePath, err := execLookPathDarwin("sqlite3"); err == nil {
		tasks = append(tasks, d.sqliteVacuumTasks(sqlitePath)...)
	}
	return tasks
}

const optimizeSQLiteMaxBytes int64 = 100 * 1024 * 1024

var darwinProcessRunningMu sync.RWMutex
var darwinProcessRunning = darwinAnyProcessRunning

func (d darwinAdapter) sqliteVacuumTasks(sqlitePath string) []domain.MaintenanceTask {
	darwinProcessRunningMu.RLock()
	processRunning := darwinProcessRunning
	darwinProcessRunningMu.RUnlock()
	if processRunning("Mail", "Safari", "Messages") {
		return nil
	}
	candidates := []struct {
		id          string
		title       string
		description string
		patterns    []string
		steps       []string
	}{
		{
			id:          "macos.optimize.sqlite-mail-envelope",
			title:       "Vacuum Mail envelope index",
			description: "Compacts Mail envelope index databases to reclaim free pages after heavy message churn.",
			patterns:    []string{filepath.Join(d.home, "Library", "Mail", "V*", "MailData", "Envelope Index*")},
			steps: []string{
				"Mail stays closed while the compacting pass runs",
				"Use when Mail search or indexing feels bloated after large archive changes",
			},
		},
		{
			id:          "macos.optimize.sqlite-messages",
			title:       "Vacuum Messages database",
			description: "Compacts the Messages chat database without deleting conversation data.",
			patterns:    []string{filepath.Join(d.home, "Library", "Messages", "chat.db")},
			steps: []string{
				"Messages stays closed while the database compacts",
				"Use when Messages storage stays bloated after large attachment cleanup",
			},
		},
		{
			id:          "macos.optimize.sqlite-safari-history",
			title:       "Vacuum Safari history database",
			description: "Compacts Safari history storage to reclaim free pages left by deleted history rows.",
			patterns:    []string{filepath.Join(d.home, "Library", "Safari", "History.db")},
			steps: []string{
				"Safari stays closed while the database compacts",
				"Use when Safari history storage keeps growing after cleanup",
			},
		},
		{
			id:          "macos.optimize.sqlite-safari-topsites",
			title:       "Vacuum Safari Top Sites database",
			description: "Compacts Safari Top Sites storage after stale snapshot churn.",
			patterns:    []string{filepath.Join(d.home, "Library", "Safari", "TopSites.db")},
			steps: []string{
				"Safari stays closed while the database compacts",
				"Use when Safari UI data remains bloated after cache cleanup",
			},
		},
	}
	var tasks []domain.MaintenanceTask
	for _, candidate := range candidates {
		index := 0
		for _, pattern := range candidate.patterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}
			slices.Sort(matches)
			for _, match := range matches {
				if !eligibleSQLiteOptimizeTarget(match) {
					continue
				}
				index++
				id := candidate.id
				if index > 1 {
					id = fmt.Sprintf("%s-%d", candidate.id, index)
				}
				tasks = append(tasks, domain.MaintenanceTask{
					ID:             id,
					Title:          candidate.title,
					Description:    candidate.description,
					Risk:           domain.RiskReview,
					Capability:     "sqlite3 available; target app closed",
					Action:         domain.ActionCommand,
					CommandPath:    sqlitePath,
					CommandArgs:    []string{match, "VACUUM;"},
					TimeoutSeconds: 20,
					Steps:          candidate.steps,
				})
			}
		}
	}
	return tasks
}

func eligibleSQLiteOptimizeTarget(path string) bool {
	if strings.HasSuffix(path, "-wal") || strings.HasSuffix(path, "-shm") {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if info.Size() <= 0 || info.Size() > optimizeSQLiteMaxBytes {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	header := make([]byte, len("SQLite format 3\x00"))
	if _, err := file.Read(header); err != nil {
		return false
	}
	return string(header) == "SQLite format 3\x00"
}

func darwinAnyProcessRunning(names ...string) bool {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := exec.Command("/usr/bin/pgrep", "-x", name).Run(); err == nil {
			return true
		}
	}
	return false
}

func (d darwinAdapter) IsProcessRunning(names ...string) bool {
	return darwinAnyProcessRunning(names...)
}

func (d darwinAdapter) IsAdminPath(path string) bool {
	for _, prefix := range []string{"/Applications", "/Library", "/System"} {
		if strings.HasPrefix(filepath.Clean(path), prefix) {
			return true
		}
	}
	return false
}

func (d darwinAdapter) bundleIdentifier(ctx context.Context, bundlePath string) string {
	if bundlePath == "" {
		return ""
	}
	infoPlist := filepath.Join(bundlePath, "Contents", "Info.plist")
	if _, err := os.Stat(infoPlist); err != nil {
		return ""
	}
	out, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", infoPlist, "CFBundleIdentifier").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	raw, readErr := os.ReadFile(infoPlist)
	if readErr != nil {
		return ""
	}
	content := string(raw)
	key := "<key>CFBundleIdentifier</key>"
	keyIndex := strings.Index(content, key)
	if keyIndex < 0 {
		return ""
	}
	content = content[keyIndex+len(key):]
	start := strings.Index(content, "<string>")
	end := strings.Index(content, "</string>")
	if start < 0 || end < 0 || end <= start+len("<string>") {
		return ""
	}
	return strings.TrimSpace(content[start+len("<string>") : end])
}

func (d darwinAdapter) findUninstallHelper(bundlePath, appName string) string {
	candidates := []string{
		filepath.Join(filepath.Dir(bundlePath), "Uninstall "+appName+".app"),
		filepath.Join(filepath.Dir(bundlePath), appName+" Uninstaller.app"),
		filepath.Join(filepath.Dir(bundlePath), "Uninstaller.app"),
		filepath.Join(bundlePath, "Contents", "Resources", "Uninstaller.app"),
		filepath.Join(bundlePath, "Contents", "Resources", "uninstall.sh"),
		filepath.Join(bundlePath, "Contents", "MacOS", "uninstall"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return domain.NormalizePath(candidate)
		}
	}
	files, err := os.ReadDir(filepath.Dir(bundlePath))
	if err != nil {
		return ""
	}
	nameKey := normalizedNameKey(appName)
	for _, file := range files {
		if !file.IsDir() || !strings.HasSuffix(file.Name(), ".app") {
			continue
		}
		base := strings.TrimSuffix(file.Name(), ".app")
		key := normalizedNameKey(base)
		if key == "" || !strings.Contains(key, "uninstall") {
			continue
		}
		if strings.Contains(key, nameKey) || strings.Contains(nameKey, key) || slices.Contains([]string{"uninstaller", "uninstall"}, key) {
			return domain.NormalizePath(filepath.Join(filepath.Dir(bundlePath), file.Name()))
		}
	}
	return ""
}
