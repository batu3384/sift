package engine

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/platform"
)

type protectedFamilySpec struct {
	id          string
	title       string
	description string
	roots       func(platform.Adapter) []string
}

func availableProtectedFamilies(adapter platform.Adapter) []domain.ProtectedFamily {
	specs := familySpecs(adapter)
	out := make([]domain.ProtectedFamily, 0, len(specs))
	for _, spec := range specs {
		out = append(out, domain.ProtectedFamily{
			ID:          spec.id,
			Title:       spec.title,
			Description: spec.description,
		})
	}
	return out
}

func familyProtectedRoots(adapter platform.Adapter, active []string) ([]string, []string) {
	specs := familySpecs(adapter)
	selected := make([]string, 0, len(active))
	var roots []string
	for _, family := range active {
		spec, ok := findFamilySpec(specs, family)
		if !ok {
			continue
		}
		selected = append(selected, spec.id)
		roots = append(roots, spec.roots(adapter)...)
	}
	return normalizePolicyPaths(roots), dedupeLower(selected)
}

func matchingFamilyIDs(adapter platform.Adapter, active []string, path string) []string {
	specs := familySpecs(adapter)
	matches := make([]string, 0, len(active))
	for _, family := range active {
		spec, ok := findFamilySpec(specs, family)
		if !ok {
			continue
		}
		for _, root := range normalizePolicyPaths(spec.roots(adapter)) {
			if domain.HasPathPrefix(path, root) {
				matches = append(matches, spec.id)
				break
			}
		}
	}
	return dedupeLower(matches)
}

func findFamilySpec(specs []protectedFamilySpec, id string) (protectedFamilySpec, bool) {
	key := strings.ToLower(strings.TrimSpace(id))
	for _, spec := range specs {
		if spec.id == key {
			return spec, true
		}
	}
	return protectedFamilySpec{}, false
}

func familySpecs(adapter platform.Adapter) []protectedFamilySpec {
	switch adapter.Name() {
	case "windows":
		return windowsFamilySpecs(adapter)
	default:
		return darwinFamilySpecs(adapter)
	}
}

func darwinFamilySpecs(adapter platform.Adapter) []protectedFamilySpec {
	roots := adapter.CuratedRoots()
	home := homeFromRoots(adapter)
	return []protectedFamilySpec{
		{
			id:          "browser_profiles",
			title:       "Browser Profiles",
			description: "Protect browser profiles, history, and session state while still allowing safe cache exceptions.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "Library", "Application Support", "Google", "Chrome"),
					filepath.Join(home, "Library", "Application Support", "Microsoft", "Edge"),
					filepath.Join(home, "Library", "Application Support", "BraveSoftware", "Brave-Browser"),
					filepath.Join(home, "Library", "Application Support", "Firefox"),
					filepath.Join(home, "Library", "Safari"),
				}
			},
		},
		{
			id:          "password_managers",
			title:       "Password Managers",
			description: "Protect credential vaults and local security databases.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "Library", "Application Support", "1Password"),
					filepath.Join(home, "Library", "Application Support", "Bitwarden"),
					filepath.Join(home, "Library", "Application Support", "LastPass"),
					filepath.Join(home, "Library", "Keychains"),
				}
			},
		},
		{
			id:          "vpn_proxy",
			title:       "VPN and Proxy Tools",
			description: "Protect network configuration used by VPN and proxy utilities.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "Library", "Application Support", "Proxyman"),
					filepath.Join(home, "Library", "Application Support", "Charles"),
					filepath.Join(home, "Library", "Application Support", "Surge"),
					filepath.Join(home, "Library", "Application Support", "Tailscale"),
					filepath.Join(home, ".config", "tailscale"),
					filepath.Join(home, ".config", "wireguard"),
				}
			},
		},
		{
			id:          "developer_identity",
			title:       "Developer Identity",
			description: "Protect SSH, GPG, cloud, and cluster identity material.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, ".ssh"),
					filepath.Join(home, ".gnupg"),
					filepath.Join(home, ".aws"),
					filepath.Join(home, ".kube"),
				}
			},
		},
		{
			id:          "mail_accounts",
			title:       "Mail and Account Data",
			description: "Protect local mail stores and account session data.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "Library", "Mail"),
					filepath.Join(home, "Library", "Containers", "com.apple.mail"),
					filepath.Join(home, "Library", "Group Containers", "com.microsoft.Outlook"),
				}
			},
		},
		{
			id:          "ai_workspaces",
			title:       "AI Workspaces",
			description: "Protect local AI app state, workspaces, and model stores.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "Library", "Application Support", "Claude"),
					filepath.Join(home, "Library", "Application Support", "ChatGPT"),
					filepath.Join(home, ".ollama"),
					filepath.Join(home, ".cache", "huggingface"),
				}
			},
		},
		{
			id:          "ide_settings",
			title:       "IDE Settings",
			description: "Protect editor and IDE configuration while allowing rebuildable caches elsewhere.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "Library", "Application Support", "Code", "User"),
					filepath.Join(home, "Library", "Application Support", "JetBrains"),
					filepath.Join(home, ".config", "zed"),
				}
			},
		},
		{
			id:          "launcher_state",
			title:       "Launchers and Automation",
			description: "Protect Raycast, Alfred, and launcher automation state while allowing safe cache cleanup elsewhere.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "Library", "Application Support", "Raycast"),
					filepath.Join(home, "Library", "Application Support", "Alfred"),
					filepath.Join(home, "Library", "Application Support", "com.runningwithcrayons.Alfred"),
					filepath.Join(home, "Library", "Containers", "com.raycast.macos.BrowserExtension"),
					filepath.Join(home, "Library", "Containers", "com.raycast.macos.RaycastAppIntents"),
					filepath.Join(home, "Library", "Mobile Documents"),
				}
			},
		},
		{
			id:          "safe_cache_domains",
			title:       "Safe Cache Domains",
			description: "Expose the current safe-cache exception surface used under protected roots.",
			roots: func(platform.Adapter) []string {
				return append([]string{}, roots.Browser...)
			},
		},
	}
}

