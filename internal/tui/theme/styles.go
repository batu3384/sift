// Package theme provides styling constants and utilities for the SIFT TUI.
// It defines the visual language used across all TUI components.
package theme

import "github.com/charmbracelet/lipgloss"

// Style definitions - base lipgloss styles used throughout the TUI.
var (
	TitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F3EBDD"))
	HeaderStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E4C389"))
	PanelTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A7C5CF"))
	PanelMetaStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7E8D95"))
	SafeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#8FC5A0"))
	ReviewStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#D8A66A"))
	HighStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#D97A68"))
	MutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7E8D95"))
	FooterStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#A7C5CF"))
	FooterBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#B5C5CC")).Background(lipgloss.Color("#0F1418")).BorderTop(true).BorderForeground(lipgloss.Color("#2C3A41")).Padding(0, 1)
	InfoBarStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F3EBDD")).Background(lipgloss.Color("#172027")).Padding(0, 1)
	ErrorBarStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1C1B8")).Background(lipgloss.Color("#241614")).Padding(0, 1)
	SelectedLine     = lipgloss.NewStyle().Foreground(lipgloss.Color("#0F1418")).Background(lipgloss.Color("#9CB8C2")).Bold(true)
	AppStyle         = lipgloss.NewStyle().Padding(0, 1)
	TopBarStyle      = lipgloss.NewStyle().BorderBottom(true).BorderForeground(lipgloss.Color("#2C3A41")).PaddingBottom(0)
	PanelStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#34505A")).Background(lipgloss.Color("#11181C")).Padding(0, 1)
	ActivePanelStyle = PanelStyle.BorderForeground(lipgloss.Color("#6C909D")).Background(lipgloss.Color("#152027"))
	CardStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#304048")).Background(lipgloss.Color("#141C22")).Padding(0, 1)
	CompactCardStyle = lipgloss.NewStyle().Padding(0, 1)
	KeyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#0F1418")).Background(lipgloss.Color("#D5AE73")).Bold(true).Padding(0, 1)
	KeyTextStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7E8D95"))
	WordmarkStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F3EBDD"))
	RailStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#A7C5CF")).Bold(true)
	BrandBoxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#47616B")).Padding(0, 1)
	SafeBadgeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#0F1418")).Background(lipgloss.Color("#8FC5A0")).Bold(true).Padding(0, 1)
	ReviewBadgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#0F1418")).Background(lipgloss.Color("#D8A66A")).Bold(true).Padding(0, 1)
	HighBadgeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#0F1418")).Background(lipgloss.Color("#D97A68")).Bold(true).Padding(0, 1)
	SafeTokenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8FC5A0")).Bold(true)
	ReviewTokenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D8A66A")).Bold(true)
	HighTokenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#D97A68")).Bold(true)
	CardLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7E8D95"))
	AccentFrameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#54717B"))
)