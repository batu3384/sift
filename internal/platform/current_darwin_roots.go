//go:build darwin

package platform

import (
	"os"
	"path/filepath"
	"strings"
)

func (d darwinAdapter) CuratedRoots() CuratedRoots {
	manifest := darwinCleanManifestRoots(d.home)
	roots := CuratedRoots{
		Temp: []string{
			os.TempDir(),
			filepath.Join(d.home, "Library", "Caches"),
			filepath.Join(d.home, "Library", "Application Support", "com.apple.wallpaper", "aerials", "videos"),
			filepath.Join(d.home, "Library", "Containers", "com.apple.wallpaper.agent", "Data", "Library", "Caches"),
			filepath.Join(d.home, "Library", "Application Support", "com.apple.idleassetsd"),
			"/Library/Application Support/com.apple.idleassetsd/Customer",
			filepath.Join(d.home, "Library", "Messages", "StickerCache"),
			filepath.Join(d.home, "Library", "Messages", "Caches", "Previews", "StickerCache"),
			filepath.Join(d.home, "Library", "Messages", "Caches", "Previews", "Attachments"),
			filepath.Join(d.home, "Library", "Caches", "com.apple.QuickLook.thumbnailcache"),
			filepath.Join(d.home, "Library", "Caches", "Quick Look"),
			filepath.Join(d.home, "Library", "Autosave Information"),
			filepath.Join(d.home, "Library", "IdentityCaches"),
			filepath.Join(d.home, "Library", "Suggestions"),
			filepath.Join(d.home, "Library", "Calendars", "Calendar Cache"),
			filepath.Join(d.home, "Library", "Caches", "io.mpv"),
			filepath.Join(d.home, "Library", "Caches", "md.obsidian"),
			filepath.Join(d.home, "Library", "Application Support", "Figma", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "Figma", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "Cursor", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "Cursor", "GPUCache"),
			filepath.Join(d.home, "Library", "Caches", "com.anthropic.claudefordesktop"),
			filepath.Join(d.home, "Library", "Caches", "com.openai.chat"),
			filepath.Join(d.home, "Library", "Caches", "tv.plex.player.desktop"),
			filepath.Join(d.home, "Library", "Caches", "com.runningwithcrayons.Alfred"),
			filepath.Join(d.home, "Library", "Application Support", "Postman", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "Postman", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "VSCodium", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "VSCodium", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "Zed", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "ChatGPT", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "ChatGPT", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "Zed", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "Legcord", "GPUCache"),
			filepath.Join(d.home, "Library", "Caches", "com.valvesoftware.steam"),
			filepath.Join(d.home, "Library", "Caches", "com.epicgames.EpicGamesLauncher"),
			filepath.Join(d.home, "Library", "Caches", "us.zoom.xos"),
			filepath.Join(d.home, "Library", "Caches", "notion.id"),
			filepath.Join(d.home, "Library", "Caches", "com.blizzard.Battle.net"),
			filepath.Join(d.home, "Library", "Application Support", "Battle.net", "Cache"),
			filepath.Join(d.home, "Library", "Caches", "com.todoist.mac.Todoist"),
			filepath.Join(d.home, "Library", "Caches", "com.microsoft.teams2"),
			filepath.Join(d.home, "Library", "Application Support", "Microsoft", "Teams", "Cache"),
			filepath.Join(d.home, "Library", "Application Support", "Microsoft", "Teams", "Code Cache"),
			filepath.Join(d.home, "Library", "Application Support", "Microsoft", "Teams", "GPUCache"),
			filepath.Join(d.home, "Library", "Caches", "com.colliderli.iina"),
			filepath.Join(d.home, "Library", "Caches", "org.videolan.vlc"),
			filepath.Join(d.home, "Library", "Caches", "ru.keepcoder.Telegram"),
			filepath.Join(d.home, "Library", "Caches", "com.tencent.xinWeChat"),
			filepath.Join(d.home, "Library", "Caches", "net.whatsapp.WhatsApp"),
			filepath.Join(d.home, "Library", "Caches", "com.readdle.smartemail-Mac"),
			filepath.Join(d.home, "Library", "Application Support", "com.bohemiancoding.sketch3", "cache"),
			filepath.Join(d.home, "Library", "Application Support", "Claude", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "Claude", "DawnGraphiteCache"),
			filepath.Join(d.home, "Library", "Application Support", "Claude", "DawnWebGPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "Claude", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "Adobe", "Common", "Media Cache Files"),
			filepath.Join(d.home, "Library", "Application Support", "Steam", "htmlcache"),
			filepath.Join(d.home, "Library", "Application Support", "Steam", "appcache"),
			filepath.Join(d.home, "Library", "Application Support", "Steam", "depotcache"),
			filepath.Join(d.home, "Library", "Application Support", "Steam", "steamapps", "shadercache"),
			filepath.Join(d.home, "Library", "Application Support", "Discord", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "Discord", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Application Support", "Slack", "GPUCache"),
			filepath.Join(d.home, "Library", "Application Support", "Slack", "Service Worker", "CacheStorage"),
			filepath.Join(d.home, "Library", "Caches", "com.skype.skype"),
		},
		Logs: []string{
			filepath.Join(d.home, "Library", "Logs"),
			filepath.Join(d.home, "Library", "Logs", "DiagnosticReports"),
			filepath.Join(d.home, "Library", "Logs", "CoreSimulator"),
			filepath.Join(d.home, "Library", "Logs", "Figma"),
			filepath.Join(d.home, "Library", "Logs", "Postman"),
			filepath.Join(d.home, "Library", "Logs", "Zed"),
			filepath.Join(d.home, "Library", "Logs", "ChatGPT"),
			filepath.Join(d.home, "Library", "Containers", "com.docker.docker", "Data", "log"),
			filepath.Join(d.home, "Library", "Application Support", "ChatGPT", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Claude", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Figma", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "VSCodium", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Steam", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Microsoft", "Teams", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Postman", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Zed", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Cursor", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Legcord", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Logseq", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Riot Client", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "minecraft", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "lunarclient", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "Lark", "logs"),
			filepath.Join(d.home, "Library", "Application Support", "DingTalk", "logs"),
		},
		Developer: []string{
			filepath.Join(d.home, ".cache"),
			filepath.Join(d.home, ".cache", "go-build"),
			filepath.Join(d.home, ".npm"),
			filepath.Join(d.home, ".npm", "_cacache"),
			filepath.Join(d.home, ".cargo"),
			filepath.Join(d.home, ".cargo", "registry", "cache"),
			filepath.Join(d.home, ".cargo", "git", "db"),
			filepath.Join(d.home, ".gradle", "caches"),
			filepath.Join(d.home, ".cache", "puppeteer"),
			filepath.Join(d.home, ".cache", "ms-playwright"),
			filepath.Join(d.home, "Library", "Developer", "Xcode", "DerivedData"),
			filepath.Join(d.home, "Library", "Developer", "Xcode", "Archives"),
			filepath.Join(d.home, "Library", "Developer", "Xcode", "iOS DeviceSupport"),
			filepath.Join(d.home, "Library", "Developer", "CoreSimulator", "Caches"),
			filepath.Join(d.home, "Library", "Caches", "JetBrains"),
			filepath.Join(d.home, "Library", "Caches", "com.konghq.insomnia"),
			filepath.Join(d.home, "Library", "Caches", "com.microsoft.VSCode"),
			filepath.Join(d.home, "Library", "Caches", "com.tinyapp.TablePlus"),
			filepath.Join(d.home, "Library", "Caches", "com.github.GitHubDesktop"),
			filepath.Join(d.home, "Library", "Caches", "com.getpaw.Paw"),
			filepath.Join(d.home, "Library", "Caches", "com.charlesproxy.charles"),
			filepath.Join(d.home, "Library", "Caches", "com.proxyman.NSProxy"),
			filepath.Join(d.home, "Library", "Application Support", "Code", "Cache"),
			filepath.Join(d.home, "Library", "Application Support", "Code", "CachedExtensions"),
			filepath.Join(d.home, "Library", "Application Support", "Code", "DawnGraphiteCache"),
			filepath.Join(d.home, "Library", "Application Support", "Code", "DawnWebGPUCache"),
		},
		Browser: []string{
			filepath.Join(d.home, "Library", "Caches", "Google", "Chrome"),
			filepath.Join(d.home, "Library", "Caches", "Google", "Chrome", "Default", "Cache"),
			filepath.Join(d.home, "Library", "Caches", "Firefox"),
			filepath.Join(d.home, "Library", "Caches", "Safari"),
			filepath.Join(d.home, "Library", "Application Support", "Google", "Chrome"),
			filepath.Join(d.home, "Library", "Application Support", "Microsoft", "EdgeUpdater", "apps", "msedge-stable"),
		},
		Installer: []string{
			filepath.Join(d.home, "Downloads"),
			filepath.Join(d.home, "Desktop"),
			filepath.Join(d.home, "Documents"),
			filepath.Join(d.home, "Public"),
			filepath.Join("/Users", "Shared"),
			filepath.Join("/Users", "Shared", "Downloads"),
			filepath.Join(d.home, "Library", "Mobile Documents", "com~apple~CloudDocs", "Downloads"),
			filepath.Join(d.home, "Library", "Application Support", "Telegram Desktop"),
			filepath.Join(d.home, "Downloads", "Telegram Desktop"),
		},
		PackageManager: []string{
			filepath.Join(d.home, "Library", "Caches", "Homebrew"),
			filepath.Join(d.home, "Library", "Caches", "pip"),
			filepath.Join(d.home, "Library", "Caches", "pnpm"),
			filepath.Join(d.home, "Library", "Caches", "Yarn"),
		},
		AppSupport: []string{
			filepath.Join(d.home, "Library", "Application Support"),
			filepath.Join(d.home, "Library", "Containers"),
			filepath.Join(d.home, "Library", "Preferences"),
		},
		RecentItems: []string{
			filepath.Join(d.home, "Library", "Application Support", "com.apple.sharedfilelist", "com.apple.LSSharedFileList.RecentApplications.sfl2"),
			filepath.Join(d.home, "Library", "Application Support", "com.apple.sharedfilelist", "com.apple.LSSharedFileList.RecentDocuments.sfl2"),
			filepath.Join(d.home, "Library", "Preferences", "com.apple.recentitems.plist"),
		},
		SystemUpdate: []string{
			filepath.Join(d.home, "Library", "Updates"),
			"/Library/Updates",
			"/Library/Apple/usr/share/rosetta/rosetta_update_bundle",
			filepath.Join(d.home, "Library", "Caches", "com.apple.rosetta.update"),
			filepath.Join(d.home, "Library", "Caches", "com.apple.amp.mediasevicesd"),
		},
		CloudOffice: []string{
			// Dropbox
			filepath.Join(d.home, "Library", "Caches", "com.getdropbox.dropbox"),
			filepath.Join(d.home, "Library", "Application Support", "Dropbox"),
			// OneDrive
			filepath.Join(d.home, "Library", "Caches", "com.microsoft.OneDrive"),
			filepath.Join(d.home, "Library", "Application Support", "OneDrive"),
			// Google Drive
			filepath.Join(d.home, "Library", "Caches", "com.google.GoogleDrive"),
			filepath.Join(d.home, "Library", "Application Support", "Google", "DriveFS"),
			// Microsoft Teams
			filepath.Join(d.home, "Library", "Caches", "com.microsoft.teams2"),
			filepath.Join(d.home, "Library", "Application Support", "Microsoft", "Teams"),
			// Slack
			filepath.Join(d.home, "Library", "Caches", "com.slack.Slack"),
			filepath.Join(d.home, "Library", "Application Support", "Slack"),
			// Notion
			filepath.Join(d.home, "Library", "Caches", "com.notion.id"),
			filepath.Join(d.home, "Library", "Application Support", "Notion"),
			// iCloud
			filepath.Join(d.home, "Library", "Caches", "com.apple.CloudDocs"),
			filepath.Join(d.home, "Library", "Mobile Documents"),
			// Discord
			filepath.Join(d.home, "Library", "Caches", "com.hnc.Discord"),
			filepath.Join(d.home, "Library", "Application Support", "Discord"),
			// Zoom
			filepath.Join(d.home, "Library", "Caches", "us.zoom.xos"),
			filepath.Join(d.home, "Library", "Application Support", "zoom.us"),
			// WebEx
			filepath.Join(d.home, "Library", "Caches", "com.webex.meetingmanager"),
			filepath.Join(d.home, "Library", "Application Support", "Webex"),
		},
		Virtualization: []string{
			// Docker
			filepath.Join(d.home, "Library", "Caches", "com.docker.docker"),
			filepath.Join(d.home, "Library", "Application Support", "Docker"),
			filepath.Join(d.home, "Library", "Containers", "com.docker.docker"),
			// VMware
			filepath.Join(d.home, "Library", "Caches", "com.vmware.fusion"),
			filepath.Join(d.home, "Library", "Application Support", "VMware"),
			filepath.Join(d.home, "Library", "Preferences", "com.vmware.fusion"),
			// Parallels
			filepath.Join(d.home, "Library", "Caches", "com.parallels.desktop.console"),
			filepath.Join(d.home, "Library", "Application Support", "Parallels"),
			// VirtualBox
			filepath.Join(d.home, "VirtualBox VMs"),
			filepath.Join(d.home, ".config", "VirtualBox"),
			// Vagrant
			filepath.Join(d.home, ".vagrant.d", "cache"),
			filepath.Join(d.home, ".vagrant.d", "boxes"),
			// Podman
			filepath.Join(d.home, "Library", "Caches", "io.podman"),
			filepath.Join(d.home, ".local", "share", "containers"),
			// Lima (Lima VM)
			filepath.Join(d.home, "Library", "Caches", "dev.lima"),
			filepath.Join(d.home, ".lima"),
			// QEMU
			filepath.Join(d.home, ".config", "qemu"),
			filepath.Join(d.home, ".local", "share", "qemu"),
		},
		DeviceBackups: []string{
			// iOS Device Backups
			filepath.Join(d.home, "Library", "Application Support", "MobileSync"),
			filepath.Join(d.home, "Library", "Application Support", "MobileSync", "Backup"),
			// iTunes/Finder Backup
			filepath.Join(d.home, "Library", "Application Support", "MobileSync", "Backup"),
			// Xcode iOS Simulator
			filepath.Join(d.home, "Library", "Developer", "CoreSimulator", "Devices"),
			filepath.Join(d.home, "Library", "Developer", "Xcode", "iOS DeviceSupport"),
			// Android SDK
			filepath.Join(d.home, "Library", "Caches", "Android"),
			filepath.Join(d.home, ".android"),
			filepath.Join(d.home, ".gradle"),
			// ADB
			filepath.Join(d.home, ".local", "share", "android"),
		},
		TimeMachine: []string{
			// Time Machine
			filepath.Join(d.home, "Library", "Application Support", "TimeMachine"),
			filepath.Join(d.home, "Library", "Caches", "com.apple.TimeMachine"),
			// Local Time Machine Snapshots
			"/Volumes",
			// Backups.backupdb
			filepath.Join(d.home, "Library", "Application Support", "com.apple.backupd"),
		},
		// Font Cache
		FontCache: []string{
			filepath.Join(d.home, "Library", "Caches", "com.apple.fonts"),
			"/Library/Caches/com.apple.fonts",
			filepath.Join(d.home, "Library", "Fonts"),
			"/Library/Fonts",
		},
		// Print Spooler
		PrintSpooler: []string{
			filepath.Join(d.home, "Library", "Printers"),
			"/var/spool/cups",
			filepath.Join(d.home, "Library", "Application Support", "CUPS"),
		},
		// Xcode
		Xcode: []string{
			filepath.Join(d.home, "Library", "Developer", "Xcode", "DerivedData"),
			filepath.Join(d.home, "Library", "Developer", "Xcode", "Archives"),
			filepath.Join(d.home, "Library", "Developer", "Xcode", "iOS DeviceSupport"),
			filepath.Join(d.home, "Library", "Developer", "CoreSimulator", "Caches"),
			filepath.Join(d.home, "Library", "Developer", "CoreSimulator", "Devices"),
			filepath.Join(d.home, "Library", "Caches", "com.apple.dt.Xcode"),
			filepath.Join(d.home, "Library", "Developer", "Xcode", "Products"),
		},
		// Unity
		Unity: []string{
			filepath.Join(d.home, "Library", "Caches", "com.unity3d"),
			filepath.Join(d.home, "Library", "Application Support", "Unity"),
			filepath.Join(d.home, "Library", "Application Support", "Unity", "Editor"),
		},
		// Unreal Engine
		Unreal: []string{
			filepath.Join(d.home, "Library", "Caches", "com.epicgames.UnrealEngine"),
			filepath.Join(d.home, "Library", "Application Support", "Unreal Engine"),
		},
		// Android
		Android: []string{
			filepath.Join(d.home, "Library", "Caches", "Android"),
			filepath.Join(d.home, ".android"),
			filepath.Join(d.home, ".gradle"),
			filepath.Join(d.home, "Library", "Application Support", "Android"),
			filepath.Join(d.home, ".local", "share", "android"),
		},
		// Rust
		Rust: []string{
			filepath.Join(d.home, ".cargo", "registry"),
			filepath.Join(d.home, ".cargo", "git"),
			filepath.Join(d.home, "Library", "Caches", "rustup"),
		},
		// Node.js
		NodeModules: []string{
			filepath.Join(d.home, "node_modules"),
			filepath.Join(d.home, ".npm"),
			filepath.Join(d.home, ".npm", "_cacache"),
		},
		// Python
		PythonCache: []string{
			filepath.Join(d.home, "Library", "Caches", "pip"),
			filepath.Join(d.home, ".cache", "pip"),
			filepath.Join(d.home, "Library", "Caches", "python"),
			filepath.Join(d.home, ".local", "lib", "python"),
		},
		// Go
		GoCache: []string{
			filepath.Join(d.home, "Library", "Caches", "go-build"),
			filepath.Join(d.home, ".cache", "go-build"),
			filepath.Join(d.home, "go", "pkg", "mod"),
		},
		// Fonts
		Fonts: []string{
			filepath.Join(d.home, "Library", "Fonts"),
			"/Library/Fonts",
			filepath.Join(d.home, ".fonts"),
			filepath.Join(d.home, ".local", "share", "fonts"),
		},
		// Diagnostics
		Diagnostics: []string{
			filepath.Join(d.home, "Library", "Logs", "DiagnosticReports"),
			filepath.Join(d.home, "Library", "Logs", "CoreSimulator"),
			filepath.Join(d.home, "Library", "Logs", "CrashReporter"),
			"/var/log",
		},
		// Media
		Media: []string{
			filepath.Join(d.home, "Library", "Caches", "com.apple.AppleMediaServices"),
			filepath.Join(d.home, "Library", "Caches", "tv.plex.player.desktop"),
			filepath.Join(d.home, "Library", "Application Support", "Plex"),
			filepath.Join(d.home, "Library", "Application Support", "VLC"),
			filepath.Join(d.home, "Library", "Caches", "com.spotify.client"),
			filepath.Join(d.home, "Library", "Application Support", "Spotify"),
		},
	}
	roots.Temp = append(roots.Temp, manifest.Temp...)
	roots.Logs = append(roots.Logs, manifest.Logs...)
	roots.Developer = append(roots.Developer, manifest.Developer...)
	roots.Temp = dedupePaths(roots.Temp)
	roots.Logs = dedupePaths(roots.Logs)
	roots.Developer = dedupePaths(roots.Developer)
	return roots
}

func (d darwinAdapter) ProtectedPaths() []string {
	return []string{
		"/System",
		"/Applications",
		"/Library",
		"/usr",
		"/bin",
		"/sbin",
		filepath.Join(d.home, "Library", "Safari"),
		filepath.Join(d.home, "Library", "Keychains"),
		filepath.Join(d.home, ".ssh"),
		filepath.Join(d.home, ".gnupg"),
		filepath.Join(d.home, "Library", "Application Support", "Raycast"),
		filepath.Join(d.home, "Library", "Application Support", "Alfred"),
		filepath.Join(d.home, "Library", "Containers", "com.raycast.macos.BrowserExtension"),
		filepath.Join(d.home, "Library", "Mobile Documents"),
	}
}

func (d darwinAdapter) ResolveTargets(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		// Use adapter's home directory for consistent path handling
		if item == "~" {
			out = append(out, d.home)
			continue
		}
		if strings.HasPrefix(item, "~/") {
			out = append(out, filepath.Join(d.home, item[2:]))
			continue
		}
		out = append(out, item)
	}
	return out
}