func windowsFamilySpecs(adapter platform.Adapter) []protectedFamilySpec {
	home := homeFromRoots(adapter)
	return []protectedFamilySpec{
		{
			id:          "browser_profiles",
			title:       "Browser Profiles",
			description: "Protect browser profiles, history, and session state while still allowing safe cache exceptions.",
			roots: func(a platform.Adapter) []string {
				roots := []string{
					filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data"),
					filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "User Data"),
					filepath.Join(home, "AppData", "Local", "BraveSoftware", "Brave-Browser", "User Data"),
					filepath.Join(home, "AppData", "Roaming", "Mozilla", "Firefox", "Profiles"),
					filepath.Join(home, "AppData", "Local", "Mozilla", "Firefox", "Profiles"),
					filepath.Join(home, "AppData", "Local", "Packages"),
					filepath.Join(home, "AppData", "Local", "Microsoft", "Windows", "WebCache"),
				}
				return append(roots, a.CuratedRoots().Browser...)
			},
		},
		{
			id:          "password_managers",
			title:       "Password Managers",
			description: "Protect local credential vaults and roaming secrets.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "AppData", "Local", "1Password"),
					filepath.Join(home, "AppData", "Roaming", "1Password"),
					filepath.Join(home, "AppData", "Roaming", "Bitwarden"),
					filepath.Join(home, "AppData", "Roaming", "LastPass"),
					filepath.Join(home, "AppData", "Roaming", "Microsoft", "Credentials"),
					filepath.Join(home, "AppData", "Roaming", "Microsoft", "Crypto"),
				}
			},
		},
		{
			id:          "vpn_proxy",
			title:       "VPN and Proxy Tools",
			description: "Protect local VPN and proxy settings plus stateful network tools.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "AppData", "Local", "Tailscale"),
					filepath.Join(home, "AppData", "Roaming", "WireGuard"),
					filepath.Join(home, "AppData", "Local", "Proxyman"),
					filepath.Join(home, "AppData", "Roaming", "Charles"),
					filepath.Join(home, ".config", "tailscale"),
				}
			},
		},
		{
			id:          "developer_identity",
			title:       "Developer Identity",
			description: "Protect SSH, GPG, cloud, and cluster identity material.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, ".ssh"),
					filepath.Join(home, ".gnupg"),
					filepath.Join(home, ".aws"),
					filepath.Join(home, ".kube"),
				}
			},
		},
		{
			id:          "mail_accounts",
			title:       "Mail and Account Data",
			description: "Protect local mail store and account session data.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "AppData", "Local", "Microsoft", "Outlook"),
					filepath.Join(home, "AppData", "Roaming", "Thunderbird", "Profiles"),
					filepath.Join(home, "AppData", "Local", "Packages", "microsoft.windowscommunicationsapps"),
				}
			},
		},
		{
			id:          "ai_workspaces",
			title:       "AI Workspaces",
			description: "Protect AI app state, workspaces, and local model stores.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "AppData", "Roaming", "Claude"),
					filepath.Join(home, "AppData", "Roaming", "ChatGPT"),
					filepath.Join(home, ".ollama"),
					filepath.Join(home, ".cache", "huggingface"),
				}
			},
		},
		{
			id:          "ide_settings",
			title:       "IDE Settings",
			description: "Protect editor and IDE configuration while leaving caches reviewable.",
			roots: func(platform.Adapter) []string {
				return []string{
					filepath.Join(home, "AppData", "Roaming", "Code", "User"),
					filepath.Join(home, "AppData", "Roaming", "JetBrains"),
					filepath.Join(home, "AppData", "Local", "Programs", "Microsoft VS Code"),
				}
			},
		},
	}
}

func homeFromRoots(adapter platform.Adapter) string {
	targets := adapter.ResolveTargets([]string{"~"})
	if len(targets) > 0 && targets[0] != "~" {
		return targets[0]
	}
	for _, roots := range [][]string{
		adapter.CuratedRoots().Temp,
		adapter.CuratedRoots().Logs,
		adapter.CuratedRoots().Developer,
		adapter.CuratedRoots().Installer,
	} {
		for _, root := range roots {
			if root == "" {
				continue
			}
			cleaned := filepath.Clean(root)
			if idx := strings.Index(strings.ToLower(cleaned), strings.ToLower(string(filepath.Separator)+"library"+string(filepath.Separator))); idx > 0 {
				return cleaned[:idx]
			}
			if strings.Contains(strings.ToLower(cleaned), strings.ToLower(string(filepath.Separator)+"appdata"+string(filepath.Separator))) {
				parts := strings.Split(cleaned, string(filepath.Separator))
				for i := range parts {
					if strings.EqualFold(parts[i], "AppData") && i > 0 {
						return strings.Join(parts[:i], string(filepath.Separator))
					}
				}
			}
		}
	}
	return ""
}

func dedupeLower(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" || slices.Contains(out, normalized) {
			continue
		}
		out = append(out, normalized)
	}
	return out
}
