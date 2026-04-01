package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/batuhanyuksel/sift/internal/config"
	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/platform"
	"github.com/batuhanyuksel/sift/internal/rules"
)

func (s *Service) buildPolicy(opts ScanOptions, defs []rules.Definition, extraAllowed []string) domain.ProtectionPolicy {
	systemProtected := append([]string{}, s.Adapter.ProtectedPaths()...)
	userProtected := append([]string{}, s.Config.ProtectedPaths...)
	commandProtected := normalizePolicyPaths(s.Config.CommandExcludes[config.NormalizeCommandName(opts.Command)])
	familyRoots, families := familyProtectedRoots(s.Adapter, s.Config.ProtectedFamilies)
	protected := append([]string{}, systemProtected...)
	protected = append(protected, userProtected...)
	protected = append(protected, commandProtected...)
	protected = append(protected, familyRoots...)

	// Add whitelist paths to protected (Mole-style whitelist)
	whitelistPaths := normalizePolicyPaths(s.Config.Whitelist)
	protected = append(protected, whitelistPaths...)

	allowed := append([]string{}, s.resolveTargetsForPlan(opts)...)
	for _, definition := range defs {
		if definition.Roots == nil {
			continue
		}
		allowed = append(allowed, definition.Roots(s.Adapter, opts.Targets)...)
	}
	allowed = append(allowed, extraAllowed...)

	return domain.ProtectionPolicy{
		ProtectedPaths:          normalizePolicyPaths(protected),
		Command:                 config.NormalizeCommandName(opts.Command),
		CommandProtectedPaths:   commandProtected,
		UserProtectedPaths:      normalizePolicyPaths(userProtected),
		SystemProtectedPaths:    normalizePolicyPaths(systemProtected),
		FamilyProtectedPaths:    normalizePolicyPaths(familyRoots),
		ProtectedFamilies:       families,
		ProtectedPathExceptions: normalizePolicyPaths(safeProtectedExceptions(s.Adapter.CuratedRoots())),
		AllowedRoots:            normalizePolicyPaths(allowed),
		TrashOnly:               s.Config.TrashMode != "permanent",
		AllowAdmin:              opts.AllowAdmin,
		BlockSymlinks:           true,
		WhitelistPaths:          whitelistPaths,
	}
}

func safeProtectedExceptions(roots platform.CuratedRoots) []string {
	exceptions := make([]string, 0, len(roots.Temp)+len(roots.Logs)+len(roots.Developer)+len(roots.Browser)+len(roots.Installer)+len(roots.PackageManager))
	exceptions = append(exceptions, roots.Temp...)
	exceptions = append(exceptions, roots.Logs...)
	exceptions = append(exceptions, roots.Developer...)
	exceptions = append(exceptions, roots.Browser...)
	exceptions = append(exceptions, roots.Installer...)
	exceptions = append(exceptions, roots.PackageManager...)
	return exceptions
}

func (s *Service) ExplainProtection(path string) domain.ProtectionExplanation {
	return s.ExplainProtectionForCommand(path, "")
}

func (s *Service) ExplainProtectionForCommand(path, command string) domain.ProtectionExplanation {
	normalized := domain.NormalizePath(path)
	command = config.NormalizeCommandName(command)
	userProtected := normalizePolicyPaths(s.Config.ProtectedPaths)
	commandProtected := normalizePolicyPaths(s.Config.CommandExcludes[command])
	systemProtected := normalizePolicyPaths(s.Adapter.ProtectedPaths())
	exceptions := normalizePolicyPaths(safeProtectedExceptions(s.Adapter.CuratedRoots()))
	families := matchingFamilyIDs(s.Adapter, s.Config.ProtectedFamilies, normalized)

	explanation := domain.ProtectionExplanation{
		Path:             normalized,
		Command:          command,
		CommandMatches:   matchingRoots(normalized, commandProtected),
		UserMatches:      matchingRoots(normalized, userProtected),
		SystemMatches:    matchingRoots(normalized, systemProtected),
		FamilyMatches:    families,
		ExceptionMatches: matchingRoots(normalized, exceptions),
		State:            domain.ProtectionStateUnprotected,
		Message:          "No user or built-in protection rule matches this path.",
	}
	switch {
	case len(explanation.CommandMatches) > 0:
		explanation.State = domain.ProtectionStateCommandProtected
		if command != "" {
			explanation.Message = "Blocked by a command-scoped exclusion for " + command + "."
		} else {
			explanation.Message = "Blocked by a command-scoped exclusion."
		}
	case len(explanation.UserMatches) > 0:
		explanation.State = domain.ProtectionStateUserProtected
		explanation.Message = "Blocked by a user-configured protected path."
	case len(explanation.ExceptionMatches) > 0 && (len(explanation.SystemMatches) > 0 || len(explanation.FamilyMatches) > 0):
		explanation.State = domain.ProtectionStateSafeException
		if len(explanation.FamilyMatches) > 0 {
			explanation.Message = "Allowed as a curated safe cache path under an active protected family."
		} else {
			explanation.Message = "Allowed as a curated safe cache path under a built-in protected root."
		}
	case len(explanation.FamilyMatches) > 0:
		explanation.State = domain.ProtectionStateUserProtected
		explanation.Message = "Blocked by an active protected family."
	case len(explanation.SystemMatches) > 0:
		explanation.State = domain.ProtectionStateSystemProtected
		explanation.Message = "Blocked by a built-in protected path."
	}
	return explanation
}

