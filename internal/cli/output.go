package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/batu3384/sift/internal/domain"
	"github.com/batu3384/sift/internal/platform"
	"github.com/fatih/color"
)

// nowUnix returns current Unix timestamp
func nowUnix() int64 {
	return time.Now().Unix()
}

// Color styles for CLI output
var (
	safeStyle   = color.New(color.FgGreen).Add(color.Bold)
	reviewStyle = color.New(color.FgYellow).Add(color.Bold)
	highStyle   = color.New(color.FgRed).Add(color.Bold)
	infoStyle   = color.New(color.FgCyan)
	mutedStyle  = color.New(color.FgWhite)
)

func (r *runtimeState) wantsJSONOutput(command string, writer io.Writer) bool {
	if r.flags.JSON {
		return true
	}
	if r.flags.Plain {
		return false
	}
	switch strings.TrimSpace(command) {
	case "status", "analyze", "check":
		return isPipedWriter(writer)
	default:
		return false
	}
}

func printCheckReport(writer io.Writer, report domain.CheckReport) error {
	findingWord := map[bool]string{true: "finding", false: "findings"}[report.Summary.Warn == 1]
	_, _ = fmt.Fprintf(writer, "CHECK  %s  %d %s  %d autofixable\n", report.Platform, report.Summary.Warn, findingWord, report.Summary.Autofixable)
	currentGroup := domain.CheckGroup("")
	for _, item := range report.Items {
		if item.Group != currentGroup {
			currentGroup = item.Group
			_, _ = fmt.Fprintf(writer, "%s\n", strings.ToUpper(string(currentGroup)))
		}
		_, _ = fmt.Fprintf(writer, "  [%s] %-20s %s\n", strings.ToUpper(item.Status), item.Name, item.Message)
		if len(item.Commands) > 0 {
			for _, command := range item.Commands[:min(len(item.Commands), 2)] {
				_, _ = fmt.Fprintf(writer, "        -> %s\n", command)
			}
		}
	}
	return nil
}

