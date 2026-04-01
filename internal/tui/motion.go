package tui

import (
	"strings"

	"github.com/batuhanyuksel/sift/internal/domain"
	"github.com/batuhanyuksel/sift/internal/engine"
	"github.com/batuhanyuksel/sift/internal/platform"
)

type motionMode string

const (
	motionModeIdle     motionMode = "idle"
	motionModeLoading  motionMode = "loading"
	motionModeProgress motionMode = "progress"
	motionModeAlert    motionMode = "alert"
	motionModeReview   motionMode = "review"
)

type motionState struct {
	Frame int
	Pulse bool
	Mode  motionMode
	Phase string
	Scene string
}

func newMotionState(frame int, pulse bool, mode motionMode, phase, scene string) motionState {
	if strings.TrimSpace(phase) == "" {
		phase = "steady"
	}
	if strings.TrimSpace(scene) == "" {
		scene = "rail"
	}
	return motionState{
		Frame: frame,
		Pulse: pulse,
		Mode:  mode,
		Phase: phase,
		Scene: scene,
	}
}

func signalRailLabel(frame int, pulse bool) string {
	return signalRailLabelForMotion(newMotionState(frame, pulse, motionModeIdle, "steady", "rail"))
}

func signalRailLabelForMotion(motion motionState) string {
	marks := []string{"S1", "S2", "S3", "S4"}
	switch motion.Mode {
	case motionModeLoading:
		marks = []string{"L1", "L2", "L3", "L4"}
	case motionModeProgress:
		marks = []string{"F1", "F2", "F3", "F4"}
	case motionModeAlert:
		marks = []string{"A1", "A2", "A3", "A4"}
	case motionModeReview:
		marks = []string{"R1", "R2", "R3", "R4"}
	}
	index := motion.Frame % len(marks)
	if index < 0 {
		index = 0
	}
	label := marks[index]
	suffix := strings.ToLower(strings.TrimSpace(motion.Phase))
	if suffix == "" {
		suffix = "steady"
	}
	if motion.Pulse {
		return railStyle.Render(label + " " + suffix)
	}
	return mutedStyle.Render(label + " " + suffix)
}

func footerMotionLabel(motion motionState) string {
	switch motion.Mode {
	case motionModeAlert:
		// Urgent pulse: double-ring ↔ asterisk
		if motion.Pulse {
			return "◈ LIVE RAIL 15s"
		}
		return "◎ LIVE RAIL 15s"
	case motionModeProgress:
		// Active pulse: hollow ↔ filled circle
		if motion.Pulse {
			return "● LIVE RAIL 15s"
		}
		return "◌ LIVE RAIL 15s"
	case motionModeReview:
		// Deliberate pulse: hollow ↔ filled diamond
		if motion.Pulse {
			return "◆ LIVE RAIL 15s"
		}
		return "◇ LIVE RAIL 15s"
	case motionModeLoading:
		if motion.Pulse {
			return "◉ LIVE RAIL 15s"
		}
		return "◌ LIVE RAIL 15s"
	default:
		if motion.Pulse {
			return "◉ LIVE RAIL 15s"
		}
		return "○ LIVE RAIL 15s"
	}
}

func loadingPulseLine(label string, motion motionState) string {
	display := loadingDisplayLabel(label)
	// Show dynamic stage for analyze operations
	analyzeStage := getAnalyzeStage(label, motion.Frame)
	if analyzeStage != "" {
		return spinnerGlyph(motion) + " " + loadingVerb(label) + " " + display + loadingPulseSuffix(motion.Frame, motion.Pulse) + "  •  " + analyzeStage
	}
	return spinnerGlyph(motion) + " " + loadingVerb(label) + " " + display + loadingPulseSuffix(motion.Frame, motion.Pulse) + "  •  " + loadingStageScript(label, motion)
}

func motionCadenceLabel(motion motionState) string {
	phase := strings.TrimSpace(motion.Phase)
	if phase == "" {
		phase = "steady"
	}
	scene := strings.TrimSpace(motion.Scene)
	if scene == "" {
		scene = "rail"
	}
	return scene + " cadence  •  " + phase
}

