package tui

import "strings"

// mascotFrame renders the 6-line × 7-char animated SIFT mascot.
//
// Uses simple ASCII-friendly characters for a professional look:
//   - idle:     * *  u   → relaxed, eyes blink alternately
//   - loading:  o o  0   → attentive, waiting
//   - progress: O/o  0    → eyes scan left-right, working
//   - alert:    X X  ^   → concerned
//   - review:   <> ~    → thoughtful
//
// Alt satırdaki 5 karakterlik bar, CPU/ilerleme yüküne göre titreşir.
func mascotFrame(motion motionState, cpuPercent float64) string {
	frame := motion.Frame % 4
	if frame < 0 {
		frame = 0
	}
	e1, e2, mouth, mouthTone := mascotExpression(frame, motion.Mode)
	bars := mascotActivityBars(cpuPercent, frame, motion.Mode)

	b := accentFrameStyle.Render
	eye1 := railStyle.Render(e1)
	eye2 := railStyle.Render(e2)

	var mouthR string
	switch mouthTone {
	case "safe":
		mouthR = safeStyle.Render(mouth)
	case "review":
		mouthR = reviewStyle.Render(mouth)
	case "high":
		mouthR = highStyle.Render(mouth)
	default:
		mouthR = mutedStyle.Render(mouth)
	}

	var barR string
	switch motion.Mode {
	case motionModeAlert:
		barR = highStyle.Render(bars)
	case motionModeProgress:
		barR = reviewStyle.Render(bars)
	case motionModeLoading:
		barR = reviewStyle.Render(bars)
	default:
		barR = mutedStyle.Render(bars)
	}

	// Her satır tam olarak 7 görünür karakter genişliğinde.
	lines := []string{
		b("+-----+"),
		b("|") + " " + eye1 + " " + eye2 + " " + b("|"),
		b("|") + "  " + mouthR + "  " + b("|"),
		b("`--+--'"),
		b("   |   "),
		" " + barR + " ",
	}
	return strings.Join(lines, "\n")
}

func compactMascotFrame(motion motionState, cpuPercent float64) string {
	frame := motion.Frame % 4
	if frame < 0 {
		frame = 0
	}
	e1, e2, _, _ := mascotExpression(frame, motion.Mode)
	bars := []rune(mascotActivityBars(cpuPercent, frame, motion.Mode))
	if len(bars) < 3 {
		bars = []rune{'-', '-', '-'}
	}
	center := string(bars[1:4])
	b := accentFrameStyle.Render
	eye1 := railStyle.Render(e1)
	eye2 := railStyle.Render(e2)

	var barR string
	switch motion.Mode {
	case motionModeAlert:
		barR = highStyle.Render(center)
	case motionModeProgress, motionModeLoading:
		barR = reviewStyle.Render(center)
	default:
		barR = mutedStyle.Render(center)
	}

	lines := []string{
		b("+---+"),
		b("|") + eye1 + eye2 + b("|"),
		b("`") + barR + b("'"),
	}
	return strings.Join(lines, "\n")
}

// mascotExpression returns (eye1, eye2, mouth, mouthTone) for the given frame and mode.
func mascotExpression(frame int, mode motionMode) (eye1, eye2, mouth, tone string) {
	switch mode {
	case motionModeIdle:
		// Rahat — art arda sol/sağ göz kırpar.
		mouth, tone = "u", "safe"
		switch frame {
		case 1:
			eye1, eye2 = "-", "*"
		case 3:
			eye1, eye2 = "*", "-"
		default:
			eye1, eye2 = "*", "*"
		}
	case motionModeLoading:
		// Dikkatli — veri bekleniyor.
		mouth, tone = "0", "muted"
		if frame%2 == 0 {
			eye1, eye2 = "o", "o"
		} else {
			eye1, eye2 = "O", "o"
		}
	case motionModeProgress:
		// Aktif tarama — sol-sağ gözler değişir.
		mouth, tone = "0", "review"
		if frame%2 == 0 {
			eye1, eye2 = "O", "o"
		} else {
			eye1, eye2 = "o", "O"
		}
	case motionModeAlert:
		// Endişeli — sabit bakış.
		eye1, eye2 = "X", "X"
		mouth, tone = "^", "high"
	case motionModeReview:
		// Düşünceli — gözler titreşir.
		mouth, tone = "~", "muted"
		if frame%2 == 0 {
			eye1, eye2 = "<", ">"
		} else {
			eye1, eye2 = ">", "<"
		}
	default:
		eye1, eye2 = "o", "o"
		mouth, tone = "~", "muted"
	}
	return
}

// mascotActivityBars returns a 5-character bar string that pulses with
// CPU / progress load. Pattern changes based on motion mode for visual variety.
// Uses block Unicode characters for a professional look.
func mascotActivityBars(cpuPercent float64, frame int, mode motionMode) string {
	// 8 levels using Unicode block characters
	bars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	level := int(cpuPercent / 14.3) // 0-100 → 0-7
	if level > 7 {
		level = 7
	}
	if level < 0 {
		level = 0
	}

	// 5-position wave patterns — each frame has different height offset.
	var offsets [4][5]int
	switch mode {
	case motionModeProgress, motionModeLoading:
		// Rising wave left to right.
		offsets = [4][5]int{
			{0, 1, 2, 1, 0},
			{1, 2, 3, 2, 1},
			{2, 3, 2, 1, 0},
			{1, 2, 1, 0, 1},
		}
	case motionModeAlert:
		// Erratic vibration.
		offsets = [4][5]int{
			{2, 0, 4, 0, 3},
			{0, 4, 0, 4, 1},
			{4, 0, 3, 0, 4},
			{0, 3, 0, 2, 1},
		}
	case motionModeReview:
		// Slow peak scan left to right — thoughtful rhythm.
		offsets = [4][5]int{
			{2, 1, 0, 0, 0},
			{1, 2, 1, 0, 0},
			{0, 1, 2, 1, 0},
			{0, 0, 1, 2, 1},
		}
	default:
		// Soft breathing.
		offsets = [4][5]int{
			{0, 1, 0, 1, 0},
			{1, 0, 1, 0, 1},
			{0, 1, 0, 1, 0},
			{1, 0, 1, 0, 1},
		}
	}
	o := offsets[frame%4]
	clamp := func(v int) string {
		if v < 0 {
			return bars[0]
		}
		if v > 7 {
			return bars[7]
		}
		return bars[v]
	}
	return clamp(level+o[0]) + clamp(level+o[1]) + clamp(level+o[2]) + clamp(level+o[3]) + clamp(level+o[4])
}
