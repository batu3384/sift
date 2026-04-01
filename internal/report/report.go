package report

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
	"github.com/batu3384/sift/internal/store"
)

func Bundle(ctx context.Context, st *store.Store, plan domain.ExecutionPlan, cfg config.Config) (string, string, error) {
	root, err := Dir()
	if err != nil {
		return "", "", err
	}
	return BundleAt(ctx, root, st, plan, cfg)
}

func Dir() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sift", "reports"), nil
}

func BundleAt(ctx context.Context, dir string, st *store.Store, plan domain.ExecutionPlan, cfg config.Config) (string, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	reportID := uuid.NewString()
	path := filepath.Join(dir, reportID+".zip")
	file, err := os.Create(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	sanitizedPlan := plan
	sanitizedConfig := cfg
	diagnostics := platform.Current().Diagnostics(ctx)
	recent := []store.RecentScan(nil)
	execution := (*store.ExecutionSummary)(nil)
	statusSummary := (*store.StatusSummary)(nil)
	auditRecords := []store.AuditRecord(nil)
	if st != nil {
		recent, _ = st.RecentScans(ctx, 10)
		execution, _ = st.LatestExecution(ctx)
		summary, err := st.BuildStatusSummary(ctx, plan.Platform, 10)
		if err == nil {
			statusSummary = &summary
		}
		auditRecords, _ = st.RecentAuditRecords(time.Now().UTC(), 50)
	}
	if cfg.Diagnostics.Redaction {
		sanitizedPlan = redactPlan(plan)
		sanitizedConfig = redactConfig(cfg)
		diagnostics = redactDiagnostics(diagnostics)
		recent = redactRecentScans(recent)
		execution = redactExecutionSummary(execution)
		statusSummary = redactStatusSummary(statusSummary)
		auditRecords = redactAuditRecords(auditRecords)
	}
	if err := writeJSON(writer, "plan.json", sanitizedPlan); err != nil {
		return "", "", err
	}
	if err := writeJSON(writer, "config.json", sanitizedConfig); err != nil {
		return "", "", err
	}
	if err := writeJSON(writer, "diagnostics.json", diagnostics); err != nil {
		return "", "", err
	}
	if err := writeJSON(writer, "recent_scans.json", recent); err != nil {
		return "", "", err
	}
	if err := writeJSON(writer, "latest_execution.json", execution); err != nil {
		return "", "", err
	}
	if err := writeJSON(writer, "status_summary.json", statusSummary); err != nil {
		return "", "", err
	}
	if err := writeJSON(writer, "audit.json", auditRecords); err != nil {
		return "", "", err
	}
	meta := map[string]string{
		"report_id":  reportID,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeJSON(writer, "meta.json", meta); err != nil {
		return "", "", err
	}
	if err := writer.Close(); err != nil {
		return "", "", err
	}
	if st != nil {
		_ = st.SaveReport(ctx, reportID, plan.ScanID, path)
	}
	return reportID, path, nil
}

func redactPlan(plan domain.ExecutionPlan) domain.ExecutionPlan {
	plan.Targets = redactList(plan.Targets)
	plan.Warnings = redactList(plan.Warnings)
	plan.Policy.ProtectedPaths = redactList(plan.Policy.ProtectedPaths)
	plan.Policy.UserProtectedPaths = redactList(plan.Policy.UserProtectedPaths)
	plan.Policy.SystemProtectedPaths = redactList(plan.Policy.SystemProtectedPaths)
	plan.Policy.ProtectedPathExceptions = redactList(plan.Policy.ProtectedPathExceptions)
	plan.Policy.AllowedRoots = redactList(plan.Policy.AllowedRoots)
	for i := range plan.Items {
		plan.Items[i].Path = domain.RedactPath(plan.Items[i].Path)
		plan.Items[i].DisplayPath = domain.RedactPath(plan.Items[i].DisplayPath)
		plan.Items[i].Recovery.Location = domain.RedactPath(plan.Items[i].Recovery.Location)
		plan.Items[i].NativeCommand = redactString(plan.Items[i].NativeCommand)
	}
	return plan
}

func redactConfig(cfg config.Config) config.Config {
	cfg.ProtectedPaths = redactList(cfg.ProtectedPaths)
	cfg.PurgeSearchPaths = redactList(cfg.PurgeSearchPaths)
	return cfg
}

func redactDiagnostics(diagnostics []platform.Diagnostic) []platform.Diagnostic {
	out := make([]platform.Diagnostic, len(diagnostics))
	for i, diagnostic := range diagnostics {
		diagnostic.Message = domain.RedactPath(diagnostic.Message)
		out[i] = diagnostic
	}
	return out
}

func redactAuditRecords(records []store.AuditRecord) []store.AuditRecord {
	out := make([]store.AuditRecord, len(records))
	for i, record := range records {
		record.Payload = redactRawJSON(record.Payload)
		out[i] = record
	}
	return out
}

func redactStatusSummary(summary *store.StatusSummary) *store.StatusSummary {
	if summary == nil {
		return nil
	}
	copy := *summary
	copy.AuditLogPath = domain.RedactPath(copy.AuditLogPath)
	copy.RecentScans = redactRecentScans(copy.RecentScans)
	if copy.LastScan != nil {
		last := redactRecentScan(*copy.LastScan)
		copy.LastScan = &last
	}
	if copy.PreviousScan != nil {
		prev := redactRecentScan(*copy.PreviousScan)
		copy.PreviousScan = &prev
	}
	copy.LastExecution = redactExecutionSummary(copy.LastExecution)
	return &copy
}

func redactExecutionSummary(summary *store.ExecutionSummary) *store.ExecutionSummary {
	if summary == nil {
		return nil
	}
	copy := *summary
	copy.Warnings = redactList(copy.Warnings)
	copy.FollowUpCommands = redactList(copy.FollowUpCommands)
	return &copy
}

func redactRecentScans(scans []store.RecentScan) []store.RecentScan {
	out := make([]store.RecentScan, len(scans))
	for i, scan := range scans {
		out[i] = redactRecentScan(scan)
	}
	return out
}

func redactRecentScan(scan store.RecentScan) store.RecentScan {
	scan.Warnings = redactList(scan.Warnings)
	return scan
}

func redactList(values []string) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = redactString(value)
	}
	return out
}

func redactRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	value = redactAny(value)
	encoded, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return encoded
}

func redactAny(value any) any {
	switch typed := value.(type) {
	case string:
		return redactString(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactAny(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = redactAny(item)
		}
		return out
	default:
		return value
	}
}

func redactString(value string) string {
	value = strings.ReplaceAll(value, `\`, `/`)
	redacted := domain.RedactPath(value)
	home, err := os.UserHomeDir()
	if err != nil {
		return redacted
	}
	normalizedHome := strings.ReplaceAll(filepath.Clean(home), `\`, `/`)
	if normalizedHome == "" {
		return redacted
	}
	return strings.ReplaceAll(redacted, normalizedHome, "~")
}

func writeJSON(writer *zip.Writer, name string, value any) error {
	entry, err := writer.Create(name)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(entry)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
