// Package design provides the modern design system for SIFT TUI.
// This file contains all design tokens and core styles.
package design

import "github.com/charmbracelet/lipgloss"

// Design tokens - centralized color system
var (
	// Primary colors - warm cream palette
	ColorBackground = lipgloss.Color("#0D1117") // Deep dark background
	ColorSurface    = lipgloss.Color("#161B22") // Elevated surface
	ColorOverlay   = lipgloss.Color("#21262D") // Overlay/modal background

	// Text colors
	ColorTextPrimary   = lipgloss.Color("#E6EDF3") // Primary text (high contrast)
	ColorTextSecondary = lipgloss.Color("#8B949E") // Secondary text
	ColorTextMuted     = lipgloss.Color("#6E7681")  // Muted/disabled text

	// Accent colors
	ColorAccentPrimary   = lipgloss.Color("#58A6FF") // Primary accent (blue)
	ColorAccentSecondary = lipgloss.Color("#F0883E") // Secondary accent (orange)

	// Semantic colors - status
	ColorSuccess = lipgloss.Color("#3FB950") // Safe/completed
	ColorWarning = lipgloss.Color("#D29922") // Review/pending
	ColorDanger  = lipgloss.Color("#F85149") // High risk/error

	// Border colors
	ColorBorderDefault = lipgloss.Color("#30363D")
	ColorBorderMuted   = lipgloss.Color("#21262D")
	ColorBorderAccent  = lipgloss.Color("#58A6FF")

	// Selection
	ColorSelectionBg   = lipgloss.Color("#1F6FEB")
	ColorSelectionText = lipgloss.Color("#FFFFFF")

	// Interactive states - calculated blended color for hover
	ColorHoverBg   = lipgloss.Color("#1A5EB8") // 30% darker than #1F6FEB
	ColorActiveBg  = lipgloss.Color("#1F6FEB")
)

// Typography styles
var (
	// Display - large headings
	DisplayStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextPrimary)

	// Title - section headers
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTextPrimary)

	// Heading - subsection headers
	HeadingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextSecondary)

	// Body - regular text
	BodyStyle = lipgloss.NewStyle().
			Foreground(ColorTextPrimary)

	// Caption - small/meta text
	CaptionStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	// Code - monospace
	CodeStyle = lipgloss.NewStyle().
			Foreground(ColorAccentPrimary)
)

// Surface styles - panels, cards, containers
var (
	SurfaceStyle = lipgloss.NewStyle().
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderDefault).
			Padding(1, 2)

	SurfaceElevatedStyle = lipgloss.NewStyle().
				Background(ColorOverlay).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorderMuted).
				Padding(1, 2)

	SurfaceAccentStyle = lipgloss.NewStyle().
				Background(ColorOverlay).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorderAccent).
				Padding(1, 2)
)

// Interactive element styles
var (
	ButtonStyle = lipgloss.NewStyle().
			Foreground(ColorTextPrimary).
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderDefault).
			Padding(0, 2).
			Bold(true)

	ButtonPrimaryStyle = lipgloss.NewStyle().
				Foreground(ColorTextPrimary).
				Background(ColorAccentPrimary).
				Border(lipgloss.RoundedBorder()).
				Padding(0, 2).
				Bold(true)

	ButtonHoverStyle = lipgloss.NewStyle().
				Foreground(ColorSelectionText).
				Background(ColorHoverBg).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorderAccent).
				Padding(0, 2).
				Bold(true)
)

// Status indicator styles
var (
	StatusSuccessStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	StatusWarningStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	StatusDangerStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	StatusNeutralStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted)
)

// Badge styles
var (
	BadgeSuccessStyle = lipgloss.NewStyle().
				Foreground(ColorSurface).
				Background(ColorSuccess).
				Bold(true).
				Padding(0, 1)

	BadgeWarningStyle = lipgloss.NewStyle().
				Foreground(ColorSurface).
				Background(ColorWarning).
				Bold(true).
				Padding(0, 1)

	BadgeDangerStyle = lipgloss.NewStyle().
				Foreground(ColorSurface).
				Background(ColorDanger).
				Bold(true).
				Padding(0, 1)

	BadgeNeutralStyle = lipgloss.NewStyle().
				Foreground(ColorTextPrimary).
				Background(ColorSurface).
				Bold(true).
				Padding(0, 1)
)

// Key binding styles
var (
	KeyStyle = lipgloss.NewStyle().
			Foreground(ColorSurface).
			Background(ColorAccentPrimary).
			Bold(true).
			Padding(0, 1)

	KeyTextStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)
)

// Progress bar styles
var (
	ProgressBarFill   = lipgloss.NewStyle().Foreground(ColorAccentPrimary)
	ProgressBarEmpty  = lipgloss.NewStyle().Foreground(ColorBorderMuted)
	ProgressBarBg     = lipgloss.NewStyle().Background(ColorSurface)
)