func isInteractiveTerminal() bool {
	if os.Getenv("SIFT_NO_TUI") == "1" {
		return false
	}
	if os.Getenv("SIFT_FORCE_TUI") == "1" {
		return true
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return os.Getenv("TERM") != ""
}

func isPipedWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok || file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

func resolveTUIEnabled(mode string, interactive bool) bool {
	switch mode {
	case "plain":
		return false
	case "tui":
		return interactive
	default:
		return interactive
	}
}

func printPlan(out *os.File, plan domain.ExecutionPlan, platformDebug bool, diagnostics []platform.Diagnostic) error {
	if plan.Command == "analyze" {
		return printAnalyzePlan(out, plan, platformDebug, diagnostics)
	}
	nItems := len(plan.Items)
	itemWord := map[bool]string{true: "item", false: "items"}[nItems == 1]

	// Print header with better formatting
	commandTitle := strings.ToUpper(plan.Command)
	if commandTitle == "" {
		commandTitle = "CLEAN"
	}
	_, _ = fmt.Fprintf(out, "\n")
	_, _ = fmt.Fprintf(out, "╭──────────────────────────────────────────────────────────╮\n")
	_, _ = fmt.Fprintf(out, "│ %s  •  %s reclaimable  •  %d %s  •  %s\n", commandTitle, domain.HumanBytes(plan.Totals.Bytes), nItems, itemWord, plan.Platform)
	_, _ = fmt.Fprintf(out, "╰──────────────────────────────────────────────────────────╯\n\n")

	// Print table header
	_, _ = fmt.Fprintf(out, "%-6s %-40s %12s %s\n", "STATUS", "PATH", "SIZE", "NOTES")
	_, _ = fmt.Fprintf(out, "%s\n", strings.Repeat("─", 80))

	currentCategory := domain.Category("")
	for _, item := range plan.Items {
		if item.Category != currentCategory {
			currentCategory = item.Category
			_, _ = fmt.Fprintf(out, "\n→ %s\n", domain.CategoryTitle(item.Category))
		}
		icon := "~"
		iconColor := color.New()
		if item.Status == domain.StatusProtected {
			icon = "⊘"
			iconColor = color.New(color.FgYellow)
		} else if item.Risk == domain.RiskSafe {
			icon = "✓"
			iconColor = color.New(color.FgGreen)
		} else if item.Risk == domain.RiskReview {
			icon = "◉"
			iconColor = color.New(color.FgCyan)
		} else if item.Risk == domain.RiskHigh {
			icon = "!"
			iconColor = color.New(color.FgRed)
		}
		label := strings.TrimSpace(item.DisplayPath)
		if label == "" {
			label = strings.TrimSpace(item.Path)
		}
		bytesLabel := domain.HumanBytes(item.Bytes)
		suffix := ""
		if item.Action == domain.ActionNative {
			suffix = "  [native]"
		} else if item.Action == domain.ActionCommand {
			suffix = "  [task]"
		}
		if item.Status == domain.StatusProtected && item.Policy.Reason != "" {
			suffix += "  [" + string(item.Policy.Reason) + "]"
		}
		// Truncate label if too long - show start of path for short paths, end for long
		maxLabelLen := 38
		if len(label) > maxLabelLen {
			// Show first 15 chars + ... + last 20 chars
			if len(label) > maxLabelLen+3 {
				label = label[:15] + "..." + label[len(label)-20:]
			}
		}

		_, _ = iconColor.Fprintf(out, "%-6s ", icon)
		_, _ = fmt.Fprintf(out, "%-40s %12s%s\n", label, bytesLabel, suffix)
	}
	if len(plan.Warnings) > 0 {
		_, _ = fmt.Fprintln(out, "Warnings:")
		for _, warning := range plan.Warnings {
			_, _ = fmt.Fprintf(out, "  - %s\n", warning)
		}
	}
	if platformDebug {
		_, _ = fmt.Fprintln(out, "Platform diagnostics:")
		for _, diagnostic := range diagnostics {
			_, _ = fmt.Fprintf(out, "  - [%s] %s: %s\n", diagnostic.Status, diagnostic.Name, diagnostic.Message)
		}
	}
	return nil
}

func printAnalyzePlan(out *os.File, plan domain.ExecutionPlan, platformDebug bool, diagnostics []platform.Diagnostic) error {
	_, _ = fmt.Fprintf(out, "ANALYZE  %s  %s\n", domain.HumanBytes(plan.Totals.Bytes), plan.Platform)
	diskUsage := planItemsByCategory(plan.Items, domain.CategoryDiskUsage)
	largeFiles := planItemsByCategory(plan.Items, domain.CategoryLargeFiles)
	_, _ = fmt.Fprintf(out, "Largest children: %d  Large files: %d\n", len(diskUsage), len(largeFiles))
	if len(diskUsage) > 0 {
		_, _ = fmt.Fprintf(out, "Top child: %s (%s)\n", diskUsage[0].DisplayPath, domain.HumanBytes(diskUsage[0].Bytes))
	}
	if len(largeFiles) > 0 {
		_, _ = fmt.Fprintf(out, "Top file: %s (%s)\n", largeFiles[0].DisplayPath, domain.HumanBytes(largeFiles[0].Bytes))
	}
	if len(diskUsage) > 0 {
		_, _ = fmt.Fprintln(out, "Largest children:")
		for _, item := range diskUsage {
			_, _ = fmt.Fprintf(out, "  %-10s %s\n", domain.HumanBytes(item.Bytes), item.DisplayPath)
		}
	}
	if len(largeFiles) > 0 {
		_, _ = fmt.Fprintln(out, "Large files:")
		for _, item := range largeFiles {
			_, _ = fmt.Fprintf(out, "  %-10s %s\n", domain.HumanBytes(item.Bytes), item.DisplayPath)
		}
	}
	if len(plan.Warnings) > 0 {
		_, _ = fmt.Fprintln(out, "Warnings:")
		for _, warning := range plan.Warnings {
			_, _ = fmt.Fprintf(out, "  - %s\n", warning)
		}
	}
	if platformDebug {
		_, _ = fmt.Fprintln(out, "Platform diagnostics:")
		for _, diagnostic := range diagnostics {
			_, _ = fmt.Fprintf(out, "  - [%s] %s: %s\n", diagnostic.Status, diagnostic.Name, diagnostic.Message)
		}
	}
	return nil
}

func confirm(plan domain.ExecutionPlan) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	nActions := actionableItemCount(plan)
	actionWord := map[bool]string{true: "planned action", false: "planned actions"}[nActions == 1]
	_, _ = fmt.Fprintf(os.Stdout, "Execute %d %s (%s reclaimable)? [y/N]: ", nActions, actionWord, domain.HumanBytes(plan.Totals.Bytes))
	raw, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	value := strings.TrimSpace(strings.ToLower(raw))
	return value == "y" || value == "yes", nil
}

func looksLikePath(value string) bool {
	if strings.HasPrefix(value, "~") {
		return true
	}
	if filepath.IsAbs(value) {
		return true
	}
	return strings.Contains(value, string(os.PathSeparator))
}

func actionableItemCount(plan domain.ExecutionPlan) int {
	count := 0
	for _, item := range plan.Items {
		if item.Status == domain.StatusProtected || item.Action == domain.ActionAdvisory {
			continue
		}
		count++
	}
	return count
}

func shouldExecutePlan(plan domain.ExecutionPlan) bool {
	if plan.Command == "analyze" || plan.PlanState == "empty" || plan.DryRun {
		return false
	}
	return actionableItemCount(plan) > 0
}

