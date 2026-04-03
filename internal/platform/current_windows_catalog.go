//go:build windows

package platform

import (
	"context"
	"os"
	"path/filepath"

	"github.com/batu3384/sift/internal/domain"
	"golang.org/x/sys/windows/registry"
)

func (w windowsAdapter) CuratedRoots() CuratedRoots {
	return CuratedRoots{
		Temp: []string{
			os.TempDir(),
			filepath.Join(w.localApp, "Temp"),
			filepath.Join(w.localApp, "Packages"),
			filepath.Join(w.roaming, "Figma", "Cache"),
			filepath.Join(w.roaming, "Figma", "Code Cache"),
			filepath.Join(w.roaming, "Figma", "GPUCache"),
			filepath.Join(w.roaming, "Figma", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "Postman", "Cache"),
			filepath.Join(w.roaming, "Postman", "Code Cache"),
			filepath.Join(w.roaming, "Postman", "GPUCache"),
			filepath.Join(w.roaming, "Postman", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "Zed", "Cache"),
			filepath.Join(w.roaming, "Zed", "Code Cache"),
			filepath.Join(w.roaming, "Zed", "GPUCache"),
			filepath.Join(w.roaming, "Zed", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "Claude", "Cache"),
			filepath.Join(w.roaming, "Claude", "Code Cache"),
			filepath.Join(w.roaming, "Claude", "GPUCache"),
			filepath.Join(w.roaming, "Claude", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "ChatGPT", "Cache"),
			filepath.Join(w.roaming, "ChatGPT", "Code Cache"),
			filepath.Join(w.roaming, "ChatGPT", "GPUCache"),
			filepath.Join(w.roaming, "ChatGPT", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "Cursor", "Cache"),
			filepath.Join(w.roaming, "Cursor", "Code Cache"),
			filepath.Join(w.roaming, "Cursor", "GPUCache"),
			filepath.Join(w.roaming, "Cursor", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "VSCodium", "Cache"),
			filepath.Join(w.roaming, "VSCodium", "Code Cache"),
			filepath.Join(w.roaming, "VSCodium", "GPUCache"),
			filepath.Join(w.roaming, "VSCodium", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "Slack", "Cache"),
			filepath.Join(w.roaming, "Slack", "Code Cache"),
			filepath.Join(w.roaming, "Slack", "GPUCache"),
			filepath.Join(w.roaming, "Slack", "Service Worker", "CacheStorage"),
			filepath.Join(w.roaming, "discord", "Cache"),
			filepath.Join(w.roaming, "discord", "Code Cache"),
			filepath.Join(w.roaming, "discord", "GPUCache"),
			filepath.Join(w.roaming, "discord", "Service Worker", "CacheStorage"),
		},
		Logs: []string{
			filepath.Join(w.localApp, "CrashDumps"),
			filepath.Join(w.localApp, "Microsoft", "Windows", "WER", "ReportArchive"),
			filepath.Join(w.localApp, "Microsoft", "Windows", "WER", "ReportQueue"),
			filepath.Join(w.localApp, "JetBrains", "Toolbox", "logs"),
			filepath.Join(w.localApp, "Docker", "log"),
			filepath.Join(w.roaming, "Figma", "logs"),
			filepath.Join(w.roaming, "Postman", "logs"),
			filepath.Join(w.roaming, "Zed", "logs"),
			filepath.Join(w.roaming, "Claude", "logs"),
			filepath.Join(w.roaming, "ChatGPT", "logs"),
			filepath.Join(w.roaming, "sift", "logs"),
			filepath.Join(w.roaming, "Code", "logs"),
			filepath.Join(w.roaming, "Cursor", "logs"),
			filepath.Join(w.roaming, "VSCodium", "logs"),
			filepath.Join(w.localApp, "Packages", "MicrosoftWindows.Client.WebExperience"),
		},
		Developer: []string{
			filepath.Join(w.localApp, "npm-cache"),
			filepath.Join(w.roaming, "npm-cache"),
			filepath.Join(w.localApp, "npm-cache", "_logs"),
			filepath.Join(w.localApp, "npm-cache", "_npx"),
			filepath.Join(w.localApp, "Yarn", "Cache"),
			filepath.Join(w.roaming, "Yarn", "Cache"),
			filepath.Join(w.localApp, "Yarn", "Berry", "Cache"),
			filepath.Join(w.localApp, "pnpm-store"),
			filepath.Join(w.roaming, "pnpm-store"),
			filepath.Join(w.localApp, "bun", "install", "cache"),
			filepath.Join(w.localApp, "go-build"),
			filepath.Join(w.localApp, "uv", "cache"),
			filepath.Join(w.localApp, "pip", "Cache"),
			filepath.Join(w.localApp, "ms-playwright"),
			filepath.Join(w.localApp, "puppeteer"),
			filepath.Join(w.home, ".cache", "pip"),
			filepath.Join(w.home, ".cache", "poetry"),
			filepath.Join(w.home, ".cache", "ruff"),
			filepath.Join(w.home, ".cache", "mypy"),
			filepath.Join(w.home, ".pytest_cache"),
			filepath.Join(w.home, ".ruff_cache"),
			filepath.Join(w.home, ".mypy_cache"),
			filepath.Join(w.home, ".turbo", "cache"),
			filepath.Join(w.home, ".parcel-cache"),
			filepath.Join(w.home, ".cache", "vite"),
			filepath.Join(w.home, ".cache", "webpack"),
			filepath.Join(w.home, ".cache", "eslint"),
			filepath.Join(w.home, ".cache", "prettier"),
			filepath.Join(w.home, ".cargo", "registry"),
			filepath.Join(w.home, ".cargo", "registry", "cache"),
			filepath.Join(w.home, ".cargo", "git"),
			filepath.Join(w.home, ".cargo", "git", "db"),
			filepath.Join(w.home, ".nuget", "packages"),
			filepath.Join(w.home, ".gradle", "caches"),
			filepath.Join(w.home, ".m2", "repository"),
			filepath.Join(w.localApp, "JetBrains", "Toolbox", "cache"),
			filepath.Join(w.localApp, "Microsoft", "WindowsApps"),
			filepath.Join(w.localApp, "Docker"),
			filepath.Join(w.localApp, "Programs", "Microsoft VS Code", "Cache"),
			filepath.Join(w.roaming, "Code", "Cache"),
			filepath.Join(w.roaming, "Code", "CachedData"),
			filepath.Join(w.roaming, "Code", "GPUCache"),
			filepath.Join(w.roaming, "Figma", "CachedData"),
			filepath.Join(w.roaming, "Postman", "CachedData"),
			filepath.Join(w.roaming, "Zed", "CachedData"),
			filepath.Join(w.roaming, "Claude", "CachedData"),
			filepath.Join(w.roaming, "ChatGPT", "CachedData"),
			filepath.Join(w.roaming, "Cursor", "CachedData"),
			filepath.Join(w.roaming, "Cursor", "CachedExtensions"),
			filepath.Join(w.roaming, "Cursor", "GPUCache"),
			filepath.Join(w.roaming, "VSCodium", "CachedData"),
			filepath.Join(w.roaming, "VSCodium", "CachedExtensions"),
			filepath.Join(w.roaming, "VSCodium", "GPUCache"),
		},
		Browser: []string{
			filepath.Join(w.localApp, "Google", "Chrome", "User Data", "Default", "Cache"),
			filepath.Join(w.localApp, "Google", "Chrome", "User Data", "Default", "Code Cache"),
			filepath.Join(w.localApp, "Google", "Chrome", "User Data", "Default", "GPUCache"),
			filepath.Join(w.localApp, "Google", "Chrome", "User Data", "Default", "GrShaderCache"),
			filepath.Join(w.localApp, "Microsoft", "Edge", "User Data", "Default", "Cache"),
			filepath.Join(w.localApp, "Microsoft", "Edge", "User Data", "Default", "Code Cache"),
			filepath.Join(w.localApp, "Microsoft", "Edge", "User Data", "Default", "GPUCache"),
			filepath.Join(w.localApp, "Microsoft", "Edge", "User Data", "Default", "GrShaderCache"),
			filepath.Join(w.localApp, "BraveSoftware", "Brave-Browser", "User Data", "Default", "Cache"),
			filepath.Join(w.localApp, "BraveSoftware", "Brave-Browser", "User Data", "Default", "Code Cache"),
			filepath.Join(w.localApp, "BraveSoftware", "Brave-Browser", "User Data", "Default", "GPUCache"),
			filepath.Join(w.localApp, "BraveSoftware", "Brave-Browser", "User Data", "Default", "GrShaderCache"),
			filepath.Join(w.localApp, "Mozilla", "Firefox", "Profiles"),
		},
		Installer: []string{
			filepath.Join(w.home, "Downloads"),
			filepath.Join(w.home, "Desktop"),
			filepath.Join(w.localApp, "Temp"),
		},
		PackageManager: []string{
			filepath.Join(w.localApp, "Packages"),
			filepath.Join(w.localApp, "NuGet", "v3-cache"),
			filepath.Join(w.localApp, "pip", "Cache"),
			filepath.Join(w.localApp, "uv", "cache"),
			filepath.Join(w.localApp, "Microsoft", "WinGet", "Cache"),
			filepath.Join(w.localApp, "Yarn", "Cache"),
			filepath.Join(w.localApp, "pnpm-store"),
			filepath.Join(w.home, "scoop", "cache"),
			filepath.Join(w.programDT, "chocolatey", "cache"),
			filepath.Join(w.programDT, "chocolatey", "lib-bad"),
			filepath.Join(w.home, ".m2", "repository"),
		},
		AppSupport: []string{
			w.localApp,
			w.roaming,
			filepath.Join(w.localApp, "Packages"),
			w.programDT,
		},
	}
}

