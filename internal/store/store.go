package store

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/batu3384/sift/internal/domain"
)

type Store struct {
	db   *sql.DB
	path string
}

type RecentScan struct {
	ScanID    string        `json:"scan_id"`
	Command   string        `json:"command"`
	Profile   string        `json:"profile"`
	Platform  string        `json:"platform"`
	CreatedAt time.Time     `json:"created_at"`
	Totals    domain.Totals `json:"totals"`
	Warnings  []string      `json:"warnings"`
}

type ExecutionSummary struct {
	ID               string    `json:"id"`
	ScanID           string    `json:"scan_id"`
	StartedAt        time.Time `json:"started_at"`
	FinishedAt       time.Time `json:"finished_at"`
	ItemCount        int       `json:"item_count"`
	Completed        int       `json:"completed"`
	Deleted          int       `json:"deleted"`
	Failed           int       `json:"failed"`
	Protected        int       `json:"protected"`
	Skipped          int       `json:"skipped"`
	FreedBytes       int64     `json:"freed_bytes,omitempty"`
	Warnings         []string  `json:"warnings,omitempty"`
	FollowUpCommands []string  `json:"follow_up_commands,omitempty"`
}

type StatusSummary struct {
	Platform      string            `json:"platform"`
	RecentScans   []RecentScan      `json:"recent_scans"`
	LastScan      *RecentScan       `json:"last_scan,omitempty"`
	PreviousScan  *RecentScan       `json:"previous_scan,omitempty"`
	DeltaBytes    int64             `json:"delta_bytes"`
	DeltaItems    int               `json:"delta_items"`
	LastExecution *ExecutionSummary `json:"last_execution,omitempty"`
	AuditLogPath  string            `json:"audit_log_path"`
}

