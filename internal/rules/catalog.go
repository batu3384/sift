package rules

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

type Definition struct {
	ID          string
	Name        string
	Category    domain.Category
	Risk        domain.Risk
	Action      domain.Action
	Description string
	Roots       func(platform.Adapter, []string) []string
	Scanner     func(context.Context, platform.Adapter, []string) ([]domain.Finding, []string, error)
}

func Catalog() []Definition {
	return []Definition{
		{
			ID:          "temp_files",
			Name:        "Temporary files",
			Category:    domain.CategoryTempFiles,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "OS and app temporary files.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Temp
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Temp, domain.CategoryTempFiles, domain.RiskSafe, domain.ActionTrash, "OS temporary data")
			},
		},
		{
			ID:          "logs",
			Name:        "Application logs",
			Category:    domain.CategoryLogs,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "Log files and crash records.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Logs
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Logs, domain.CategoryLogs, domain.RiskSafe, domain.ActionTrash, "Application logs")
			},
		},
		{
			ID:          "safe_system_clutter",
			Name:        "Safe system clutter",
			Category:    domain.CategorySystemClutter,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "User-level cache folders that are cheap to rebuild.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return unique(adapter.CuratedRoots().Temp, adapter.CuratedRoots().Logs)
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, unique(adapter.CuratedRoots().Temp, adapter.CuratedRoots().Logs), domain.CategorySystemClutter, domain.RiskSafe, domain.ActionTrash, "Safe user-level clutter")
			},
		},
		{
			ID:          "recent_items",
			Name:        "Recent items",
			Category:    domain.CategorySystemClutter,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "macOS recent apps, documents, hosts, and servers history files.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().RecentItems
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().RecentItems, domain.CategorySystemClutter, domain.RiskReview, domain.ActionTrash, "Recent items history")
			},
		},
		{
			ID:          "stale_login_items",
			Name:        "Stale login items",
			Category:    domain.CategoryMaintenance,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "Potential stale launch agents that reference missing apps or helpers.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return nil
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanStaleLoginItems(ctx, adapter)
			},
		},
		{
			ID:          "orphaned_services",
			Name:        "Orphaned services",
			Category:    domain.CategoryAppLeftovers,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "System launch agents and daemons whose app or helper target no longer exists.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return nil
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanOrphanedServices(ctx, adapter)
			},
		},
		{
			ID:          "system_update_artifacts",
			Name:        "System update artifacts",
			Category:    domain.CategorySystemClutter,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "macOS update caches, stale installer payloads, and Apple Silicon update bundles.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().SystemUpdate
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanSystemUpdateArtifacts(ctx, adapter)
			},
		},
		{
			ID:          "developer_caches",
			Name:        "Developer caches",
			Category:    domain.CategoryDeveloperCaches,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "IDE, build and language caches.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Developer
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanDeveloperRoots(ctx, adapter)
			},
		},
		{
			ID:          "package_manager_caches",
			Name:        "Package manager caches",
			Category:    domain.CategoryPackageCaches,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Cached downloads from package managers.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().PackageManager
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().PackageManager, domain.CategoryPackageCaches, domain.RiskReview, domain.ActionTrash, "Package manager cache")
			},
		},
		{
			ID:          "browser_data",
			Name:        "Browser caches",
			Category:    domain.CategoryBrowserData,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Browser cache folders, stale profile caches, and old framework versions.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Browser
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				findings, warnings, err := scanBrowserRoots(ctx, adapter, adapter.CuratedRoots().Browser)
				if err != nil {
					return findings, warnings, err
				}
				chromeFindings, chromeWarnings, chromeErr := scanChromeOldVersions(ctx, adapter)
				if chromeErr != nil {
					warnings = append(warnings, chromeErr.Error())
				}
				findings = append(findings, chromeFindings...)
				warnings = append(warnings, chromeWarnings...)
				edgeFindings, edgeWarnings, edgeErr := scanEdgeOldVersions(ctx, adapter)
				if edgeErr != nil {
					warnings = append(warnings, edgeErr.Error())
				}
				findings = append(findings, edgeFindings...)
				warnings = append(warnings, edgeWarnings...)
				return findings, dedupeStrings(warnings), nil
			},
		},
		{
			ID:          "installer_leftovers",
			Name:        "Installer leftovers",
			Category:    domain.CategoryInstallerLeft,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Old installer archives in Downloads.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Installer
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanInstallerFiles(ctx, adapter)
			},
		},
		{
			ID:          "app_leftovers",
			Name:        "App leftovers",
			Category:    domain.CategoryAppLeftovers,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Stale support directories from removed or moved apps.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().AppSupport
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanAppSupportRoots(ctx, adapter)
			},
		},
		{
			ID:          "cloud_office",
			Name:        "Cloud & Office",
			Category:    domain.CategoryCloudOffice,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Cloud storage and office app caches (Dropbox, OneDrive, iCloud, Teams, Slack).",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().CloudOffice
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().CloudOffice, domain.CategoryCloudOffice, domain.RiskReview, domain.ActionTrash, "Cloud & Office cache")
			},
		},
		{
			ID:          "virtualization",
			Name:        "Virtualization",
			Category:    domain.CategoryVirtualization,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Virtualization app caches (Docker, VMware, Parallels, VirtualBox).",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Virtualization
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Virtualization, domain.CategoryVirtualization, domain.RiskReview, domain.ActionTrash, "Virtualization cache")
			},
		},
		{
			ID:          "device_backups",
			Name:        "Device Backups",
			Category:    domain.CategoryDeviceBackups,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "iOS device backups and sync data.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().DeviceBackups
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().DeviceBackups, domain.CategoryDeviceBackups, domain.RiskReview, domain.ActionAdvisory, "Device backup data")
			},
		},
		{
			ID:          "time_machine",
			Name:        "Time Machine",
			Category:    domain.CategoryTimeMachine,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "Time Machine local snapshots and backup cache.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().TimeMachine
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().TimeMachine, domain.CategoryTimeMachine, domain.RiskReview, domain.ActionAdvisory, "Time Machine data")
			},
		},
		{
			ID:          "maven_cache",
			Name:        "Maven Cache",
			Category:    domain.CategoryMavenCache,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Apache Maven local repository cache (~/.m2/repository).",
			Roots: func(_ platform.Adapter, _ []string) []string {
				home, _ := os.UserHomeDir()
				return []string{filepath.Join(home, ".m2", "repository")}
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				home, _ := os.UserHomeDir()
				return scanRootEntries(ctx, adapter, []string{filepath.Join(home, ".m2", "repository")}, domain.CategoryMavenCache, domain.RiskReview, domain.ActionTrash, "Maven cache")
			},
		},
		{
			ID:          "ipfs_node",
			Name:        "IPFS Node",
			Category:    domain.CategoryIPFS,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "IPFS node data and cache.",
			Roots: func(_ platform.Adapter, _ []string) []string {
				home, _ := os.UserHomeDir()
				return []string{filepath.Join(home, ".ipfs"), filepath.Join(home, ".local", "share", "ipfs")}
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				home, _ := os.UserHomeDir()
				roots := []string{filepath.Join(home, ".ipfs"), filepath.Join(home, ".local", "share", "ipfs")}
				return scanRootEntries(ctx, adapter, roots, domain.CategoryIPFS, domain.RiskReview, domain.ActionTrash, "IPFS node data")
			},
		},
		{
			ID:          "system_caches",
			Name:        "System Caches",
			Category:    domain.CategorySystemCaches,
			Risk:        domain.RiskHigh,
			Action:      domain.ActionPermanent,
			Description: "System-level caches in /Library/Caches (requires sudo).",
			Roots: func(_ platform.Adapter, _ []string) []string {
				return []string{"/Library/Caches"}
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, []string{"/Library/Caches"}, domain.CategorySystemCaches, domain.RiskHigh, domain.ActionPermanent, "System caches")
			},
		},
		{
			ID:          "trash",
			Name:        "Trash",
			Category:    domain.CategoryTrash,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "User trash bin contents.",
			Roots: func(_ platform.Adapter, _ []string) []string {
				home, _ := os.UserHomeDir()
				return []string{filepath.Join(home, ".Trash")}
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				home, _ := os.UserHomeDir()
				return scanRootEntries(ctx, adapter, []string{filepath.Join(home, ".Trash")}, domain.CategoryTrash, domain.RiskReview, domain.ActionAdvisory, "Trash contents")
			},
		},
		{
			ID:          "finder_metadata",
			Name:        "Finder metadata",
			Category:    domain.CategorySystemClutter,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionPermanent,
			Description: ".DS_Store files scattered across the home directory.",
			Roots: func(_ platform.Adapter, _ []string) []string {
				return nil
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanDSStoreFiles(ctx, adapter)
			},
		},
		{
			ID:          "ios_device_backups",
			Name:        "iOS device backups",
			Category:    domain.CategorySystemClutter,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "Large iOS device backups in MobileSync that may be stale.",
			Roots: func(_ platform.Adapter, _ []string) []string {
				return nil
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanIOSDeviceBackups(ctx, adapter)
			},
		},
		{
			ID:          "time_machine_cleanup",
			Name:        "Time Machine incomplete backups",
			Category:    domain.CategorySystemClutter,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Incomplete Time Machine backup snapshots (*.inProgress) older than 24 hours.",
			Roots: func(_ platform.Adapter, _ []string) []string {
				return nil
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanTimeMachineFailedBackups(ctx, adapter)
			},
		},
		// Font Cache
		{
			ID:          "font_cache",
			Name:        "Font Cache",
			Category:    domain.CategoryFontCache,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "System and user font cache files.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().FontCache
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().FontCache, domain.CategoryFontCache, domain.RiskSafe, domain.ActionTrash, "Font cache")
			},
		},
		// Print Spooler
		{
			ID:          "print_spooler",
			Name:        "Print Spooler",
			Category:    domain.CategoryPrintSpooler,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "Print spooler and job files.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().PrintSpooler
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().PrintSpooler, domain.CategoryPrintSpooler, domain.RiskSafe, domain.ActionTrash, "Print spooler")
			},
		},
		// Xcode
		{
			ID:          "xcode",
			Name:        "Xcode",
			Category:    domain.CategoryXcode,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Xcode derived data, archives, and simulator caches.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Xcode
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Xcode, domain.CategoryXcode, domain.RiskReview, domain.ActionTrash, "Xcode cache")
			},
		},
		// Unity
		{
			ID:          "unity",
			Name:        "Unity",
			Category:    domain.CategoryUnity,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Unity Editor cache and build artifacts.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Unity
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Unity, domain.CategoryUnity, domain.RiskReview, domain.ActionTrash, "Unity cache")
			},
		},
		// Unreal Engine
		{
			ID:          "unreal",
			Name:        "Unreal Engine",
			Category:    domain.CategoryUnreal,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Unreal Engine cache and intermediate files.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Unreal
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Unreal, domain.CategoryUnreal, domain.RiskReview, domain.ActionTrash, "Unreal cache")
			},
		},
		// Android
		{
			ID:          "android",
			Name:        "Android",
			Category:    domain.CategoryAndroid,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Android SDK and Gradle build cache.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Android
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Android, domain.CategoryAndroid, domain.RiskReview, domain.ActionTrash, "Android cache")
			},
		},
		// Rust
		{
			ID:          "rust",
			Name:        "Rust",
			Category:    domain.CategoryRust,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Rust cargo registry and build cache.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Rust
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Rust, domain.CategoryRust, domain.RiskReview, domain.ActionTrash, "Rust cache")
			},
		},
		// Node.js
		{
			ID:          "node_modules",
			Name:        "Node Modules",
			Category:    domain.CategoryNode,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Node.js modules and npm cache.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().NodeModules
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().NodeModules, domain.CategoryNode, domain.RiskReview, domain.ActionTrash, "Node modules")
			},
		},
		// Python
		{
			ID:          "python_cache",
			Name:        "Python Cache",
			Category:    domain.CategoryPython,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "Python pip cache and __pycache__.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().PythonCache
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().PythonCache, domain.CategoryPython, domain.RiskSafe, domain.ActionTrash, "Python cache")
			},
		},
		// Go
		{
			ID:          "go_cache",
			Name:        "Go Cache",
			Category:    domain.CategoryGo,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "Go build cache and module cache.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().GoCache
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().GoCache, domain.CategoryGo, domain.RiskSafe, domain.ActionTrash, "Go cache")
			},
		},
		// Fonts
		{
			ID:          "fonts",
			Name:        "Fonts",
			Category:    domain.CategoryFonts,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "User-installed fonts (review before deleting).",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Fonts
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Fonts, domain.CategoryFonts, domain.RiskReview, domain.ActionAdvisory, "Fonts")
			},
		},
		// Diagnostics
		{
			ID:          "diagnostics",
			Name:        "Diagnostics",
			Category:    domain.CategoryDiagnostics,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "System diagnostics and crash reports.",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Diagnostics
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Diagnostics, domain.CategoryDiagnostics, domain.RiskSafe, domain.ActionTrash, "Diagnostics")
			},
		},
		// Media
		{
			ID:          "media_cache",
			Name:        "Media Cache",
			Category:    domain.CategoryMedia,
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			Description: "Media player caches (Plex, VLC, Spotify).",
			Roots: func(adapter platform.Adapter, _ []string) []string {
				return adapter.CuratedRoots().Media
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, _ []string) ([]domain.Finding, []string, error) {
				return scanRootEntries(ctx, adapter, adapter.CuratedRoots().Media, domain.CategoryMedia, domain.RiskSafe, domain.ActionTrash, "Media cache")
			},
		},
	}
}

