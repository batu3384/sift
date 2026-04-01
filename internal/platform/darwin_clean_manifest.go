//go:build darwin

package platform

import (
	"path/filepath"
	"strings"

	"github.com/batu3384/sift/internal/domain"
)

type darwinCleanFamily struct {
	ID        string
	Status    string
	Temp      []string
	Logs      []string
	Developer []string
}

func darwinCleanManifestRoots(home string) CuratedRoots {
	var roots CuratedRoots
	for _, family := range darwinCleanManifest {
		roots.Temp = append(roots.Temp, expandDarwinManifestPaths(home, family.Temp)...)
		roots.Logs = append(roots.Logs, expandDarwinManifestPaths(home, family.Logs)...)
		roots.Developer = append(roots.Developer, expandDarwinManifestPaths(home, family.Developer)...)
	}
	roots.Temp = dedupePaths(roots.Temp)
	roots.Logs = dedupePaths(roots.Logs)
	roots.Developer = dedupePaths(roots.Developer)
	return roots
}

func expandDarwinManifestPaths(home string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		joined := filepath.Join(home, filepath.FromSlash(value))
		if strings.ContainsAny(value, "*?[") {
			matches, _ := filepath.Glob(joined)
			for _, match := range matches {
				out = append(out, domain.NormalizePath(match))
			}
			continue
		}
		out = append(out, domain.NormalizePath(joined))
	}
	return out
}

