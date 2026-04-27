// Package design provides the modern motion system for SIFT TUI.
// Smooth, professional animations with easing functions.
package design

import (
	"strings"
	"time"
)

// Motion modes
type MotionMode int

const (
	MotionModeInstant MotionMode = iota
	MotionModeSmooth
	MotionModeBounce
	MotionModeElastic
)

// Easing functions for smooth animations
type EasingFunc func(t float64) float64

// Linear easing - constant speed
func LinearEasing(t float64) float64 {
	return t
}

// EaseOutQuad - fast start, slow end (decelerating)
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInQuad - slow start, fast end (accelerating)
func EaseInQuad(t float64) float64 {
	return t * t
}

// EaseInOutQuad - slow start and end, fast middle
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// EaseOutCubic - smooth deceleration
func EaseOutCubic(t float64) float64 {
	return (t-1)*t*t + 1
}

// EaseOutExpo - very fast start, very slow end
func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - pow2(-10*t)
}

// SpringEasing - bouncy spring effect
func SpringEasing(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	// Damped sine wave
	return 1 - pow2(10-t*10) * cos(t*3.14159*4)
}

// ElasticEasing - elastic bounce
func ElasticEasing(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return pow2(10*(t-1)) * cos((t-1)*3.14159*3.5)
}

// Helper math functions
func pow2(n float64) float64 {
	if n < -700 {
		return 0
	}
	result := 1.0
	exp := n
	negative := exp < 0
	if negative {
		exp = -exp
	}
	for exp >= 1 {
		if exp >= 64 {
			result *= 1.8446744073709551616e19
			exp -= 64
		} else if exp >= 32 {
			result *= 4294967296.0
			exp -= 32
		} else if exp >= 16 {
			result *= 65536.0
			exp -= 16
		} else if exp >= 8 {
			result *= 256.0
			exp -= 8
		} else if exp >= 4 {
			result *= 16.0
			exp -= 4
		} else if exp >= 2 {
			result *= 4.0
			exp -= 2
		} else {
			result *= 2.0
			exp -= 1
		}
	}
	if n < 0 {
		return 1.0 / result
	}
	return result
}

func cos(x float64) float64 {
	return cosSmall(x)
}

func cosSmall(x float64) float64 {
	// Normalize to [-pi, pi]
	for x > 3.14159 {
		x -= 6.28318
	}
	for x < -3.14159 {
		x += 6.28318
	}
	// Taylor series approximation
	x2 := x * x
	return 1 - x2/2 + x2*x2/24 - x2*x2*x2/720
}

// Animation timing constants
const (
	// Duration constants - use these for consistent timing
	DurationInstant  = 0 * time.Millisecond
	DurationFast    = 100 * time.Millisecond
	DurationNormal  = 200 * time.Millisecond
	DurationSlow    = 300 * time.Millisecond
	DurationSlower  = 500 * time.Millisecond

	// Frame timing
	FrameRate60fps   = 16 * time.Millisecond
	FrameRate30fps   = 33 * time.Millisecond

	// Spinner timing
	SpinnerFastDuration   = 80 * time.Millisecond
	SpinnerNormalDuration = 120 * time.Millisecond
	SpinnerSlowDuration   = 200 * time.Millisecond

	// Pulse animation
	PulseFastDuration    = 600 * time.Millisecond
	PulseNormalDuration  = 1000 * time.Millisecond
	PulseSlowDuration    = 2000 * time.Millisecond
)

// AnimationConfig holds animation configuration
type AnimationConfig struct {
	Enabled         bool
	ReducedMotion   bool
	Duration        time.Duration
	Easing          EasingFunc
	Repeat          bool
	AutoReverse     bool
}

// DefaultAnimationConfig returns the default animation configuration
func DefaultAnimationConfig() AnimationConfig {
	return AnimationConfig{
		Enabled:       true,
		ReducedMotion: false,
		Duration:      DurationNormal,
		Easing:        EaseOutQuad,
		Repeat:        false,
		AutoReverse:   false,
	}
}

// ReducedMotionConfig returns config optimized for reduced motion
func ReducedMotionConfig() AnimationConfig {
	return AnimationConfig{
		Enabled:       false,
		ReducedMotion: true,
		Duration:       DurationInstant,
		Easing:        LinearEasing,
		Repeat:         false,
		AutoReverse:    false,
	}
}