type AuditRecord struct {
	Kind      string          `json:"kind"`
	ScanID    string          `json:"scan_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func Path() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sift", "state.db"), nil
}

func Open() (*Store, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return OpenAt(path)
}

func OpenAt(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db, path: path}
	return store, store.migrate()
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) AuditLogPath() string {
	if s == nil {
		return ""
	}
	return s.auditLogPathFor(time.Now().UTC())
}

func (s *Store) migrate() error {
	schema := []string{
		`create table if not exists scans (
			scan_id text primary key,
			command text not null,
			profile text,
			platform text not null,
			created_at text not null,
			totals_json text not null,
			warnings_json text not null,
			plan_json text not null
		);`,
		`create table if not exists executions (
			id text primary key,
			scan_id text not null,
			started_at text not null,
			finished_at text not null,
			result_json text not null
		);`,
		`create table if not exists reports (
			id text primary key,
			scan_id text not null,
			path text not null,
			created_at text not null
		);`,
		`create table if not exists app_inventory (
			platform text primary key,
			updated_at text not null,
			apps_json text not null
		);`,
	}
	for _, stmt := range schema {
		if _, err := s.execWithRetry(context.Background(), stmt); err != nil {
			return err
		}
	}
	if _, err := s.execWithRetry(context.Background(), `pragma journal_mode = wal;`); err != nil {
		return err
	}
	if _, err := s.execWithRetry(context.Background(), `pragma busy_timeout = 5000;`); err != nil {
		return err
	}
	return nil
}

func (s *Store) SavePlan(ctx context.Context, plan domain.ExecutionPlan) error {
	totals, err := json.Marshal(plan.Totals)
	if err != nil {
		return err
	}
	warnings, err := json.Marshal(plan.Warnings)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	_, err = s.execWithRetry(ctx,
		`insert or replace into scans (scan_id, command, profile, platform, created_at, totals_json, warnings_json, plan_json)
		 values (?, ?, ?, ?, ?, ?, ?, ?)`,
		plan.ScanID, plan.Command, plan.Profile, plan.Platform, plan.CreatedAt.Format(time.RFC3339), string(totals), string(warnings), string(raw),
	)
	if err != nil {
		return err
	}
	return s.appendAuditRecord("plan", plan.ScanID, plan)
}

func (s *Store) GetPlan(ctx context.Context, scanID string) (domain.ExecutionPlan, error) {
	rows, err := s.queryWithRetry(ctx, `select plan_json from scans where scan_id = ?`, scanID)
	if err != nil {
		return domain.ExecutionPlan{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return domain.ExecutionPlan{}, err
		}
		return domain.ExecutionPlan{}, fmt.Errorf("scan %q not found", scanID)
	}
	var raw string
	if err := rows.Scan(&raw); err != nil {
		return domain.ExecutionPlan{}, err
	}
	var plan domain.ExecutionPlan
	return plan, json.Unmarshal([]byte(raw), &plan)
}

func (s *Store) SaveExecution(ctx context.Context, result domain.ExecutionResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = s.execWithRetry(ctx,
		`insert or replace into executions (id, scan_id, started_at, finished_at, result_json)
		 values (?, ?, ?, ?, ?)`,
		result.ID, result.ScanID, result.StartedAt.Format(time.RFC3339), result.FinishedAt.Format(time.RFC3339), string(raw),
	)
	if err != nil {
		return err
	}
	return s.appendAuditRecord("execution", result.ScanID, result)
}

func (s *Store) SaveReport(ctx context.Context, reportID, scanID, path string) error {
	_, err := s.execWithRetry(ctx,
		`insert or replace into reports (id, scan_id, path, created_at) values (?, ?, ?, ?)`,
		reportID, scanID, path, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	return s.appendAuditRecord("report", scanID, map[string]string{
		"report_id": reportID,
		"path":      path,
	})
}

func (s *Store) SaveAppInventory(ctx context.Context, platform string, apps []domain.AppEntry) error {
	raw, err := json.Marshal(apps)
	if err != nil {
		return err
	}
	_, err = s.execWithRetry(ctx,
		`insert or replace into app_inventory (platform, updated_at, apps_json) values (?, ?, ?)`,
		platform, time.Now().UTC().Format(time.RFC3339), string(raw),
	)
	return err
}

func (s *Store) LoadAppInventory(ctx context.Context, platform string) ([]domain.AppEntry, time.Time, error) {
	rows, err := s.queryWithRetry(ctx,
		`select updated_at, apps_json from app_inventory where platform = ? limit 1`,
		platform,
	)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, time.Time{}, err
		}
		return nil, time.Time{}, nil
	}
	var updatedAtRaw string
	var raw string
	if err := rows.Scan(&updatedAtRaw, &raw); err != nil {
		return nil, time.Time{}, err
	}
	var apps []domain.AppEntry
	if err := json.Unmarshal([]byte(raw), &apps); err != nil {
		return nil, time.Time{}, err
	}
	updatedAt, _ := time.Parse(time.RFC3339, updatedAtRaw)
	return apps, updatedAt, nil
}

func (s *Store) RecentScans(ctx context.Context, limit int) ([]RecentScan, error) {
	rows, err := s.queryWithRetry(ctx,
		`select scan_id, command, profile, platform, created_at, totals_json, warnings_json
		   from scans order by datetime(created_at) desc limit ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scans []RecentScan
	for rows.Next() {
		var scan RecentScan
		var createdAt string
		var totals string
		var warnings string
		if err := rows.Scan(&scan.ScanID, &scan.Command, &scan.Profile, &scan.Platform, &createdAt, &totals, &warnings); err != nil {
			return nil, err
		}
		scan.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		_ = json.Unmarshal([]byte(totals), &scan.Totals)
		_ = json.Unmarshal([]byte(warnings), &scan.Warnings)
		scans = append(scans, scan)
	}
	return scans, rows.Err()
}

func (s *Store) LatestExecution(ctx context.Context) (*ExecutionSummary, error) {
	return s.executionSummary(ctx,
		`select id, scan_id, started_at, finished_at, result_json
		   from executions order by datetime(finished_at) desc limit 1`,
	)
}

func (s *Store) GetExecutionForScan(ctx context.Context, scanID string) (*ExecutionSummary, error) {
	if strings.TrimSpace(scanID) == "" {
		return nil, nil
	}
	return s.executionSummary(ctx,
		`select id, scan_id, started_at, finished_at, result_json
		   from executions where scan_id = ? order by datetime(finished_at) desc limit 1`,
		scanID,
	)
}

