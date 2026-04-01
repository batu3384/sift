package platform

import (
	"context"

	"github.com/batuhanyuksel/sift/internal/domain"
)

type CuratedRoots struct {
	Temp            []string
	Logs            []string
	Developer       []string
	Browser         []string
	Installer       []string
	PackageManager  []string
	AppSupport      []string
	RecentItems     []string
	SystemUpdate    []string
	CloudOffice     []string // Dropbox, OneDrive, iCloud, Google Drive
	Virtualization  []string // Docker, VMware, Parallels, VirtualBox
	DeviceBackups   []string // iOS device backups
	TimeMachine     []string // Time Machine backups
	// New comprehensive categories
	FontCache       []string // Font cache files
	PrintSpooler    []string // Print spooler
	Xcode           []string // Xcode caches
	Unity           []string // Unity caches
	Unreal          []string // Unreal Engine caches
	Android         []string // Android SDK and build cache
	Rust            []string // Rust cargo cache
	NodeModules     []string // node_modules
	PythonCache     []string // Python cache
	GoCache         []string // Go cache
	Fonts           []string // User fonts
	Diagnostics     []string // Diagnostics logs
	Media           []string // Media caches
}

type Diagnostic struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Adapter interface {
	Name() string
	CuratedRoots() CuratedRoots
	ProtectedPaths() []string
	ResolveTargets([]string) []string
	ListApps(context.Context, bool) ([]domain.AppEntry, error)
	DiscoverRemnants(context.Context, domain.AppEntry) ([]string, []string, error)
	MaintenanceTasks(context.Context) []domain.MaintenanceTask
	Diagnostics(context.Context) []Diagnostic
	IsAdminPath(path string) bool
	IsFileInUse(context.Context, string) bool
	IsProcessRunning(...string) bool
}