// Progress animation calculator
type ProgressAnimation struct {
	From   float64
	To     float64
	During time.Duration
	Start  time.Time
	Easing EasingFunc
}

// NewProgressAnimation creates a new progress animation
func NewProgressAnimation(from, to float64, during time.Duration, easing EasingFunc) *ProgressAnimation {
	return &ProgressAnimation{
		From:   from,
		To:     to,
		During: during,
		Start:  time.Now(),
		Easing: easing,
	}
}

// Current returns the current interpolated value
func (p *ProgressAnimation) Current() float64 {
	elapsed := time.Since(p.Start)
	if elapsed >= p.During {
		return p.To
	}

	t := float64(elapsed) / float64(p.During)
	if p.Easing != nil {
		t = p.Easing(t)
	}

	return p.From + (p.To-p.From)*t
}

// IsDone returns true if animation is complete
func (p *ProgressAnimation) IsDone() bool {
	return time.Since(p.Start) >= p.During
}

// Frame sequence for smooth animations
type FrameSequence struct {
	Frames     []string
	Duration   time.Duration
	CurrentIdx int
	LastUpdate time.Time
}

// NewFrameSequence creates a frame sequence with timing
func NewFrameSequence(frames []string, frameDuration time.Duration) *FrameSequence {
	return &FrameSequence{
		Frames:     frames,
		Duration:   frameDuration,
		CurrentIdx: 0,
		LastUpdate: time.Now(),
	}
}

// Next advances to next frame and returns the current frame
func (f *FrameSequence) Next() string {
	now := time.Now()
	if now.Sub(f.LastUpdate) >= f.Duration {
		f.CurrentIdx = (f.CurrentIdx + 1) % len(f.Frames)
		f.LastUpdate = now
	}
	if len(f.Frames) == 0 {
		return ""
	}
	return f.Frames[f.CurrentIdx]
}

// Reset restarts the sequence
func (f *FrameSequence) Reset() {
	f.CurrentIdx = 0
	f.LastUpdate = time.Now()
}

// Spinner frames - modern, clean design
var SpinnerFrames = []string{
	"\u2801", // ▁
	"\u2802", // ▂
	"\u2803", // ▃
	"\u2804", // ▄
	"\u2805", // ▅
	"\u2806", // ▆
	"\u2807", // ▇
	"\u2808", // █
	"\u2809", // ▉
	"\u280A", // ▊
	"\u280B", // ▋
	"\u280C", // ▌
	"\u280D", // ▍
	"\u280E", // ▎
	"\u280F", // ▏
	"\u2810", // ▐
	"\u2811", // ░
	"\u2812", // ▒
	"\u2813", // ▓
	"\u2814", // ▔
}

// Alternative spinner - dots style
var SpinnerDotsFrames = []string{
	"⠋",
	"⠙",
	"⠹",
	"⠸",
	"⠼",
	"⠴",
	"⠦",
	"⠧",
	"⠇",
	"⠏",
}

// Alternative spinner - simple lines
var SpinnerLinesFrames = []string{
	"╱",
	"╲",
	"╱",
	"╲",
}

// FadeFrameSequence creates a fade in/out sequence
func FadeFrameSequence(min, max float64, steps int) []string {
	frames := make([]string, steps)
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		opacity := min + (max-min)*t
		// Visual representation using block characters
		filled := int(opacity * 8)
		frames[i] = strings.Repeat("█", filled) + strings.Repeat("░", 8-filled)
	}
	return frames
}

// PulseSequence creates a pulsing sequence
func PulseSequence(onFrames, offFrames []string, pulseDuration time.Duration) []string {
	result := make([]string, 0, len(onFrames)+len(offFrames))
	result = append(result, onFrames...)
	result = append(result, offFrames...)
	return result
}

// Status indicator animation frames
var StatusAnimations = map[string][]string{
	"loading": SpinnerFrames,
	"scanning": {
		"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█",
	},
	"success": {
		"✓",
	},
	"error": {
		"✗",
	},
	"warning": {
		"⚠",
	},
}