func dedupePaths(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

var darwinCleanManifest = []darwinCleanFamily{
	{
		ID:     "apple_media",
		Status: "covered",
		Temp: []string{
			"Library/Application Support/AddressBook/Sources/*/Photos.cache",
		},
	},
	{
		ID:        "android",
		Status:    "covered",
		Developer: []string{".android/cache", ".android/build-cache"},
	},
	{
		ID:        "flutter",
		Status:    "covered",
		Developer: []string{".pub-cache", "flutter/bin/cache"},
	},
	{
		ID:        "expo",
		Status:    "covered",
		Developer: []string{".expo", ".expo-shared"},
	},
	{
		ID:        "deno",
		Status:    "covered",
		Developer: []string{".cache/deno", "Library/Caches/deno"},
	},
	{
		ID:        "bazel",
		Status:    "covered",
		Developer: []string{".cache/bazel", "Library/Caches/bazel"},
	},
	{
		ID:        "grafana",
		Status:    "covered",
		Developer: []string{".grafana/cache"},
	},
	{
		ID:        "prometheus",
		Status:    "covered",
		Developer: []string{".prometheus/data/wal"},
	},
	{
		ID:        "jenkins",
		Status:    "covered",
		Developer: []string{".jenkins/workspace"},
	},
	{
		ID:     "mongodb_compass",
		Status: "covered",
		Temp: []string{
			"Library/Caches/com.mongodb.compass",
			"Library/Application Support/MongoDB Compass/Cache",
			"Library/Application Support/MongoDB Compass/Code Cache",
			"Library/Application Support/MongoDB Compass/GPUCache",
		},
	},
	{
		ID:     "redis_insight",
		Status: "covered",
		Temp: []string{
			"Library/Caches/com.redis.redisinsight",
			"Library/Application Support/RedisInsight/Cache",
			"Library/Application Support/RedisInsight/Code Cache",
			"Library/Application Support/RedisInsight/GPUCache",
		},
	},
	{
		ID:     "navicat",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.prect.NavicatPremium"},
	},
	{
		ID:     "bear",
		Status: "covered",
		Temp:   []string{"Library/Caches/net.shinyfrog.bear"},
	},
	{
		ID:     "evernote",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.evernote.Evernote"},
	},
	{
		ID:     "logseq",
		Status: "covered",
		Temp: []string{
			"Library/Application Support/Logseq/Cache",
			"Library/Application Support/Logseq/Code Cache",
			"Library/Application Support/Logseq/GPUCache",
			"Library/Application Support/Logseq/Service Worker/CacheStorage",
		},
		Logs: []string{
			"Library/Application Support/Logseq/logs",
		},
	},
	{
		ID:     "cleanshot",
		Status: "covered",
		Temp:   []string{"Library/Caches/pl.maketheweb.cleanshotx"},
	},
	{
		ID:     "downie",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.charliemonroe.Downie-4"},
	},
	{
		ID:     "pacifist",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.charlessoft.pacifist"},
	},
	{
		ID:     "riot_client",
		Status: "covered",
		Temp: []string{
			"Library/Application Support/Riot Client/Cache",
			"Library/Application Support/Riot Client/Code Cache",
			"Library/Application Support/Riot Client/GPUCache",
		},
		Logs: []string{
			"Library/Application Support/Riot Client/logs",
		},
	},
	{
		ID:     "minecraft",
		Status: "covered",
		Temp: []string{
			"Library/Application Support/minecraft/webcache2",
		},
		Logs: []string{
			"Library/Application Support/minecraft/logs",
		},
	},
	{
		ID:     "lunar_client",
		Status: "covered",
		Temp: []string{
			"Library/Application Support/lunarclient/cache",
		},
		Logs: []string{
			"Library/Application Support/lunarclient/logs",
		},
	},
	{
		ID:     "feishu",
		Status: "covered",
		Temp: []string{
			"Library/Application Support/Lark/Cache",
			"Library/Application Support/Lark/Code Cache",
			"Library/Application Support/Lark/GPUCache",
		},
		Logs: []string{
			"Library/Application Support/Lark/logs",
		},
	},
	{
		ID:     "dingtalk",
		Status: "covered",
		Temp: []string{
			"Library/Application Support/DingTalk/Cache",
			"Library/Application Support/DingTalk/Code Cache",
			"Library/Application Support/DingTalk/GPUCache",
		},
		Logs: []string{
			"Library/Application Support/DingTalk/logs",
		},
	},
	{
		ID:     "anydesk",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.anydesk.anydesk"},
	},
	{
		ID:     "gog_galaxy",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.gog.galaxy"},
	},
	{
		ID:     "ea_app",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.ea.app"},
	},
	{
		ID:     "tencent_meeting",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.tencent.meeting"},
	},
	{
		ID:     "wecom",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.tencent.WeWorkMac"},
	},
	{
		ID:     "teamviewer",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.teamviewer.*"},
	},
	{
		ID:     "todesk",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.todesk.*"},
	},
	{
		ID:     "sunlogin",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.sunlogin.*"},
	},
	{
		ID:     "airmail",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.airmail.*"},
	},
	{
		ID:     "anydo",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.any.do.*"},
	},
	{
		ID:     "the_unarchiver",
		Status: "covered",
		Temp:   []string{"Library/Caches/cx.c3.theunarchiver"},
	},
	{
		ID:     "youdao",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.youdao.YoudaoDict"},
	},
	{
		ID:     "eudict",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.eudic.*"},
	},
	{
		ID:     "bob",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.bob-build.Bob"},
	},
	{
		ID:     "miaoyan",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.tw93.MiaoYan"},
	},
	{
		ID:     "flomo",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.flomoapp.mac"},
	},
	{
		ID:     "quark",
		Status: "covered",
		Temp:   []string{"Library/Application Support/Quark/Cache/videoCache"},
	},
	{
		ID:     "cinema4d",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.maxon.cinema4d"},
	},
	{
		ID:     "autodesk",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.autodesk.*"},
	},
	{
		ID:     "sketchup",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.sketchup.*"},
	},
	{
		ID:     "netease_music",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.netease.163music"},
	},
	{
		ID:     "qq_music",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.tencent.QQMusic"},
	},
	{
		ID:     "kugou_music",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.kugou.mac"},
	},
	{
		ID:     "kuwo_music",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.kuwo.mac"},
	},
	{
		ID:     "iqiyi",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.iqiyi.player"},
	},
	{
		ID:     "tencent_video",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.tencent.tenvideo"},
	},
	{
		ID:     "bilibili",
		Status: "covered",
		Temp:   []string{"Library/Caches/tv.danmaku.bili"},
	},
	{
		ID:     "douyu",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.douyu.*"},
	},
	{
		ID:     "huya",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.huya.*"},
	},
	{
		ID:     "camo",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.reincubate.camo"},
	},
	{
		ID:     "xnip",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.xnipapp.xnip"},
	},
	{
		ID:     "transmission",
		Status: "covered",
		Temp:   []string{"Library/Caches/org.m0k.transmission"},
	},
	{
		ID:     "qbittorrent",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.qbittorrent.qBittorrent"},
	},
	{
		ID:     "klee",
		Status: "covered",
		Temp: []string{
			"Library/Caches/com.klee.desktop",
			"Library/Caches/klee_desktop",
		},
	},
	{
		ID:     "ora_browser",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.orabrowser.app"},
	},
	{
		ID:     "filo",
		Status: "covered",
		Temp: []string{
			"Library/Caches/com.filo.client",
			"Library/Application Support/Filo/production/Cache",
			"Library/Application Support/Filo/production/Code Cache",
			"Library/Application Support/Filo/production/GPUCache",
			"Library/Application Support/Filo/production/DawnGraphiteCache",
			"Library/Application Support/Filo/production/DawnWebGPUCache",
		},
	},
	{
		ID:     "yinxiang_note",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.yinxiang.*"},
	},
	{
		ID:     "aria2",
		Status: "covered",
		Temp:   []string{"Library/Caches/net.xmac.aria2gui"},
	},
	{
		ID:     "folx",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.folx.*"},
	},
	{
		ID:        "wakatime",
		Status:    "covered",
		Temp:      []string{"Library/Caches/macos-wakatime.WakaTime"},
		Developer: []string{".wakatime", ".wakatime-internal"},
	},
	{
		ID:     "cacher",
		Status: "covered",
		Logs:   []string{".cacher/logs"},
	},
	{
		ID:     "kite",
		Status: "covered",
		Logs:   []string{".kite/logs"},
	},
	{
		ID:     "input_source_pro",
		Status: "covered",
		Temp:   []string{"Library/Caches/com.runjuu.Input-Source-Pro"},
	},
}