func ByIDs(ids []string) []Definition {
	catalog := Catalog()
	if len(ids) == 0 {
		return catalog
	}
	var selected []Definition
	for _, definition := range catalog {
		if slices.Contains(ids, definition.ID) {
			selected = append(selected, definition)
		}
	}
	return selected
}

func TargetDefinitions(targets []string) []Definition {
	if len(targets) == 0 {
		return nil
	}
	return []Definition{
		{
			ID:          "target.path",
			Name:        "Targeted path analysis",
			Category:    domain.CategorySystemClutter,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "User-provided paths.",
			Roots: func(adapter platform.Adapter, raw []string) []string {
				return adapter.ResolveTargets(raw)
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, raw []string) ([]domain.Finding, []string, error) {
				return scanTargets(ctx, adapter.ResolveTargets(raw), adapter)
			},
		},
	}
}

func PurgeTargetDefinitions(targets []string) []Definition {
	if len(targets) == 0 {
		return nil
	}
	return []Definition{
		{
			ID:          "purge.project_artifact",
			Name:        "Project artifact purge",
			Category:    domain.CategoryProjectArtifacts,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Known build and dependency artifacts inside project directories.",
			Roots: func(adapter platform.Adapter, raw []string) []string {
				return adapter.ResolveTargets(raw)
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, raw []string) ([]domain.Finding, []string, error) {
				return scanPurgeTargets(ctx, adapter.ResolveTargets(raw), adapter)
			},
		},
	}
}