func (w windowsAdapter) ProtectedPaths() []string {
	return []string{
		os.Getenv("WINDIR"),
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		os.Getenv("ProgramData"),
		filepath.Join(w.localApp, "Google", "Chrome", "User Data"),
		filepath.Join(w.localApp, "Microsoft", "Edge", "User Data"),
		filepath.Join(w.localApp, "BraveSoftware", "Brave-Browser", "User Data"),
		filepath.Join(w.roaming, "Mozilla", "Firefox", "Profiles"),
		filepath.Join(w.localApp, "Mozilla", "Firefox", "Profiles"),
		filepath.Join(w.roaming, "Microsoft", "Credentials"),
		filepath.Join(w.roaming, "Microsoft", "Crypto"),
		filepath.Join(w.home, ".ssh"),
		filepath.Join(w.home, ".gnupg"),
		filepath.Join(w.home, ".aws"),
		filepath.Join(w.home, ".kube"),
	}
}

func (w windowsAdapter) MaintenanceTasks(_ context.Context) []domain.MaintenanceTask {
	return []domain.MaintenanceTask{
		{
			ID:          "windows.optimize.thumbcache",
			Title:       "Reset Explorer thumbnail cache",
			Description: "Removes stale thumbnail databases so Explorer can rebuild previews cleanly.",
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			PathGlobs: []string{
				filepath.Join(w.localApp, "Microsoft", "Windows", "Explorer", "thumbcache_*.db"),
			},
			Steps: []string{
				"Explorer thumbnails rebuild automatically",
				"Restart Explorer if previews stay stale",
			},
		},
		{
			ID:          "windows.optimize.iconcache",
			Title:       "Reset Explorer icon cache",
			Description: "Clears stale icon cache files so Windows can regenerate app and file icons.",
			Risk:        domain.RiskSafe,
			Action:      domain.ActionTrash,
			PathGlobs: []string{
				filepath.Join(w.localApp, "Microsoft", "Windows", "Explorer", "iconcache*.db"),
			},
			Steps: []string{
				"Icons rebuild after Explorer restart or sign out",
				"Use if shortcuts or file icons look stale",
			},
		},
		{
			ID:          "windows.review.startup",
			Title:       "Review startup apps",
			Description: "Reduce background load by reviewing startup entries before app cleanup.",
			Risk:        domain.RiskReview,
			Steps: []string{
				"Settings > Apps > Startup",
				"Disable rarely used apps first",
			},
		},
		{
			ID:          "windows.review.winget",
			Title:       "Review Winget/MSIX packages",
			Description: "SIFT scans package leftovers but does not auto-remove installed packages.",
			Risk:        domain.RiskSafe,
			Steps: []string{
				"Run winget list to review installed packages",
				"Use winget uninstall for package-managed apps when needed",
			},
		},
	}
}

func (w windowsAdapter) Diagnostics(_ context.Context) []Diagnostic {
	diagnostics := []Diagnostic{
		{Name: "platform", Status: "ok", Message: "windows adapter active"},
		{Name: "local_app_data", Status: "ok", Message: w.localApp},
		{Name: "roaming_app_data", Status: "ok", Message: w.roaming},
	}
	if w.programDT != "" {
		diagnostics = append(diagnostics, Diagnostic{Name: "program_data", Status: "ok", Message: w.programDT})
	}
	for _, path := range []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	} {
		key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
		if err != nil {
			continue
		}
		_ = key.Close()
		diagnostics = append(diagnostics, Diagnostic{Name: "registry_probe", Status: "ok", Message: path})
		break
	}
	return diagnostics
}
