package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestMascotExpressionIdleKnipsBothEyes checks that idle mode shows relaxed
// eyes (◡◡) by default and alternates left/right blinks on frames 1 and 3.
func TestMascotExpressionIdleKnipsBothEyes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		frame int
		e1    string
		e2    string
	}{
		{0, "◡", "◡"},
		{2, "◡", "◡"},
		{1, "─", "◡"}, // left blinks
		{3, "◡", "─"}, // right blinks
	}
	for _, tc := range cases {
		e1, e2, mouth, tone := mascotExpression(tc.frame, motionModeIdle)
		if e1 != tc.e1 || e2 != tc.e2 {
			t.Errorf("idle frame %d: expected %q/%q, got %q/%q", tc.frame, tc.e1, tc.e2, e1, e2)
		}
		if mouth != "‿" {
			t.Errorf("idle mouth: expected ‿, got %q", mouth)
		}
		if tone != "safe" {
			t.Errorf("idle tone: expected safe, got %q", tone)
		}
	}
}

// TestMascotExpressionAlertIsFixed verifies that alert mode shows the same
// tense expression (◈◈/∧ high) on every animation frame — no flickering.
func TestMascotExpressionAlertIsFixed(t *testing.T) {
	t.Parallel()

	for frame := 0; frame < 4; frame++ {
		e1, e2, mouth, tone := mascotExpression(frame, motionModeAlert)
		if e1 != "◈" || e2 != "◈" {
			t.Errorf("alert frame %d: expected ◈/◈, got %q/%q", frame, e1, e2)
		}
		if mouth != "∧" {
			t.Errorf("alert frame %d: expected mouth ∧, got %q", frame, mouth)
		}
		if tone != "high" {
			t.Errorf("alert frame %d: expected tone high, got %q", frame, tone)
		}
	}
}

// TestMascotExpressionProgressAlternatesEyes checks that progress mode
// alternates which eye is highlighted on each frame to convey active scanning.
func TestMascotExpressionProgressAlternatesEyes(t *testing.T) {
	t.Parallel()

	e1even, e2even, mouth, tone := mascotExpression(0, motionModeProgress)
	e1odd, e2odd, _, _ := mascotExpression(1, motionModeProgress)

	if e1even == e1odd && e2even == e2odd {
		t.Errorf("progress eyes should alternate, got identical %q/%q for frames 0 and 1", e1even, e2even)
	}
	if mouth != "○" {
		t.Errorf("progress mouth: expected ○, got %q", mouth)
	}
	if tone != "review" {
		t.Errorf("progress tone: expected review, got %q", tone)
	}
}

// TestMascotExpressionLoadingHasMutedTone checks that loading mode uses a
// calm, muted mouth tone (○ muted) — distinct from progress (○ review).
func TestMascotExpressionLoadingHasMutedTone(t *testing.T) {
	t.Parallel()

	_, _, mouth, tone := mascotExpression(0, motionModeLoading)
	if mouth != "○" {
		t.Errorf("loading mouth: expected ○, got %q", mouth)
	}
	if tone != "muted" {
		t.Errorf("loading tone: expected muted, got %q", tone)
	}
}

// TestMascotExpressionReviewSwapsDiamonds confirms that review mode swaps
// ◆/◇ between frames so the mascot appears to "think".
func TestMascotExpressionReviewSwapsDiamonds(t *testing.T) {
	t.Parallel()

	e1even, e2even, _, _ := mascotExpression(0, motionModeReview)
	e1odd, e2odd, _, _ := mascotExpression(1, motionModeReview)

	if e1even == e1odd || e2even == e2odd {
		t.Errorf("review eyes should swap between frames: frame0=%q/%q frame1=%q/%q", e1even, e2even, e1odd, e2odd)
	}
}