func PurgeDiscoveryDefinitions(targets []string) []Definition {
	if len(targets) == 0 {
		return nil
	}
	return []Definition{
		{
			ID:          "purge.project_artifact_scan",
			Name:        "Project artifact discovery",
			Category:    domain.CategoryProjectArtifacts,
			Risk:        domain.RiskReview,
			Action:      domain.ActionTrash,
			Description: "Scan configured roots for known project artifacts without deleting them.",
			Roots: func(adapter platform.Adapter, raw []string) []string {
				return adapter.ResolveTargets(raw)
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, raw []string) ([]domain.Finding, []string, error) {
				return scanPurgeDiscovery(ctx, adapter.ResolveTargets(raw), adapter)
			},
		},
	}
}

func AnalysisDefinitions(targets []string) []Definition {
	return []Definition{
		{
			ID:          "analyze.disk_usage",
			Name:        "Disk usage overview",
			Category:    domain.CategoryDiskUsage,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "Immediate children of the requested paths sorted by size.",
			Roots: func(adapter platform.Adapter, raw []string) []string {
				return adapter.ResolveTargets(raw)
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, raw []string) ([]domain.Finding, []string, error) {
				return scanDiskUsage(ctx, adapter.ResolveTargets(raw))
			},
		},
		{
			ID:          "analyze.large_files",
			Name:        "Large files",
			Category:    domain.CategoryLargeFiles,
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "Largest regular files under the requested targets.",
			Roots: func(adapter platform.Adapter, raw []string) []string {
				return adapter.ResolveTargets(raw)
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, raw []string) ([]domain.Finding, []string, error) {
				return scanLargeFiles(ctx, adapter.ResolveTargets(raw))
			},
		},
		{
			ID:          "analyze.duplicates",
			Name:        "Duplicate files",
			Category:    "duplicates",
			Risk:        domain.RiskReview,
			Action:      domain.ActionAdvisory,
			Description: "Files with identical content (by hash).",
			Roots: func(adapter platform.Adapter, raw []string) []string {
				return adapter.ResolveTargets(raw)
			},
			Scanner: func(ctx context.Context, adapter platform.Adapter, raw []string) ([]domain.Finding, []string, error) {
				return scanDuplicates(ctx, adapter.ResolveTargets(raw))
			},
		},
	}
}

var purgeArtifactNames = map[string]struct{}{
	"node_modules":  {},
	"dist":          {},
	"build":         {},
	"bin":           {},
	"obj":           {},
	"target":        {},
	".next":         {},
	".nuxt":         {},
	".output":       {},
	"coverage":      {},
	".pytest_cache": {},
	".mypy_cache":   {},
	".tox":          {},
	".nox":          {},
	"__pycache__":   {},
	"venv":          {},
	".venv":         {},
	".gradle":       {},
	".turbo":        {},
	".angular":      {},
	".astro":        {},
	".cxx":          {},
	".dart_tool":    {},
	".expo":         {},
	".parcel-cache": {},
	".svelte-kit":   {},
	".zig-cache":    {},
	"zig-out":       {},
	"DerivedData":   {},
}