func motionSignatureBand(motion motionState) string {
	bands := []string{"▁▂▃▂", "▂▃▄▃", "▃▄▅▄", "▄▅▆▅"}
	switch motion.Mode {
	case motionModeAlert:
		bands = []string{"▅▃▆▃", "▆▂▇▂", "▇▃▆▃", "▆▂▅▂"}
	case motionModeProgress:
		bands = []string{"▁▃▆▇", "▂▄▆▇", "▃▅▇▇", "▄▆▇▇"}
	case motionModeReview:
		// Deliberate left-to-right peak sweep — contemplative rhythm.
		bands = []string{"▆▄▂▁", "▄▆▄▂", "▂▄▆▄", "▁▂▄▆"}
	case motionModeLoading:
		bands = []string{"▁▂▄▆", "▂▃▅▆", "▃▄▆▇", "▂▃▅▇"}
	}
	index := motion.Frame % len(bands)
	if index < 0 {
		index = 0
	}
	return bands[index]
}

func motionSceneAtmosphere(motion motionState) string {
	phase := strings.ToUpper(strings.TrimSpace(motion.Phase))
	if phase == "" {
		phase = "STEADY"
	}
	scene := strings.ToUpper(strings.TrimSpace(motion.Scene))
	if scene == "" {
		scene = "RAIL"
	}
	return motionSceneGlyph(motion) + " " + scene + " FIELD " + motionSignatureBand(motion) + "  •  " + phase + " WINDOW"
}

func motionSceneGlyph(motion motionState) string {
	switch strings.ToLower(strings.TrimSpace(motion.Scene)) {
	case "control":
		if motion.Mode == motionModeAlert {
			return "⌁"
		}
		return "⌂"
	case "monitor":
		if motion.Mode == motionModeAlert {
			return "◬"
		}
		return "◫"
	case "analyze":
		if motion.Mode == motionModeAlert {
			return "◪"
		}
		return "◧"
	case "task":
		if motion.Mode == motionModeAlert {
			return "✦"
		}
		return "◆"
	case "target":
		if motion.Mode == motionModeAlert {
			return "⬢"
		}
		return "⬡"
	case "cleanup":
		if motion.Mode == motionModeAlert {
			return "◩"
		}
		return "◨"
	case "decision":
		if motion.Mode == motionModeAlert {
			return "✧"
		}
		return "◇"
	case "inventory":
		return "▣"
	case "protect":
		return "▤"
	case "apply":
		return "▥"
	default:
		if motion.Mode == motionModeAlert {
			return "◎"
		}
		return "○"
	}
}

func spinnerGlyph(motion motionState) string {
	frames := spinnerFrames
	switch motion.Mode {
	case motionModeAlert:
		frames = []string{"◴", "◷", "◶", "◵"}
	case motionModeProgress:
		// Pulsing circle: empty → outline → filled → outline
		frames = []string{"◌", "◎", "●", "◎"}
	case motionModeReview:
		// Diamond sweep: empty → center-dot → filled → center-dot
		frames = []string{"◇", "◈", "◆", "◈"}
	}
	if len(frames) == 0 {
		return "•"
	}
	index := motion.Frame % len(frames)
	if index < 0 {
		index = 0
	}
	return frames[index]
}

func statusMotionState(model statusModel) motionState {
	mode := motionModeIdle
	if statusHasActiveAlerts(model.live, model.diagnostics, model.updateNotice) {
		mode = motionModeAlert
	}
	return newMotionState(model.signalFrame, model.pulse, mode, statusMotionPhase(model.live), "monitor")
}

func homeMotionState(model homeModel) motionState {
	mode := motionModeIdle
	if statusHasActiveAlerts(model.live, model.diagnostics, model.updateNotice) {
		mode = motionModeAlert
	}
	return newMotionState(model.signalFrame, model.pulse, mode, homeMotionPhase(model.live, model.diagnostics, model.updateNotice), "control")
}

