package platform

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/batuhanyuksel/sift/internal/domain"
)

func aliasCandidates(app domain.AppEntry) []string {
	values := []string{
		app.DisplayName,
		app.Name,
		strings.TrimSuffix(filepath.Base(app.BundlePath), filepath.Ext(app.BundlePath)),
	}
	for _, support := range app.SupportPaths {
		values = append(values, strings.TrimSuffix(filepath.Base(support), filepath.Ext(support)))
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
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

func normalizedNameKey(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func exactNameMatches(root string, aliases []string) ([]string, []string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{domain.NormalizePath(root) + ": " + err.Error()}
	}
	keys := map[string]struct{}{}
	for _, alias := range aliases {
		key := normalizedNameKey(alias)
		if key != "" {
			keys[key] = struct{}{}
		}
	}
	var matches []string
	for _, entry := range entries {
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if _, ok := keys[normalizedNameKey(base)]; !ok {
			continue
		}
		matches = append(matches, domain.NormalizePath(filepath.Join(root, entry.Name())))
	}
	return matches, nil
}

func prefixMatches(root, prefix string) ([]string, []string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{domain.NormalizePath(root) + ": " + err.Error()}
	}
	var matches []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			matches = append(matches, domain.NormalizePath(filepath.Join(root, entry.Name())))
		}
	}
	return matches, nil
}

func existingUniquePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		normalized := domain.NormalizePath(path)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		if _, err := os.Lstat(normalized); err != nil {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