// TestMascotActivityBarsIsAlwaysFiveChars ensures the CPU bar is exactly
// 5 characters wide regardless of CPU load or animation mode.
func TestMascotActivityBarsIsAlwaysFiveChars(t *testing.T) {
	t.Parallel()

	modes := []motionMode{
		motionModeIdle, motionModeLoading, motionModeProgress,
		motionModeAlert, motionModeReview,
	}
	cpuValues := []float64{0.0, 14.0, 50.0, 85.0, 100.0}

	for _, mode := range modes {
		for _, cpu := range cpuValues {
			for frame := 0; frame < 4; frame++ {
				bars := mascotActivityBars(cpu, frame, mode)
				if n := len([]rune(bars)); n != 5 {
					t.Errorf("mode=%v cpu=%.0f frame=%d: expected 5-char bar, got %d: %q", mode, cpu, frame, n, bars)
				}
			}
		}
	}
}

// TestMascotActivityBarsHigherCPUMeansHigherBars verifies that bars at 100%
// CPU are visually taller (later in the bar slice) than at 0%.
func TestMascotActivityBarsHigherCPUMeansHigherBars(t *testing.T) {
	t.Parallel()

	minBars := mascotActivityBars(0.0, 0, motionModeIdle)
	maxBars := mascotActivityBars(100.0, 0, motionModeIdle)

	if minBars == maxBars {
		t.Errorf("expected different bars at 0%% vs 100%% CPU, both got %q", minBars)
	}
}

// TestMascotActivityBarsModesDifferAtSameCPU ensures different modes produce
// different wave patterns so the mascot communicates system state visually.
func TestMascotActivityBarsModesDifferAtSameCPU(t *testing.T) {
	t.Parallel()

	idle := mascotActivityBars(50.0, 0, motionModeIdle)
	progress := mascotActivityBars(50.0, 0, motionModeProgress)
	alert := mascotActivityBars(50.0, 0, motionModeAlert)

	if idle == progress {
		t.Errorf("idle and progress bars should differ at same cpu/frame, both %q", idle)
	}
	if progress == alert {
		t.Errorf("progress and alert bars should differ at same cpu/frame, both %q", progress)
	}
}

// TestMascotFrameIsSixLines checks the core contract: the mascot is always
// exactly 6 lines for all motion modes. This keeps the JoinHorizontal layout
// predictable when placed next to content columns.
func TestMascotFrameIsSixLines(t *testing.T) {
	t.Parallel()

	modes := []motionMode{
		motionModeIdle, motionModeLoading, motionModeProgress,
		motionModeAlert, motionModeReview,
	}
	for _, mode := range modes {
		motion := newMotionState(0, false, mode, "test", "test")
		out := mascotFrame(motion, 0.0)
		lines := strings.Split(out, "\n")
		if len(lines) != 6 {
			t.Errorf("mode %v: expected 6 lines, got %d", mode, len(lines))
		}
	}
}

// TestMascotFrameEachLineIsSevenWide verifies that after stripping ANSI codes
// every line of the mascot renders at exactly 7 visible characters. This
// ensures the column reservation (width-11) in the layout code stays accurate.
func TestMascotFrameEachLineIsSevenWide(t *testing.T) {
	t.Parallel()

	modes := []motionMode{
		motionModeIdle, motionModeLoading, motionModeProgress,
		motionModeAlert, motionModeReview,
	}
	for _, mode := range modes {
		for frame := 0; frame < 4; frame++ {
			motion := newMotionState(frame, frame%2 == 1, mode, "apply", "task")
			out := mascotFrame(motion, 42.0)
			lines := strings.Split(out, "\n")
			for i, line := range lines {
				stripped := ansi.Strip(line)
				width := ansi.StringWidth(stripped)
				if width != 7 {
					t.Errorf("mode=%v frame=%d line %d: expected 7 visible chars, got %d: %q", mode, frame, i+1, width, stripped)
				}
			}
		}
	}
}

// TestMascotFrameNegativeFrameDoesNotPanic ensures the frame<0 guard in
// mascotFrame prevents out-of-bounds slicing when motion.Frame is negative.
func TestMascotFrameNegativeFrameDoesNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("mascotFrame panicked with negative frame: %v", r)
		}
	}()

	motion := newMotionState(-5, false, motionModeIdle, "steady", "rail")
	_ = mascotFrame(motion, 0.0)
}