func normalizePolicyPaths(paths []string) []string {
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
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func applyPolicy(item domain.Finding, decision domain.PolicyDecision) domain.Finding {
	item.Policy = decision
	if !decision.Allowed && item.Action != domain.ActionAdvisory {
		item.Status = domain.StatusProtected
		item.Recovery = domain.RecoveryHint{
			Message:  decision.Message,
			Location: "policy guard",
		}
	}
	return item
}

func evaluatePolicy(policy domain.ProtectionPolicy, item domain.Finding, permanent bool) domain.PolicyDecision {
	if item.Action == domain.ActionNative {
		if strings.TrimSpace(item.NativeCommand) == "" {
			return deny(domain.ProtectionUnsafeCommand, "Native uninstall command is missing.")
		}
		if _, err := parseNativeCommand(item.NativeCommand); err != nil {
			return deny(domain.ProtectionUnsafeCommand, err.Error())
		}
		if item.RequiresAdmin && !policy.AllowAdmin {
			return deny(domain.ProtectionAdminRequired, "Requires --admin on this platform.")
		}
		return domain.PolicyDecision{
			Allowed: true,
		}
	}
	if item.Action == domain.ActionCommand {
		if err := validateManagedCommand(item.CommandPath, item.CommandArgs); err != nil {
			return deny(domain.ProtectionUnsafeCommand, err.Error())
		}
		if item.RequiresAdmin && !policy.AllowAdmin {
			return deny(domain.ProtectionAdminRequired, "Requires --admin on this platform.")
		}
		return domain.PolicyDecision{
			Allowed: true,
		}
	}
	destructive := item.Action != domain.ActionAdvisory
	if !destructive {
		return domain.PolicyDecision{
			Allowed: true,
			Reason:  domain.ProtectionAdvisoryOnly,
			Message: "Read-only finding.",
		}
	}

	if item.Path == "" {
		return deny(domain.ProtectionEmptyPath, "Empty paths cannot be deleted.")
	}
	if !filepath.IsAbs(item.Path) {
		return deny(domain.ProtectionRelativePath, "Relative paths are not allowed for destructive actions.")
	}
	if domain.HasControlChars(item.Path) {
		return deny(domain.ProtectionControlChars, "Paths with control characters are blocked.")
	}
	if domain.ContainsTraversal(item.Path) {
		return deny(domain.ProtectionTraversal, "Traversal-style paths are blocked.")
	}
	if domain.IsRootPath(item.Path) {
		return deny(domain.ProtectionCriticalRoot, "Critical root paths cannot be deleted.")
	}

	info, err := os.Lstat(item.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return deny(domain.ProtectionMissingPath, "Path is no longer present.")
		}
		return deny(domain.ProtectionMissingPath, err.Error())
	}
	if policy.BlockSymlinks && info.Mode()&os.ModeSymlink != 0 {
		return deny(domain.ProtectionSymlink, "Symlink targets are blocked from destructive actions.")
	}
	if item.RequiresAdmin && !policy.AllowAdmin {
		return deny(domain.ProtectionAdminRequired, "Requires --admin on this platform.")
	}
	for _, prefix := range policy.UserProtectedPaths {
		if domain.HasPathPrefix(item.Path, prefix) {
			return deny(domain.ProtectionProtectedPath, "Protected by policy.")
		}
	}
	for _, prefix := range policy.CommandProtectedPaths {
		if domain.HasPathPrefix(item.Path, prefix) {
			message := "Excluded by command policy."
			if policy.Command != "" {
				message = "Excluded from " + policy.Command + " by command policy."
			}
			return deny(domain.ProtectionCommandExcluded, message)
		}
	}
	for _, prefix := range policy.SystemProtectedPaths {
		if !domain.HasPathPrefix(item.Path, prefix) {
			continue
		}
		if isUnderAnyRoot(item.Path, policy.ProtectedPathExceptions) {
			continue
		}
		// Allow if path is explicitly in AllowedRoots (e.g., uninstalling a specific app)
		if isUnderAnyRoot(item.Path, policy.AllowedRoots) {
			continue
		}
		return deny(domain.ProtectionProtectedPath, "Protected by built-in policy.")
	}
	for _, prefix := range policy.FamilyProtectedPaths {
		if !domain.HasPathPrefix(item.Path, prefix) {
			continue
		}
		if isUnderAnyRoot(item.Path, policy.ProtectedPathExceptions) {
			continue
		}
		return deny(domain.ProtectionProtectedPath, "Protected by family policy.")
	}
	if len(policy.AllowedRoots) > 0 && !isUnderAnyRoot(item.Path, policy.AllowedRoots) {
		return deny(domain.ProtectionOutsideAllowedRoots, "Outside allowed cleanup roots.")
	}

	return domain.PolicyDecision{
		Allowed:       true,
		RequiresTrash: policy.TrashOnly && !permanent,
	}
}

func deny(reason domain.ProtectionReason, message string) domain.PolicyDecision {
	return domain.PolicyDecision{
		Allowed: false,
		Reason:  reason,
		Message: message,
	}
}

func isUnderAnyRoot(path string, roots []string) bool {
	for _, root := range roots {
		if domain.HasPathPrefix(path, root) {
			return true
		}
	}
	return false
}

func matchingRoots(path string, roots []string) []string {
	matches := make([]string, 0, len(roots))
	for _, root := range roots {
		if domain.HasPathPrefix(path, root) {
			matches = append(matches, root)
		}
	}
	return matches
}