func planItemsByCategory(items []domain.Finding, category domain.Category) []domain.Finding {
	filtered := make([]domain.Finding, 0, len(items))
	for _, item := range items {
		if item.Category == category {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func formatFloatSeries(values []float64, limit int) string {
	if len(values) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(values) {
		limit = len(values)
	}
	parts := make([]string, 0, limit+1)
	for _, value := range values[:limit] {
		parts = append(parts, fmt.Sprintf("%.0f%%", value))
	}
	if len(values) > limit {
		parts = append(parts, fmt.Sprintf("+%d more", len(values)-limit))
	}
	return strings.Join(parts, " ")
}

// ProgressOutput provides CLI progress feedback like Mole
type ProgressOutput struct {
	out           *os.File
	totalBytes    int64
	showProgress  bool
	categoryCount int
	currentIndex  int
	startTime     int64
	spinnerFrame  int
}

// Spinner frames for animation
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewProgressOutput creates a new progress output handler
func NewProgressOutput(out *os.File) *ProgressOutput {
	return &ProgressOutput{
		out:          out,
		showProgress: true,
		startTime:    0,
	}
}

// getSpinner returns the current spinner character
func (p *ProgressOutput) getSpinner() string {
	frame := p.spinnerFrame % len(spinnerChars)
	return spinnerChars[frame]
}

// advanceSpinner advances the spinner frame
func (p *ProgressOutput) advanceSpinner() {
	p.spinnerFrame++
}

// formatProgressBar creates a progress bar with percentage
func formatProgressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := int(float64(current) / float64(total) * float64(width))
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// Disable turns off progress output
func (p *ProgressOutput) Disable() {
	p.showProgress = false
}

// OnCategoryScan is called after each category scan
func (p *ProgressOutput) OnCategoryScan(ruleID string, ruleName string, itemsFound int, bytesFound int64) {
	if !p.showProgress {
		return
	}

	// Initialize start time on first call
	if p.startTime == 0 {
		p.startTime = nowUnix()
	}

	p.totalBytes += bytesFound
	p.currentIndex++

	// Get spinner and create progress bar
	spinner := p.getSpinner()
	p.advanceSpinner()

	// Color the output based on bytes found
	bytesStr := domain.HumanBytes(bytesFound)
	var bytesColored string
	if bytesFound > 1_000_000_000 { // > 1GB
		bytesColored = highStyle.Sprint(bytesStr)
	} else if bytesFound > 100_000_000 { // > 100MB
		bytesColored = reviewStyle.Sprint(bytesStr)
	} else {
		bytesColored = safeStyle.Sprint(bytesStr)
	}

	// Print progress line for each rule (show all rules, not just overwrite)
	progressBar := formatProgressBar(p.currentIndex, max(p.categoryCount, 1), 10)
	_, _ = fmt.Fprintf(p.out, "  %s [%s] %-30s %s  (%d items)\n",
		spinner,
		progressBar,
		ruleName+":",
		bytesColored,
		itemsFound,
	)
	p.out.Sync()
}

// SetCategoryCount sets the total number of categories for progress calculation
func (p *ProgressOutput) SetCategoryCount(count int) {
	p.categoryCount = count
}

// OnScanStart is called when scanning begins
func (p *ProgressOutput) OnScanStart(profile string) {
	if !p.showProgress {
		return
	}
	p.startTime = nowUnix()
	p.currentIndex = 0
	p.totalBytes = 0
	p.spinnerFrame = 0

	// Print header
	_, _ = infoStyle.Fprintf(p.out, "\n╭─────────────────────────────────────────────╮\n")
	_, _ = fmt.Fprintf(p.out, "│ ")
	_, _ = infoStyle.Fprintf(p.out, "Sift")
	_, _ = fmt.Fprintf(p.out, " scanning with profile: ")
	_, _ = reviewStyle.Fprintf(p.out, "%s\n", profile)
	_, _ = fmt.Fprintf(p.out, "│ ")
	_, _ = mutedStyle.Fprintf(p.out, "Press Ctrl+C to cancel\n")
	_, _ = infoStyle.Fprintf(p.out, "╰─────────────────────────────────────────────╯\n\n")
}

// OnScanComplete is called when scanning is complete
func (p *ProgressOutput) OnScanComplete(totalBytes int64, totalItems int) {
	if !p.showProgress {
		return
	}

	// Calculate scan duration
	var durationStr string
	if p.startTime > 0 {
		elapsed := nowUnix() - p.startTime
		if elapsed < 60 {
			durationStr = fmt.Sprintf("%ds", elapsed)
		} else if elapsed < 3600 {
			durationStr = fmt.Sprintf("%.1fm", float64(elapsed)/60)
		} else {
			durationStr = fmt.Sprintf("%.1fh", float64(elapsed)/3600)
		}
	}

	// Print enhanced summary
	_, _ = fmt.Fprintf(p.out, "\n")
	_, _ = safeStyle.Fprintf(p.out, "✓ Scan complete")
	_, _ = fmt.Fprintf(p.out, ": %s reclaimable from %d items", domain.HumanBytes(totalBytes), totalItems)
	if durationStr != "" {
		_, _ = fmt.Fprintf(p.out, " in %s", durationStr)
	}
	_, _ = fmt.Fprintf(p.out, "\n")
}

// PrintRunningAppWarning prints a warning if an app is running
func PrintRunningAppWarning(out *os.File, appName string) {
	_, _ = fmt.Fprintf(out, "⚠ %s running · cleanup skipped for this category\n", appName)
}
