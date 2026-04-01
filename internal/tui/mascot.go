package tui

import "strings"

// mascotFrame renders the 6-line × 7-char animated SIFT mascot.
//
// Yüz ifadesi (gözler + ağız) motion moduna göre değişir:
//   - idle:     ◡ ◡  ‿   → rahat, gözler art arda kırpar
//   - loading:  ● ●  ○   → dikkatli, bekliyor
//   - progress: ◉/● ○    → gözler sol-sağ tarar, çalışıyor
//   - alert:    ◈ ◈  ∧   → endişeli
//   - review:   ◆/◇ ~    → düşünceli
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
		b("╭─────╮"),
		b("│") + " " + eye1 + " " + eye2 + " " + b("│"),
		b("│") + "  " + mouthR + "  " + b("│"),
		b("╰──┬──╯"),
		b("   │   "),
		" " + barR + " ",
	}
	return strings.Join(lines, "\n")
}

// mascotExpression returns (eye1, eye2, mouth, mouthTone) for the given frame and mode.
func mascotExpression(frame int, mode motionMode) (eye1, eye2, mouth, tone string) {
	switch mode {
	case motionModeIdle:
		// Rahat — art arda sol/sağ göz kırpar.
		mouth, tone = "‿", "safe"
		switch frame {
		case 1:
			eye1, eye2 = "─", "◡"
		case 3:
			eye1, eye2 = "◡", "─"
		default:
			eye1, eye2 = "◡", "◡"
		}
	case motionModeLoading:
		// Dikkatli — veri bekleniyor.
		mouth, tone = "○", "muted"
		if frame%2 == 0 {
			eye1, eye2 = "●", "●"
		} else {
			eye1, eye2 = "◉", "●"
		}
	case motionModeProgress:
		// Aktif tarama — sol-sağ gözler değişir.
		mouth, tone = "○", "review"
		if frame%2 == 0 {
			eye1, eye2 = "◉", "●"
		} else {
			eye1, eye2 = "●", "◉"
		}
	case motionModeAlert:
		// Endişeli — sabit bakış.
		eye1, eye2 = "◈", "◈"
		mouth, tone = "∧", "high"
	case motionModeReview:
		// Düşünceli — elmas gözler titreşir.
		mouth, tone = "~", "muted"
		if frame%2 == 0 {
			eye1, eye2 = "◇", "◆"
		} else {
			eye1, eye2 = "◆", "◇"
		}
	default:
		eye1, eye2 = "●", "●"
		mouth, tone = "~", "muted"
	}
	return
}

// mascotActivityBars returns a 5-character Unicode bar string that pulses with
// CPU / progress load. Pattern changes based on motion mode for visual variety.
func mascotActivityBars(cpuPercent float64, frame int, mode motionMode) string {
	bars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	level := int(cpuPercent / 14.3) // 0-100 → 0-7
	if level > 7 {
		level = 7
	}

	// 5-pozisyon dalga örüntüleri — her frame için farklı yükseklik ofseti.
	var offsets [4][5]int
	switch mode {
	case motionModeProgress, motionModeLoading:
		// Soldan sağa yükselen dalga.
		offsets = [4][5]int{
			{0, 1, 2, 1, 0},
			{1, 2, 3, 2, 1},
			{2, 3, 2, 1, 0},
			{1, 2, 1, 0, 1},
		}
	case motionModeAlert:
		// Düzensiz titreşim.
		offsets = [4][5]int{
			{2, 0, 3, 0, 2},
			{0, 3, 0, 3, 0},
			{3, 0, 2, 0, 3},
			{0, 2, 0, 2, 0},
		}
	case motionModeReview:
		// Soldan sağa yavaş tepe tarama — düşünceli ritim.
		offsets = [4][5]int{
			{2, 1, 0, 0, 0},
			{1, 2, 1, 0, 0},
			{0, 1, 2, 1, 0},
			{0, 0, 1, 2, 1},
		}
	default:
		// Yumuşak nefes alma.
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