func progressMotionState(progress progressModel) motionState {
	phase := "review"
	switch progress.currentPhase {
	case domain.ProgressPhaseStarting:
		phase = "stage"
	case domain.ProgressPhasePreparing:
		phase = "prepare"
	case domain.ProgressPhaseRunning:
		phase = "apply"
	case domain.ProgressPhaseVerifying:
		phase = "verify"
	case domain.ProgressPhaseFinished:
		phase = "settle"
	}
	if progress.cancelRequested {
		phase = "halt"
	}
	scene := "cleanup"
	if progress.plan.Command == "uninstall" {
		scene = "target"
	} else if progress.plan.Command == "optimize" || progress.plan.Command == "autofix" {
		scene = "task"
	}
	return newMotionState(progress.spinnerFrame, progress.pulse, motionModeProgress, phase, scene)
}

func appMotionState(model appModel) motionState {
	currentLoading := model.currentLoadingLabel()
	switch {
	case currentLoading != "":
		return newMotionState(model.spinnerFrame, model.livePulse, motionModeLoading, loadingPhase(currentLoading), loadingScene(currentLoading))
	case model.route == RouteProgress:
		return progressMotionState(model.progress)
	case model.route == RouteHome:
		return homeMotionState(model.home)
	case model.route == RouteStatus:
		return statusMotionState(model.status)
	case model.route == RouteDoctor:
		mode := motionModeIdle
		if diagnosticsHaveIssues(model.doctor.diagnostics) {
			mode = motionModeAlert
		}
		return newMotionState(model.spinnerFrame, model.livePulse, mode, "inspect", "doctor")
	case model.route == RouteReview || model.route == RoutePreflight || model.route == RouteResult:
		return newMotionState(model.spinnerFrame, model.livePulse, motionModeReview, "review", "decision")
	default:
		return newMotionState(model.spinnerFrame, model.livePulse, motionModeIdle, "steady", "rail")
	}
}

func loadingPhase(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(label, "recovery"), strings.Contains(label, "review"):
		return "review"
	case strings.Contains(label, "dashboard"), strings.Contains(label, "status"), strings.Contains(label, "doctor"):
		return "monitor"
	case strings.Contains(label, "analysis"), strings.Contains(label, "analyze"):
		return "inspect"
	case strings.Contains(label, "installed apps"):
		return "index"
	case strings.Contains(label, "execution"):
		return "apply"
	case strings.Contains(label, "protect path"), strings.Contains(label, "remove protected path"):
		return "verify"
	case strings.Contains(label, "clean"), strings.Contains(label, "review"):
		return "stage"
	default:
		return "sync"
	}
}

func loadingScene(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(label, "recovery"), strings.Contains(label, "review"):
		return "decision"
	case strings.Contains(label, "dashboard"), strings.Contains(label, "status"), strings.Contains(label, "doctor"):
		return "control"
	case strings.Contains(label, "analysis"), strings.Contains(label, "analyze"):
		return "analyze"
	case strings.Contains(label, "installed apps"):
		return "inventory"
	case strings.Contains(label, "protect path"), strings.Contains(label, "remove protected path"):
		return "protect"
	case strings.Contains(label, "execution"):
		return "apply"
	case strings.Contains(label, "clean"), strings.Contains(label, "review"):
		return "cleanup"
	default:
		return "sync"
	}
}

func loadingStageScript(label string, motion motionState) string {
	state := strings.TrimSpace(motion.Phase)
	if state == "" {
		state = "sync"
	}
	return loadingStageFlow(label) + "  •  " + state
}

func loadingStageFlow(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(label, "failed item review"):
		return "load review -> inspect -> open"
	case strings.Contains(label, "current module recovery"):
		return "load module -> inspect -> open"
	case strings.Contains(label, "issue recovery review"), strings.Contains(label, "staged cleanup review"), strings.Contains(label, "review"):
		return "load review -> inspect -> open"
	case strings.Contains(label, "dashboard"), strings.Contains(label, "status"), strings.Contains(label, "doctor"):
		return "load status -> refresh"
	case strings.Contains(label, "refresh analysis"), strings.Contains(label, "analysis refresh"):
		return "reload folder -> refresh"
	case strings.Contains(label, "open folder analysis"), strings.Contains(label, "analysis target"), strings.Contains(label, "analyze:"):
		return "scan disk -> find large -> analyze"
	case strings.Contains(label, "analyze"):
		return "scan home -> find large -> analyze"
	case strings.Contains(label, "installed apps"):
		return "index -> inspect"
	case strings.Contains(label, "protect path"), strings.Contains(label, "remove protected path"):
		return "save -> verify"
	case strings.Contains(label, "execution"):
		return "verify -> apply -> settle"
	default:
		return "load -> inspect"
	}
}