var projectMarkers = []string{
	".git",
	"Makefile",
	"package.json",
	"pnpm-workspace.yaml",
	"turbo.json",
	"nx.json",
	"rush.json",
	"lerna.json",
	"go.mod",
	"Cargo.toml",
	"Cargo.lock",
	"pubspec.yaml",
	"pyproject.toml",
	"requirements.txt",
	"Pipfile",
	"composer.json",
	"Gemfile",
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	"Directory.Build.props",
	"Directory.Build.targets",
	"*.sln",
	"*.xcodeproj",
	"*.xcworkspace",
	"build.zig",
	"build.zig.zon",
	"WORKSPACE",
	"WORKSPACE.bazel",
	"MODULE.bazel",
}

var protectedPurgeContainers = map[string]struct{}{
	"vendor":      {},
	"Pods":        {},
	".terraform":  {},
	"Carthage":    {},
	"third_party": {},
	"ThirdParty":  {},
}

const maxFindingsPerRoot = 25
const maxAnalyzeDiskUsage = 20
const maxAnalyzeLargeFiles = 15
const analyzeLargeFileMinBytes = 1 << 20
const analyzeFoldedDirMaxDepth = 8
const maxPurgeDiscoveryFindings = 100

var spotlightLargeFileSearch = discoverSpotlightLargeFiles
var analyzeDiskUsageLoader = scanDiskUsageFresh
var analyzeLargeFilesLoader = scanLargeFilesFresh
var analyzeHooksMu sync.RWMutex

var leafCleanupBasenames = map[string]struct{}{
	"_cacache":           {},
	"_logs":              {},
	"_npx":               {},
	"appcache":           {},
	"cache":              {},
	"cache2":             {},
	"cacheddata":         {},
	"cachedextensions":   {},
	"cachestorage":       {},
	"code cache":         {},
	"crashdumps":         {},
	"dawngraphitecache":  {},
	"dawnwebgpucache":    {},
	"depotcache":         {},
	"deriveddata":        {},
	"documentationcache": {},
	"documentationindex": {},
	"gpucache":           {},
	"grshadercache":      {},
	"htmlcache":          {},
	"logs":               {},
	"media cache files":  {},
	"reportarchive":      {},
	"reportqueue":        {},
	"shader-cache":       {},
	"shadercache":        {},
	"startupcache":       {},
	"thumbnails":         {},
	"videocache":         {},
}

var analyzeFoldIgnoredFiles = map[string]struct{}{
	".DS_Store":   {},
	".localized":  {},
	"Thumbs.db":   {},
	"desktop.ini": {},
}