func (s *Store) executionSummary(ctx context.Context, query string, args ...interface{}) (*ExecutionSummary, error) {
	var id, scanID, startedAt, finishedAt, raw string
	rows, err := s.queryWithRetry(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if err := rows.Scan(&id, &scanID, &startedAt, &finishedAt, &raw); err != nil {
		return nil, err
	}
	var result domain.ExecutionResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	summary := &ExecutionSummary{
		ID:               id,
		ScanID:           scanID,
		StartedAt:        result.StartedAt,
		FinishedAt:       result.FinishedAt,
		Warnings:         append([]string{}, result.Warnings...),
		FollowUpCommands: append([]string{}, result.FollowUpCommands...),
	}
	for _, item := range result.Items {
		summary.ItemCount++
		switch item.Status {
		case domain.StatusCompleted:
			summary.Completed++
			summary.FreedBytes += item.Bytes
		case domain.StatusDeleted:
			summary.Deleted++
			summary.FreedBytes += item.Bytes
		case domain.StatusFailed:
			summary.Failed++
		case domain.StatusProtected:
			summary.Protected++
		default:
			summary.Skipped++
		}
	}
	if summary.StartedAt.IsZero() {
		summary.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	}
	if summary.FinishedAt.IsZero() {
		summary.FinishedAt, _ = time.Parse(time.RFC3339, finishedAt)
	}
	return summary, nil
}

func (s *Store) BuildStatusSummary(ctx context.Context, platform string, limit int) (StatusSummary, error) {
	scans, err := s.RecentScans(ctx, limit)
	if err != nil {
		return StatusSummary{}, err
	}
	summary := StatusSummary{
		Platform:     platform,
		RecentScans:  scans,
		AuditLogPath: s.auditLogPathFor(time.Now().UTC()),
	}
	if len(scans) > 0 {
		summary.LastScan = &scans[0]
	}
	if len(scans) > 1 {
		summary.PreviousScan = &scans[1]
		summary.DeltaBytes = scans[0].Totals.Bytes - scans[1].Totals.Bytes
		summary.DeltaItems = scans[0].Totals.ItemCount - scans[1].Totals.ItemCount
	}
	execution, err := s.LatestExecution(ctx)
	if err != nil {
		return StatusSummary{}, err
	}
	summary.LastExecution = execution
	return summary, nil
}

func (s *Store) RecentAuditRecords(now time.Time, limit int) ([]AuditRecord, error) {
	path := s.auditLogPathFor(now)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var records []AuditRecord
	for scanner.Scan() {
		var record AuditRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(records) > limit {
		records = records[len(records)-limit:]
	}
	return records, nil
}

func (s *Store) AuditRecordsForScan(now time.Time, scanID string, limit int) ([]AuditRecord, error) {
	if strings.TrimSpace(scanID) == "" {
		return nil, nil
	}
	records, err := s.RecentAuditRecords(now, 0)
	if err != nil {
		return nil, err
	}
	filtered := make([]AuditRecord, 0, len(records))
	for _, record := range records {
		if record.ScanID == scanID {
			filtered = append(filtered, record)
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered, nil
}

type auditEnvelope struct {
	Kind      string      `json:"kind"`
	ScanID    string      `json:"scan_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

func (s *Store) appendAuditRecord(kind, scanID string, payload interface{}) error {
	path := s.auditLogPathFor(time.Now().UTC())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	record := auditEnvelope{
		Kind:      kind,
		ScanID:    scanID,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := writer.Write(raw); err != nil {
		return err
	}
	if _, err := writer.WriteString("\n"); err != nil {
		return err
	}
	return writer.Flush()
}

func (s *Store) auditLogPathFor(now time.Time) string {
	return filepath.Join(filepath.Dir(s.path), "audit", now.Format("2006-01-02")+".ndjson")
}

func (s *Store) execWithRetry(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var (
		result sql.Result
		err    error
	)
	for attempt := 0; attempt < 8; attempt++ {
		result, err = s.db.ExecContext(ctx, query, args...)
		if !isBusyError(err) {
			return result, err
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	return result, err
}

func (s *Store) queryWithRetry(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var (
		rows *sql.Rows
		err  error
	)
	for attempt := 0; attempt < 8; attempt++ {
		rows, err = s.db.QueryContext(ctx, query, args...)
		if !isBusyError(err) {
			return rows, err
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	return rows, err
}

func isBusyError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "sqlite_busy")
}

// ScanStats holds statistics about scans
type ScanStats struct {
	TotalScans       int              `json:"total_scans"`
	TotalBytesFound  int64            `json:"total_bytes_found"`
	TotalItemsFound  int              `json:"total_items_found"`
	AverageBytes     int64            `json:"average_bytes"`
	AverageItems     float64          `json:"average_items"`
	LargestScan      int64            `json:"largest_scan"`
	SmallestScan     int64            `json:"smallest_scan"`
	ProfileBreakdown map[string]int64 `json:"profile_breakdown"`
}

// ExecutionStats holds statistics about executions
type ExecutionStats struct {
	TotalExecutions int     `json:"total_executions"`
	TotalDeleted    int     `json:"total_deleted"`
	TotalFailed     int     `json:"total_failed"`
	TotalProtected  int     `json:"total_protected"`
	TotalFreedBytes int64   `json:"total_freed_bytes"`
	AverageFreed    int64   `json:"average_freed_bytes"`
	SuccessRate     float64 `json:"success_rate"`
}

// GetScanStats returns statistics about all scans
func (s *Store) GetScanStats(ctx context.Context) (*ScanStats, error) {
	stats := &ScanStats{
		ProfileBreakdown: make(map[string]int64),
	}

	// Get total scans
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scans").Scan(&stats.TotalScans)
	if err != nil {
		return nil, err
	}

	if stats.TotalScans == 0 {
		return stats, nil
	}

	// Get totals using JSON extraction
	var totalBytes int64
	var totalItems int
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(json_extract(totals_json, '$.bytes')), 0),
			COALESCE(SUM(json_extract(totals_json, '$.item_count')), 0)
		FROM scans`).Scan(&totalBytes, &totalItems)
	if err != nil {
		return nil, err
	}
	stats.TotalBytesFound = totalBytes
	stats.TotalItemsFound = totalItems

	// Calculate averages
	if stats.TotalScans > 0 {
		stats.AverageBytes = totalBytes / int64(stats.TotalScans)
		stats.AverageItems = float64(totalItems) / float64(stats.TotalScans)
	}

	// Get min/max
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(MAX(json_extract(totals_json, '$.bytes')), 0),
			COALESCE(MIN(json_extract(totals_json, '$.bytes')), 0)
		FROM scans`).Scan(&stats.LargestScan, &stats.SmallestScan)
	if err != nil {
		return nil, err
	}

	// Get profile breakdown
	rows, err := s.db.QueryContext(ctx, `
		SELECT profile, SUM(json_extract(totals_json, '$.bytes')) 
		FROM scans GROUP BY profile`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var profile string
		var bytes int64
		if err := rows.Scan(&profile, &bytes); err != nil {
			return nil, err
		}
		stats.ProfileBreakdown[profile] = bytes
	}

	return stats, nil
}

// GetExecutionStats returns statistics about all executions
func (s *Store) GetExecutionStats(ctx context.Context) (*ExecutionStats, error) {
	stats := &ExecutionStats{}

	// Get total executions
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM executions").Scan(&stats.TotalExecutions)
	if err != nil {
		return nil, err
	}

	if stats.TotalExecutions == 0 {
		return stats, nil
	}

	// Get totals from result_json
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(json_extract(result_json, '$.deleted')), 0),
			COALESCE(SUM(json_extract(result_json, '$.failed')), 0),
			COALESCE(SUM(json_extract(result_json, '$.protected')), 0),
			COALESCE(SUM(json_extract(result_json, '$.freed_bytes')), 0)
		FROM executions`).Scan(&stats.TotalDeleted, &stats.TotalFailed, &stats.TotalProtected, &stats.TotalFreedBytes)
	if err != nil {
		return nil, err
	}

	// Calculate averages and success rate
	if stats.TotalExecutions > 0 {
		stats.AverageFreed = stats.TotalFreedBytes / int64(stats.TotalExecutions)
		if stats.TotalDeleted+stats.TotalFailed > 0 {
			stats.SuccessRate = float64(stats.TotalDeleted) / float64(stats.TotalDeleted+stats.TotalFailed) * 100
		}
	}

	return stats, nil
}

// GetWeeklyTrend returns daily totals for the last N days
func (s *Store) GetWeeklyTrend(ctx context.Context, days int) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT date(created_at) as day, 
			SUM(json_extract(totals_json, '$.bytes')) as bytes, 
			SUM(json_extract(totals_json, '$.item_count')) as items
		FROM scans
		WHERE datetime(created_at) >= datetime('now', '-? days')
		GROUP BY date(created_at)
		ORDER BY day ASC`, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var day string
		var bytes int64
		var items int
		if err := rows.Scan(&day, &bytes, &items); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"date":  day,
			"bytes": bytes,
			"items": items,
		})
	}
	return results, nil
}

// GetMonthlyTrend returns weekly totals for the last N weeks
func (s *Store) GetMonthlyTrend(ctx context.Context, weeks int) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT strftime('%Y-W%W', created_at) as week, 
			SUM(json_extract(totals_json, '$.bytes')) as bytes, 
			SUM(json_extract(totals_json, '$.item_count')) as items
		FROM scans
		WHERE datetime(created_at) >= datetime('now', '-? weeks')
		GROUP BY strftime('%Y-W%W', created_at)
		ORDER BY week ASC`, weeks,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var week string
		var bytes int64
		var items int
		if err := rows.Scan(&week, &bytes, &items); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"week":  week,
			"bytes": bytes,
			"items": items,
		})
	}
	return results, nil
}