// analyzeStages returns cycling stage descriptions for analyze operations
// based on the motion frame to show progress
func analyzeStages(frame int) []string {
	return []string{
		"scanning disk usage...",
		"finding large files...",
		"detecting duplicates...",
		"building results...",
	}
}

// getAnalyzeStage returns a specific stage based on frame number
func getAnalyzeStage(label string, frame int) string {
	labelLower := strings.ToLower(label)
	if !strings.Contains(labelLower, "analyze") {
		return ""
	}
	stages := analyzeStages(frame)
	// Cycle through stages based on frame
	stageIndex := (frame / 40) % len(stages)
	return stages[stageIndex]
}

func loadingDisplayLabel(label string) string {
	raw := strings.TrimSpace(label)
	if raw == "" {
		return "current task"
	}
	// Handle analyze with target path (e.g., "analyze: /Users/...")
	if strings.HasPrefix(strings.ToLower(raw), "analyze:") {
		target := strings.TrimPrefix(raw, "analyze:")
		target = strings.TrimSpace(target)
		if target != "" {
			// Truncate long paths for display
			if len(target) > 40 {
				target = "..." + target[len(target)-37:]
			}
			return "scanning " + target
		}
		return "scanning files"
	}
	switch strings.ToLower(raw) {
	case "analysis target", "open folder analysis":
		return "scanning selected folder"
	case "analysis refresh", "refresh analysis view":
		return "scanning files"
	case "analyze":
		return "scanning files"
	case "cleanup review", "staged cleanup review":
		return "staged cleanup review"
	case "retry review", "failed item review":
		return "failed item review"
	case "module recovery review", "current module recovery":
		return "current module recovery review"
	case "recovery review", "issue recovery review":
		return "issue recovery review"
	case "installed apps":
		return "installed app inventory"
	case "execution":
		return "approved changes"
	}
	return raw
}

func loadingVerb(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(label, "refresh"), strings.Contains(label, "dashboard"), strings.Contains(label, "status"), strings.Contains(label, "doctor"):
		return "Refreshing"
	case strings.Contains(label, "execution"):
		return "Applying"
	case strings.Contains(label, "review"), strings.Contains(label, "recovery"):
		return "Preparing"
	case strings.Contains(label, "analyze"), strings.Contains(label, "analysis"), strings.Contains(label, "scanning"):
		return "Scanning"
	case strings.Contains(label, "installed apps"):
		return "Indexing"
	default:
		return "Loading"
	}
}

// diagnosticsHaveIssues returns true when any diagnostic has warn or error
// status. Used to gate motion modes — both statuses deserve the alert animation.
func diagnosticsHaveIssues(diagnostics []platform.Diagnostic) bool {
	for _, d := range diagnostics {
		if d.Status == "warn" || d.Status == "error" {
			return true
		}
	}
	return false
}

func statusHasActiveAlerts(live *engine.SystemSnapshot, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) bool {
	if diagnosticsHaveIssues(diagnostics) {
		return true
	}
	if update != nil && update.Available {
		return true
	}
	if live != nil && len(live.OperatorAlerts) > 0 {
		return true
	}
	if live != nil {
		if pressure := statusPressureLabel(live); pressure != "" && pressure != "steady" {
			return true
		}
		if live.ThermalState != "" && strings.ToLower(live.ThermalState) != "nominal" {
			return true
		}
	}
	return false
}

func homeMotionPhase(live *engine.SystemSnapshot, diagnostics []platform.Diagnostic, update *engine.UpdateNotice) string {
	switch {
	case statusHasActiveAlerts(live, diagnostics, update):
		return "alert"
	case live != nil:
		return "monitor"
	default:
		return "ready"
	}
}

func statusMotionPhase(live *engine.SystemSnapshot) string {
	if live == nil {
		return "idle"
	}
	if len(live.OperatorAlerts) > 0 {
		return "alert"
	}
	if pressure := statusPressureLabel(live); pressure != "" && pressure != "steady" {
		return "watch"
	}
	return "steady"
}