var cleanupSourceMatchers = []struct {
	needle string
	label  string
}{
	{"library/caches/homebrew/downloads", "Homebrew downloads"},
	{"library/caches/homebrew", "Homebrew cache"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recentapplications.sfl2", "Recent apps"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recentdocuments.sfl2", "Recent documents"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recentservers.sfl2", "Recent servers"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recenthosts.sfl2", "Recent hosts"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recentapplications.sfl", "Recent apps"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recentdocuments.sfl", "Recent documents"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recentservers.sfl", "Recent servers"},
	{"application support/com.apple.sharedfilelist/com.apple.lssharedfilelist.recenthosts.sfl", "Recent hosts"},
	{"preferences/com.apple.recentitems.plist", "Recent items preferences"},
	{"application support/microsoft/edgeupdater/apps/msedge-stable", "Edge updater old version"},
	{"library/updates", "macOS updates cache"},
	{"/library/updates", "System library updates"},
	{"rosetta_update_bundle", "Rosetta 2 update bundle"},
	{"com.apple.rosetta.update", "Rosetta 2 user cache"},
	{"com.apple.amp.mediasevicesd", "Apple Silicon media cache"},
	{"application support/com.apple.idleassetsd/customer", "System idle assets"},
	{"application support/com.apple.idleassetsd", "Idle assets cache"},
	{"application support/com.apple.wallpaper/aerials/videos", "Aerial wallpaper videos"},
	{"containers/com.apple.wallpaper.agent/data/library/caches", "Wallpaper agent cache"},
	{"library/messages/stickercache", "Messages sticker cache"},
	{"library/messages/caches/previews/attachments", "Messages preview attachment cache"},
	{"library/messages/caches/previews/stickercache", "Messages preview sticker cache"},
	{"library/caches/com.apple.quicklook.thumbnailcache", "Quick Look thumbnail cache"},
	{"library/caches/quick look", "Quick Look cache"},
	{"library/autosave information", "Autosave information"},
	{"library/identitycaches", "Identity caches"},
	{"library/suggestions", "Suggestions cache"},
	{"library/calendars/calendar cache", "Calendar cache"},
	{"application support/addressbook/sources/", "Address Book photo cache"},
	{".npm/_cacache", "npm cache"},
	{".npm/_logs", "npm logs"},
	{".npm/_npx", "npm npx cache"},
	{"pnpm-store", "pnpm store"},
	{"bun/install/cache", "Bun cache"},
	{"yarn/cache", "Yarn cache"},
	{"berry/cache", "Yarn Berry cache"},
	{"library/caches/yarn", "Yarn cache"},
	{".cache/pip", "pip cache"},
	{"library/caches/pip", "pip cache"},
	{".cache/uv", "uv cache"},
	{".android/cache", "Android cache"},
	{".android/build-cache", "Android build cache"},
	{".pub-cache", "Dart pub cache"},
	{"flutter/bin/cache", "Flutter SDK cache"},
	{".expo-shared", "Expo shared cache"},
	{".expo", "Expo cache"},
	{".cache/deno", "Deno cache"},
	{"library/caches/deno", "Deno cache"},
	{".cache/bazel", "Bazel cache"},
	{"library/caches/bazel", "Bazel cache"},
	{".grafana/cache", "Grafana cache"},
	{".prometheus/data/wal", "Prometheus WAL"},
	{".jenkins/workspace/", "Jenkins workspace artifacts"},
	{".cache/poetry", "Poetry cache"},
	{".cache/ruff", "Ruff cache"},
	{".cache/mypy", "MyPy cache"},
	{".pytest_cache", "Pytest cache"},
	{".ruff_cache", "Ruff cache"},
	{".mypy_cache", "MyPy cache"},
	{".cache/huggingface", "Hugging Face cache"},
	{".cache/torch", "PyTorch cache"},
	{".cache/tensorflow", "TensorFlow cache"},
	{".cache/wandb", "Weights & Biases cache"},
	{".cache/go-build", "Go build cache"},
	{".cargo/registry/cache", "Cargo registry cache"},
	{".cargo/registry", "Cargo registry"},
	{".cargo/git/db", "Cargo git cache"},
	{".cargo/git", "Cargo git cache"},
	{".gradle/caches", "Gradle cache"},
	{".m2/repository", "Maven repository cache"},
	{".ivy2/cache", "Ivy cache"},
	{".composer/cache", "Composer cache"},
	{"library/developer/xcode/deriveddata", "Xcode derived data"},
	{"library/developer/xcode/archives", "Xcode archives"},
	{"library/developer/xcode/documentationcache", "Xcode documentation cache"},
	{"library/developer/xcode/documentationindex", "Xcode documentation index"},
	{"library/developer/xcode/products", "Xcode build products"},
	{"library/developer/coresimulator/caches", "Simulator cache"},
	{"library/developer/coresimulator/devices", "CoreSimulator device cache"},
	{"library/logs/coresimulator", "CoreSimulator logs"},
	{"library/caches/jetbrains", "JetBrains IDE caches"},
	{"application support/jetbrains/toolbox/cache", "JetBrains Toolbox cache"},
	{"application support/jetbrains/toolbox/logs", "JetBrains Toolbox logs"},
	{"jetbrains/toolbox/cache", "JetBrains Toolbox cache"},
	{"jetbrains/toolbox/logs", "JetBrains Toolbox logs"},
	{"ios device logs", "iOS device logs"},
	{"watchos device logs", "watchOS device logs"},
	{"application support/code/cache", "VS Code cache"},
	{"application support/code/cachedextensions", "VS Code extension cache"},
	{"application support/code/cacheddata", "VS Code data cache"},
	{"application support/code/dawngraphitecache", "VS Code Dawn cache"},
	{"application support/code/dawnwebgpucache", "VS Code WebGPU cache"},
	{"application support/code/gpucache", "VS Code GPU cache"},
	{"application support/code/logs", "VS Code logs"},
	{"library/caches/com.microsoft.vscode", "VS Code cache"},
	{"application support/figma/cache", "Figma cache"},
	{"application support/figma/code cache", "Figma code cache"},
	{"application support/figma/cacheddata", "Figma data cache"},
	{"application support/figma/gpucache", "Figma GPU cache"},
	{"application support/figma/service worker/cachestorage", "Figma service worker cache"},
	{"application support/figma/logs", "Figma logs"},
	{"application support/postman/cache", "Postman cache"},
	{"application support/postman/code cache", "Postman code cache"},
	{"application support/postman/cacheddata", "Postman data cache"},
	{"application support/postman/gpucache", "Postman GPU cache"},
	{"application support/postman/service worker/cachestorage", "Postman service worker cache"},
	{"application support/postman/logs", "Postman logs"},
	{"application support/zed/cache", "Zed cache"},
	{"application support/zed/code cache", "Zed code cache"},
	{"application support/zed/cacheddata", "Zed data cache"},
	{"application support/zed/gpucache", "Zed GPU cache"},
	{"application support/zed/service worker/cachestorage", "Zed service worker cache"},
	{"application support/zed/logs", "Zed logs"},
	{"application support/claude/cache", "Claude cache"},
	{"application support/claude/code cache", "Claude code cache"},
	{"application support/claude/cacheddata", "Claude data cache"},
	{"application support/claude/dawngraphitecache", "Claude Dawn cache"},
	{"application support/claude/dawnwebgpucache", "Claude WebGPU cache"},
	{"application support/claude/gpucache", "Claude GPU cache"},
	{"application support/claude/service worker/cachestorage", "Claude service worker cache"},
	{"application support/claude/logs", "Claude logs"},
	{"application support/chatgpt/cache", "ChatGPT cache"},
	{"application support/chatgpt/code cache", "ChatGPT code cache"},
	{"application support/chatgpt/cacheddata", "ChatGPT data cache"},
	{"application support/chatgpt/gpucache", "ChatGPT GPU cache"},
	{"application support/chatgpt/service worker/cachestorage", "ChatGPT service worker cache"},
	{"application support/chatgpt/logs", "ChatGPT logs"},
	{"library/logs/claude", "Claude logs"},
	{"library/logs/chatgpt", "ChatGPT logs"},
	{"library/logs/figma", "Figma logs"},
	{"library/logs/postman", "Postman logs"},
	{"library/logs/zed", "Zed logs"},
	{"library/caches/com.openai.chat", "ChatGPT cache"},
	{"library/caches/com.anthropic.claudefordesktop", "Claude desktop cache"},
	{"library/caches/com.konghq.insomnia", "Insomnia cache"},
	{"library/caches/com.tinyapp.tableplus", "TablePlus cache"},
	{"library/caches/com.getpaw.paw", "Paw API cache"},
	{"library/caches/com.charlesproxy.charles", "Charles Proxy cache"},
	{"library/caches/com.proxyman.nsproxy", "Proxyman cache"},
	{"library/caches/com.github.githubdesktop", "GitHub Desktop cache"},
	{"library/caches/com.bohemiancoding.sketch3", "Sketch cache"},
	{"application support/com.bohemiancoding.sketch3/cache", "Sketch app cache"},
	{"library/caches/adobe", "Adobe cache"},
	{"application support/adobe/common/media cache files", "Adobe media cache"},
	{"library/caches/com.adobe.premierepro", "Premiere Pro cache"},
	{"library/caches/com.apple.finalcut", "Final Cut Pro cache"},
	{"library/caches/com.blackmagic-design.davinciresolve", "DaVinci Resolve cache"},
	{"library/caches/net.telestream.screenflow10", "ScreenFlow cache"},
	{"library/caches/org.blenderfoundation.blender", "Blender cache"},
	{"library/caches/sentrycrash", "Sentry crash reports"},
	{"library/caches/kscrash", "KSCrash reports"},
	{"library/caches/com.crashlytics.data", "Crashlytics data"},
	{"roaming/figma/cache", "Figma cache"},
	{"roaming/figma/code cache", "Figma code cache"},
	{"roaming/figma/cacheddata", "Figma data cache"},
	{"roaming/figma/gpucache", "Figma GPU cache"},
	{"roaming/figma/service worker/cachestorage", "Figma service worker cache"},
	{"roaming/figma/logs", "Figma logs"},
	{"roaming/postman/cache", "Postman cache"},
	{"roaming/postman/code cache", "Postman code cache"},
	{"roaming/postman/cacheddata", "Postman data cache"},
	{"roaming/postman/gpucache", "Postman GPU cache"},
	{"roaming/postman/service worker/cachestorage", "Postman service worker cache"},
	{"roaming/postman/logs", "Postman logs"},
	{"roaming/zed/cache", "Zed cache"},
	{"roaming/zed/code cache", "Zed code cache"},
	{"roaming/zed/cacheddata", "Zed data cache"},
	{"roaming/zed/gpucache", "Zed GPU cache"},
	{"roaming/zed/service worker/cachestorage", "Zed service worker cache"},
	{"roaming/zed/logs", "Zed logs"},
	{"roaming/claude/cache", "Claude cache"},
	{"roaming/claude/code cache", "Claude code cache"},
	{"roaming/claude/cacheddata", "Claude data cache"},
	{"roaming/claude/gpucache", "Claude GPU cache"},
	{"roaming/claude/service worker/cachestorage", "Claude service worker cache"},
	{"roaming/claude/logs", "Claude logs"},
	{"roaming/chatgpt/cache", "ChatGPT cache"},
	{"roaming/chatgpt/code cache", "ChatGPT code cache"},
	{"roaming/chatgpt/cacheddata", "ChatGPT data cache"},
	{"roaming/chatgpt/gpucache", "ChatGPT GPU cache"},
	{"roaming/chatgpt/service worker/cachestorage", "ChatGPT service worker cache"},
	{"roaming/chatgpt/logs", "ChatGPT logs"},
	{"application support/cursor/cache", "Cursor cache"},
	{"application support/cursor/code cache", "Cursor code cache"},
	{"application support/cursor/cachedextensions", "Cursor extension cache"},
	{"application support/cursor/cacheddata", "Cursor data cache"},
	{"application support/cursor/gpucache", "Cursor GPU cache"},
	{"application support/cursor/service worker/cachestorage", "Cursor service worker cache"},
	{"application support/cursor/logs", "Cursor logs"},
	{"roaming/cursor/cache", "Cursor cache"},
	{"roaming/cursor/code cache", "Cursor code cache"},
	{"roaming/cursor/cachedextensions", "Cursor extension cache"},
	{"roaming/cursor/cacheddata", "Cursor data cache"},
	{"roaming/cursor/gpucache", "Cursor GPU cache"},
	{"roaming/cursor/service worker/cachestorage", "Cursor service worker cache"},
	{"roaming/cursor/logs", "Cursor logs"},
	{"application support/vscodium/cache", "VSCodium cache"},
	{"application support/vscodium/code cache", "VSCodium code cache"},
	{"application support/vscodium/cachedextensions", "VSCodium extension cache"},
	{"application support/vscodium/cacheddata", "VSCodium data cache"},
	{"application support/vscodium/gpucache", "VSCodium GPU cache"},
	{"application support/vscodium/service worker/cachestorage", "VSCodium service worker cache"},
	{"application support/vscodium/logs", "VSCodium logs"},
	{"roaming/vscodium/cache", "VSCodium cache"},
	{"roaming/vscodium/code cache", "VSCodium code cache"},
	{"roaming/vscodium/cachedextensions", "VSCodium extension cache"},
	{"roaming/vscodium/cacheddata", "VSCodium data cache"},
	{"roaming/vscodium/gpucache", "VSCodium GPU cache"},
	{"roaming/vscodium/service worker/cachestorage", "VSCodium service worker cache"},
	{"roaming/vscodium/logs", "VSCodium logs"},
	{"application support/microsoft/teams/logs", "Teams logs"},
	{"application support/microsoft/teams/cache", "Teams cache"},
	{"application support/microsoft/teams/code cache", "Teams code cache"},
	{"application support/microsoft/teams/gpucache", "Teams GPU cache"},
	{"containers/com.docker.docker/data/log", "Docker Desktop logs"},
	{"application support/slack/cache", "Slack cache"},
	{"application support/slack/code cache", "Slack code cache"},
	{"application support/slack/gpucache", "Slack GPU cache"},
	{"application support/slack/service worker/cachestorage", "Slack service worker cache"},
	{"roaming/slack/cache", "Slack cache"},
	{"roaming/slack/code cache", "Slack code cache"},
	{"roaming/slack/gpucache", "Slack GPU cache"},
	{"roaming/slack/service worker/cachestorage", "Slack service worker cache"},
	{"application support/discord/cache", "Discord cache"},
	{"application support/discord/code cache", "Discord code cache"},
	{"application support/discord/gpucache", "Discord GPU cache"},
	{"application support/discord/service worker/cachestorage", "Discord service worker cache"},
	{"application support/legcord/cache", "Legcord cache"},
	{"application support/legcord/code cache", "Legcord code cache"},
	{"application support/legcord/gpucache", "Legcord GPU cache"},
	{"application support/legcord/logs", "Legcord logs"},
	{"application support/steam/htmlcache", "Steam web cache"},
	{"application support/steam/appcache", "Steam app cache"},
	{"application support/steam/depotcache", "Steam depot cache"},
	{"application support/steam/steamapps/shadercache", "Steam shader cache"},
	{"application support/steam/logs", "Steam logs"},
	{"application support/battle.net/cache", "Battle.net app cache"},
	{"roaming/discord/cache", "Discord cache"},
	{"roaming/discord/code cache", "Discord code cache"},
	{"roaming/discord/gpucache", "Discord GPU cache"},
	{"roaming/discord/service worker/cachestorage", "Discord service worker cache"},
	{"library/caches/notion.id", "Notion cache"},
	{"library/caches/md.obsidian", "Obsidian cache"},
	{"library/caches/com.runningwithcrayons.alfred", "Alfred cache"},
	{"library/caches/com.microsoft.teams2", "Teams cache"},
	{"library/caches/us.zoom.xos", "Zoom cache"},
	{"library/caches/ru.keepcoder.telegram", "Telegram cache"},
	{"library/caches/com.tencent.xinwechat", "WeChat cache"},
	{"library/caches/com.skype.skype", "Skype cache"},
	{"library/caches/net.whatsapp.whatsapp", "WhatsApp cache"},
	{"library/caches/com.todoist.mac.todoist", "Todoist cache"},
	{"library/caches/com.valvesoftware.steam", "Steam cache"},
	{"library/caches/com.epicgames.epicgameslauncher", "Epic Games cache"},
	{"library/caches/com.blizzard.battle.net", "Battle.net cache"},
	{"library/caches/com.colliderli.iina", "IINA cache"},
	{"library/caches/org.videolan.vlc", "VLC cache"},
	{"library/caches/io.mpv", "MPV cache"},
	{"library/caches/tv.plex.player.desktop", "Plex cache"},
	{"library/caches/com.readdle.smartemail-mac", "Spark cache"},
	{"library/caches/com.mongodb.compass", "MongoDB Compass cache"},
	{"application support/mongodb compass/cache", "MongoDB Compass cache"},
	{"application support/mongodb compass/code cache", "MongoDB Compass code cache"},
	{"application support/mongodb compass/gpucache", "MongoDB Compass GPU cache"},
	{"library/caches/com.redis.redisinsight", "Redis Insight cache"},
	{"application support/redisinsight/cache", "Redis Insight cache"},
	{"application support/redisinsight/code cache", "Redis Insight code cache"},
	{"application support/redisinsight/gpucache", "Redis Insight GPU cache"},
	{"library/caches/com.prect.navicatpremium", "Navicat cache"},
	{"library/caches/net.shinyfrog.bear", "Bear cache"},
	{"library/caches/com.evernote.evernote", "Evernote cache"},
	{"application support/logseq/cache", "Logseq cache"},
	{"application support/logseq/code cache", "Logseq code cache"},
	{"application support/logseq/gpucache", "Logseq GPU cache"},
	{"application support/logseq/service worker/cachestorage", "Logseq service worker cache"},
	{"application support/logseq/logs", "Logseq logs"},
	{"library/caches/pl.maketheweb.cleanshotx", "CleanShot cache"},
	{"library/caches/com.charliemonroe.downie-4", "Downie cache"},
	{"library/caches/com.charlessoft.pacifist", "Pacifist cache"},
	{"application support/riot client/cache", "Riot Client cache"},
	{"application support/riot client/code cache", "Riot Client code cache"},
	{"application support/riot client/gpucache", "Riot Client GPU cache"},
	{"application support/riot client/logs", "Riot Client logs"},
	{"application support/minecraft/webcache2", "Minecraft web cache"},
	{"application support/minecraft/logs", "Minecraft logs"},
	{"application support/lunarclient/cache", "Lunar Client cache"},
	{"application support/lunarclient/logs", "Lunar Client logs"},
	{"application support/lark/cache", "Feishu cache"},
	{"application support/lark/code cache", "Feishu code cache"},
	{"application support/lark/gpucache", "Feishu GPU cache"},
	{"application support/lark/logs", "Feishu logs"},
	{"application support/dingtalk/cache", "DingTalk cache"},
	{"application support/dingtalk/code cache", "DingTalk code cache"},
	{"application support/dingtalk/gpucache", "DingTalk GPU cache"},
	{"application support/dingtalk/logs", "DingTalk logs"},
	{"library/caches/com.anydesk.anydesk", "AnyDesk cache"},
	{"library/caches/com.gog.galaxy", "GOG Galaxy cache"},
	{"library/caches/com.ea.app", "EA app cache"},
	{"library/caches/com.klee.desktop", "Klee cache"},
	{"library/caches/klee_desktop", "Klee desktop cache"},
	{"library/caches/com.orabrowser.app", "Ora Browser cache"},
	{"library/caches/com.filo.client", "Filo cache"},
	{"application support/filo/production/cache", "Filo cache"},
	{"application support/filo/production/code cache", "Filo code cache"},
	{"application support/filo/production/gpucache", "Filo GPU cache"},
	{"application support/filo/production/dawngraphitecache", "Filo Dawn cache"},
	{"application support/filo/production/dawnwebgpucache", "Filo WebGPU cache"},
	{"library/caches/net.xmac.aria2gui", "Aria2 cache"},
	{"library/caches/com.folx.", "Folx cache"},
	{"library/caches/com.yinxiang.", "Yinxiang Note cache"},
	{".cacher/logs", "Cacher logs"},
	{".kite/logs", "Kite logs"},
	{"library/caches/com.runjuu.input-source-pro", "Input Source Pro cache"},
	{"library/caches/macos-wakatime.wakatime", "WakaTime cache"},
	{"library/caches/com.tencent.meeting", "Tencent Meeting cache"},
	{"library/caches/com.tencent.weworkmac", "WeCom cache"},
	{"library/caches/com.teamviewer.", "TeamViewer cache"},
	{"library/caches/com.todesk.", "ToDesk cache"},
	{"library/caches/com.sunlogin.", "Sunlogin cache"},
	{"library/caches/com.airmail.", "Airmail cache"},
	{"library/caches/com.any.do.", "Any.do cache"},
	{"library/caches/cx.c3.theunarchiver", "The Unarchiver cache"},
	{"library/caches/com.youdao.youdaodict", "Youdao Dictionary cache"},
	{"library/caches/com.eudic.", "Eudict cache"},
	{"library/caches/com.bob-build.bob", "Bob Translation cache"},
	{"library/caches/com.tw93.miaoyan", "MiaoYan cache"},
	{"library/caches/com.flomoapp.mac", "Flomo cache"},
	{"application support/quark/cache/videocache", "Quark video cache"},
	{"library/caches/com.maxon.cinema4d", "Cinema 4D cache"},
	{"library/caches/com.autodesk.", "Autodesk cache"},
	{"library/caches/com.sketchup.", "SketchUp cache"},
	{"library/caches/com.netease.163music", "NetEase Music cache"},
	{"library/caches/com.tencent.qqmusic", "QQ Music cache"},
	{"library/caches/com.kugou.mac", "Kugou Music cache"},
	{"library/caches/com.kuwo.mac", "Kuwo Music cache"},
	{"library/caches/com.iqiyi.player", "iQIYI cache"},
	{"library/caches/com.tencent.tenvideo", "Tencent Video cache"},
	{"library/caches/tv.danmaku.bili", "Bilibili cache"},
	{"library/caches/com.douyu.", "Douyu cache"},
	{"library/caches/com.huya.", "Huya cache"},
	{"library/caches/com.reincubate.camo", "Camo cache"},
	{"library/caches/com.xnipapp.xnip", "Xnip cache"},
	{"library/caches/org.m0k.transmission", "Transmission cache"},
	{"library/caches/com.qbittorrent.qbittorrent", "qBittorrent cache"},
	{"roaming/code/gpucache", "VS Code GPU cache"},
	{"ms-playwright", "Playwright browser cache"},
	{"puppeteer", "Puppeteer browser cache"},
	{"library/developer/xcode/ios devicesupport", "Xcode iOS device support"},
	{"library/developer/xcode/watchos devicesupport", "Xcode watchOS device support"},
	{"library/developer/xcode/tvos devicesupport", "Xcode tvOS device support"},
	{"google/chrome/user data/default/code cache", "Chrome code cache"},
	{"google/chrome/user data/default/gpucache", "Chrome GPU cache"},
	{"google/chrome/user data/default/grshadercache", "Chrome shader cache"},
	{"google/chrome/user data/default/cache", "Chrome cache"},
	{"microsoft/edge/user data/default/code cache", "Edge code cache"},
	{"microsoft/edge/user data/default/gpucache", "Edge GPU cache"},
	{"microsoft/edge/user data/default/grshadercache", "Edge shader cache"},
	{"microsoft/edge/user data/default/cache", "Edge cache"},
	{"bravesoftware/brave-browser/user data/default/code cache", "Brave code cache"},
	{"bravesoftware/brave-browser/user data/default/gpucache", "Brave GPU cache"},
	{"bravesoftware/brave-browser/user data/default/grshadercache", "Brave shader cache"},
	{"bravesoftware/brave-browser/user data/default/cache", "Brave cache"},
	{"application support/google/chrome/default/code cache", "Chrome code cache"},
	{"application support/google/chrome/default/gpucache", "Chrome GPU cache"},
	{"application support/google/chrome/default/grshadercache", "Chrome shader cache"},
	{"application support/google/chrome/default/cache", "Chrome cache"},
	{"application support/microsoft edge/default/code cache", "Edge code cache"},
	{"application support/microsoft edge/default/gpucache", "Edge GPU cache"},
	{"application support/microsoft edge/default/grshadercache", "Edge shader cache"},
	{"application support/microsoft edge/default/cache", "Edge cache"},
	{"application support/bravesoftware/brave-browser/default/code cache", "Brave code cache"},
	{"application support/bravesoftware/brave-browser/default/gpucache", "Brave GPU cache"},
	{"application support/bravesoftware/brave-browser/default/grshadercache", "Brave shader cache"},
	{"application support/bravesoftware/brave-browser/default/cache", "Brave cache"},
	{"mozilla/firefox/profiles", "Firefox profile caches"},
	{"containers/com.apple.safari/data/library/caches", "Safari cache"},
	{"mail downloads", "Mail downloads"},
	{"crashdumps", "Crash dumps"},
	{"wer/reportarchive", "Windows error reports"},
	{"wer/reportqueue", "Windows crash queue"},
	{"docker/buildx/cache", "Docker Buildx cache"},
	{"docker", "Docker cache"},
	{".turbo/cache", "Turbo cache"},
	{".parcel-cache", "Parcel cache"},
	{".cache/vite", "Vite cache"},
	{".cache/webpack", "Webpack cache"},
	{".cache/eslint", "ESLint cache"},
	{".cache/prettier", "Prettier cache"},
	{"scoop/cache", "Scoop cache"},
	{"chocolatey/cache", "Chocolatey cache"},
	{"winget/cache", "WinGet cache"},
	{"nuget/v3-cache", "NuGet cache"},
	// Cloud storage caches
	{"library/caches/com.getdropbox.dropbox", "Dropbox cache"},
	{"library/caches/com.dropbox.client2", "Dropbox cache"},
	{"library/caches/com.google.googledrive", "Google Drive cache"},
	{"library/caches/com.microsoft.onedrive", "OneDrive cache"},
	{"library/caches/com.baidu.netdisk", "Baidu Netdisk cache"},
	{"library/caches/com.box.desktop", "Box cache"},
	// Virtualization caches
	{"library/caches/com.vmware.fusion", "VMware Fusion cache"},
	{"library/caches/com.parallels.desktop.launch", "Parallels cache"},
	{"virtualbox vms/.cache", "VirtualBox cache"},
	{".vagrant.d/tmp", "Vagrant temp files"},
}
