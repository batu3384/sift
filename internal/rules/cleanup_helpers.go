package rules

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
)

func allCuratedRoots(adapter platform.Adapter) []string {
	roots := adapter.CuratedRoots()
	all := append([]string{}, roots.Temp...)
	all = append(all, roots.Logs...)
	all = append(all, roots.Developer...)
	all = append(all, roots.Browser...)
	all = append(all, roots.Installer...)
	all = append(all, roots.PackageManager...)
	all = append(all, roots.AppSupport...)
	all = append(all, roots.RecentItems...)
	all = append(all, roots.SystemUpdate...)
	return dedupeStrings(all)
}

func darwinEdgeRunning() bool {
	return currentAdapterIsProcessRunning("Microsoft Edge")
}

func compareVersionish(left string, right string) int {
	leftParts := strings.FieldsFunc(left, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	rightParts := strings.FieldsFunc(right, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	maxParts := len(leftParts)
	if len(rightParts) > maxParts {
		maxParts = len(rightParts)
	}
	for i := 0; i < maxParts; i++ {
		lv := partValue(leftParts, i)
		rv := partValue(rightParts, i)
		if lv != rv {
			if lv < rv {
				return -1
			}
			return 1
		}
	}
	return strings.Compare(left, right)
}

func dedupeFindings(in []domain.Finding) []domain.Finding {
	seen := map[string]struct{}{}
	out := make([]domain.Finding, 0, len(in))
	for _, finding := range in {
		key := finding.RuleID + "|" + finding.Path + "|" + finding.Source
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, finding)
	}
	return out
}

func partValue(parts []string, index int) int {
	if index >= len(parts) {
		return 0
	}
	value, err := strconv.Atoi(parts[index])
	if err != nil {
		return 0
	}
	return value
}

func shouldSkipCuratedOverlap(path string, root string, curated []string) bool {
	for _, candidate := range curated {
		normalized := domain.NormalizePath(candidate)
		if normalized == "" || normalized == root || normalized == path {
			continue
		}
		if domain.HasPathPrefix(normalized, path) {
			return true
		}
	}
	return false
}

func isSIFTOwnedPath(path string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(domain.NormalizePath(path), `\`, `/`))
	switch {
	case strings.Contains(normalized, "/library/application support/sift"):
		return true
	case strings.Contains(normalized, "/library/caches/sift"):
		return true
	case strings.Contains(normalized, "/library/logs/sift"):
		return true
	case strings.Contains(normalized, "/appdata/roaming/sift"):
		return true
	case strings.Contains(normalized, "/appdata/local/sift"):
		return true
	case strings.HasSuffix(normalized, "/.cache/sift"):
		return true
	default:
		return false
	}
}

func shouldTreatRootAsLeaf(root string, category domain.Category) bool {
	if root == "" {
		return false
	}
	normalized := strings.ToLower(strings.ReplaceAll(domain.NormalizePath(root), `\`, `/`))
	if normalized == "" || isBroadCleanupRoot(normalized) {
		return false
	}
	base := strings.ToLower(filepath.Base(normalized))
	if strings.Contains(normalized, "/library/caches/") && strings.Contains(base, ".") {
		return true
	}
	if _, ok := leafCleanupBasenames[base]; ok {
		switch base {
		case "cache", "logs":
			return strings.Contains(normalized, "/application support/") ||
				strings.Contains(normalized, "/user data/") ||
				strings.Contains(normalized, "/appdata/") ||
				strings.Contains(normalized, "/packages/") ||
				strings.Contains(normalized, "/steam/") ||
				strings.Contains(normalized, "/.android/") ||
				strings.Contains(normalized, "/flutter/bin/")
		default:
			return true
		}
	}
	switch {
	case strings.Contains(normalized, "/library/developer/xcode/deriveddata"),
		strings.Contains(normalized, "/library/developer/xcode/archives"),
		strings.Contains(normalized, "/library/developer/xcode/ios devicesupport"),
		strings.Contains(normalized, "/library/developer/xcode/watchos devicesupport"),
		strings.Contains(normalized, "/library/developer/xcode/tvos devicesupport"),
		strings.Contains(normalized, "/library/developer/coresimulator/caches"),
		strings.Contains(normalized, "/library/developer/xcode/documentationcache"),
		strings.Contains(normalized, "/library/developer/xcode/documentationindex"),
		strings.Contains(normalized, "/docker/buildx/cache"),
		strings.Contains(normalized, "/ms-playwright"),
		strings.Contains(normalized, "/puppeteer"),
		strings.Contains(normalized, "/.android/cache"),
		strings.Contains(normalized, "/.android/build-cache"),
		strings.Contains(normalized, "/.pub-cache"),
		strings.Contains(normalized, "/flutter/bin/cache"),
		strings.Contains(normalized, "/.expo"),
		strings.Contains(normalized, "/.expo-shared"),
		strings.Contains(normalized, "/.cache/deno"),
		strings.Contains(normalized, "/library/caches/deno"),
		strings.Contains(normalized, "/.cache/bazel"),
		strings.Contains(normalized, "/library/caches/bazel"),
		strings.Contains(normalized, "/.grafana/cache"),
		strings.Contains(normalized, "/.prometheus/data/wal"),
		(strings.Contains(normalized, "/.jenkins/workspace/") && (strings.HasSuffix(normalized, "/target") || strings.HasSuffix(normalized, "/build"))),
		strings.Contains(normalized, "/library/caches/com.mongodb.compass"),
		strings.Contains(normalized, "/application support/mongodb compass/cache"),
		strings.Contains(normalized, "/application support/mongodb compass/code cache"),
		strings.Contains(normalized, "/application support/mongodb compass/gpucache"),
		strings.Contains(normalized, "/library/caches/com.redis.redisinsight"),
		strings.Contains(normalized, "/application support/redisinsight/cache"),
		strings.Contains(normalized, "/application support/redisinsight/code cache"),
		strings.Contains(normalized, "/application support/redisinsight/gpucache"),
		strings.Contains(normalized, "/library/caches/com.prect.navicatpremium"),
		strings.Contains(normalized, "/library/caches/net.shinyfrog.bear"),
		strings.Contains(normalized, "/library/caches/com.evernote.evernote"),
		strings.Contains(normalized, "/application support/logseq/cache"),
		strings.Contains(normalized, "/application support/logseq/code cache"),
		strings.Contains(normalized, "/application support/logseq/gpucache"),
		strings.Contains(normalized, "/application support/logseq/service worker/cachestorage"),
		strings.Contains(normalized, "/application support/logseq/logs"),
		strings.Contains(normalized, "/library/caches/pl.maketheweb.cleanshotx"),
		strings.Contains(normalized, "/library/caches/com.charliemonroe.downie-4"),
		strings.Contains(normalized, "/library/caches/com.charlessoft.pacifist"),
		strings.Contains(normalized, "/application support/riot client/cache"),
		strings.Contains(normalized, "/application support/riot client/code cache"),
		strings.Contains(normalized, "/application support/riot client/gpucache"),
		strings.Contains(normalized, "/application support/riot client/logs"),
		strings.Contains(normalized, "/application support/minecraft/webcache2"),
		strings.Contains(normalized, "/application support/minecraft/logs"),
		strings.Contains(normalized, "/application support/lunarclient/cache"),
		strings.Contains(normalized, "/application support/lunarclient/logs"),
		strings.Contains(normalized, "/application support/lark/cache"),
		strings.Contains(normalized, "/application support/lark/code cache"),
		strings.Contains(normalized, "/application support/lark/gpucache"),
		strings.Contains(normalized, "/application support/lark/logs"),
		strings.Contains(normalized, "/application support/dingtalk/cache"),
		strings.Contains(normalized, "/application support/dingtalk/code cache"),
		strings.Contains(normalized, "/application support/dingtalk/gpucache"),
		strings.Contains(normalized, "/application support/dingtalk/logs"),
		strings.Contains(normalized, "/library/caches/com.anydesk.anydesk"),
		strings.Contains(normalized, "/library/caches/com.gog.galaxy"),
		strings.Contains(normalized, "/library/caches/com.ea.app"),
		strings.Contains(normalized, "/library/caches/com.klee.desktop"),
		strings.Contains(normalized, "/library/caches/klee_desktop"),
		strings.Contains(normalized, "/library/caches/com.orabrowser.app"),
		strings.Contains(normalized, "/library/caches/com.filo.client"),
		strings.Contains(normalized, "/application support/filo/production/cache"),
		strings.Contains(normalized, "/application support/filo/production/code cache"),
		strings.Contains(normalized, "/application support/filo/production/gpucache"),
		strings.Contains(normalized, "/application support/filo/production/dawngraphitecache"),
		strings.Contains(normalized, "/application support/filo/production/dawnwebgpucache"),
		strings.Contains(normalized, "/library/caches/net.xmac.aria2gui"),
		strings.Contains(normalized, "/library/caches/com.folx."),
		strings.Contains(normalized, "/library/caches/com.yinxiang."),
		strings.Contains(normalized, "/.cacher/logs"),
		strings.Contains(normalized, "/.kite/logs"),
		strings.Contains(normalized, "/library/caches/com.runjuu.input-source-pro"),
		strings.Contains(normalized, "/library/caches/macos-wakatime.wakatime"),
		strings.Contains(normalized, "/library/caches/com.tencent.meeting"),
		strings.Contains(normalized, "/library/caches/com.tencent.weworkmac"),
		strings.Contains(normalized, "/library/caches/com.teamviewer."),
		strings.Contains(normalized, "/library/caches/com.todesk."),
		strings.Contains(normalized, "/library/caches/com.sunlogin."),
		strings.Contains(normalized, "/library/caches/com.airmail."),
		strings.Contains(normalized, "/library/caches/com.any.do."),
		strings.Contains(normalized, "/library/caches/cx.c3.theunarchiver"),
		strings.Contains(normalized, "/library/caches/com.youdao.youdaodict"),
		strings.Contains(normalized, "/library/caches/com.eudic."),
		strings.Contains(normalized, "/library/caches/com.bob-build.bob"),
		strings.Contains(normalized, "/library/caches/com.tw93.miaoyan"),
		strings.Contains(normalized, "/library/caches/com.flomoapp.mac"),
		strings.Contains(normalized, "/application support/quark/cache/videocache"),
		strings.Contains(normalized, "/library/caches/com.maxon.cinema4d"),
		strings.Contains(normalized, "/library/caches/com.autodesk."),
		strings.Contains(normalized, "/library/caches/com.sketchup."),
		strings.Contains(normalized, "/library/caches/com.netease.163music"),
		strings.Contains(normalized, "/library/caches/com.tencent.qqmusic"),
		strings.Contains(normalized, "/library/caches/com.kugou.mac"),
		strings.Contains(normalized, "/library/caches/com.kuwo.mac"),
		strings.Contains(normalized, "/library/caches/com.iqiyi.player"),
		strings.Contains(normalized, "/library/caches/com.tencent.tenvideo"),
		strings.Contains(normalized, "/library/caches/tv.danmaku.bili"),
		strings.Contains(normalized, "/library/caches/com.douyu."),
		strings.Contains(normalized, "/library/caches/com.huya."),
		strings.Contains(normalized, "/library/caches/com.reincubate.camo"),
		strings.Contains(normalized, "/library/caches/com.xnipapp.xnip"),
		strings.Contains(normalized, "/library/caches/org.m0k.transmission"),
		strings.Contains(normalized, "/library/caches/com.qbittorrent.qbittorrent"),
		strings.Contains(normalized, "/library/caches/notion.id"),
		strings.Contains(normalized, "/library/caches/md.obsidian"),
		strings.Contains(normalized, "/library/caches/com.runningwithcrayons.alfred"),
		strings.Contains(normalized, "/library/caches/com.microsoft.teams2"),
		strings.Contains(normalized, "/library/caches/us.zoom.xos"),
		strings.Contains(normalized, "/library/caches/ru.keepcoder.telegram"),
		strings.Contains(normalized, "/library/caches/com.tencent.xinwechat"),
		strings.Contains(normalized, "/library/caches/com.skype.skype"),
		strings.Contains(normalized, "/library/caches/net.whatsapp.whatsapp"),
		strings.Contains(normalized, "/library/caches/com.todoist.mac.todoist"),
		strings.Contains(normalized, "/library/caches/com.valvesoftware.steam"),
		strings.Contains(normalized, "/library/caches/com.epicgames.epicgameslauncher"),
		strings.Contains(normalized, "/library/caches/com.blizzard.battle.net"),
		strings.Contains(normalized, "/library/caches/com.colliderli.iina"),
		strings.Contains(normalized, "/library/caches/org.videolan.vlc"),
		strings.Contains(normalized, "/library/caches/io.mpv"),
		strings.Contains(normalized, "/library/caches/tv.plex.player.desktop"),
		strings.Contains(normalized, "/library/caches/com.readdle.smartemail-mac"),
		strings.Contains(normalized, "/library/apple/usr/share/rosetta/rosetta_update_bundle"),
		strings.Contains(normalized, "/mail downloads"),
		// Cloud storage caches
		strings.Contains(normalized, "/library/caches/com.getdropbox.dropbox"),
		strings.Contains(normalized, "/library/caches/com.dropbox.client2"),
		strings.Contains(normalized, "/library/caches/com.google.googledrive"),
		strings.Contains(normalized, "/library/caches/com.microsoft.onedrive"),
		strings.Contains(normalized, "/library/caches/com.baidu.netdisk"),
		strings.Contains(normalized, "/library/caches/com.box.desktop"),
		// Virtualization caches
		strings.Contains(normalized, "/library/caches/com.vmware.fusion"),
		strings.Contains(normalized, "/library/caches/com.parallels.desktop.launch"),
		strings.HasSuffix(normalized, "/virtualbox vms/.cache"),
		strings.HasSuffix(normalized, "/.vagrant.d/tmp"):
		return true
	}
	return category == domain.CategoryLogs && strings.Contains(normalized, "/logs/")
}

func isCoreSimulatorDevicesRoot(root string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(domain.NormalizePath(root), `\`, `/`))
	return strings.HasSuffix(normalized, "/library/developer/coresimulator/devices")
}

func isBroadCleanupRoot(normalized string) bool {
	broad := []string{
		"/library/caches",
		"/library/logs",
		"/library/application support",
		"/library/containers",
		"/library/preferences",
		"/library/saved application state",
		"/library/webkit",
		"/library/httpstorages",
		"/profiles",
		"/packages",
		"/repository",
		"/registry",
		"/git",
		"/downloads",
		"/desktop",
		"/temp",
	}
	for _, suffix := range broad {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}
	return false
}

func cleanupSourceLabel(root string, fallback string) string {
	normalized := strings.ToLower(strings.ReplaceAll(domain.NormalizePath(root), `\`, `/`))
	bestLabel := ""
	bestNeedleLen := -1
	for _, matcher := range cleanupSourceMatchers {
		if strings.Contains(normalized, matcher.needle) && len(matcher.needle) > bestNeedleLen {
			bestLabel = matcher.label
			bestNeedleLen = len(matcher.needle)
		}
	}
	if bestLabel != "" {
		return bestLabel
	}
	return fallback
}

var currentAdapterIsProcessRunning = func(names ...string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	return platform.Current().IsProcessRunning(names...)
}
