package domain

import "time"

type Category string

const (
	CategoryTempFiles        Category = "temp_files"
	CategoryDeveloperCaches  Category = "developer_caches"
	CategoryBrowserData      Category = "browser_data"
	CategoryInstallerLeft    Category = "installer_leftovers"
	CategoryAppLeftovers     Category = "app_leftovers"
	CategoryPackageCaches    Category = "package_manager_caches"
	CategoryLogs             Category = "logs"
	CategorySystemClutter    Category = "safe_system_clutter"
	CategoryProjectArtifacts Category = "project_artifacts"
	CategoryLargeFiles       Category = "large_files"
	CategoryDiskUsage        Category = "disk_usage"
	CategoryMaintenance      Category = "maintenance"
	// Mole-compatible categories
	CategoryCloudOffice    Category = "cloud_office"
	CategoryVirtualization Category = "virtualization"
	CategoryDeviceBackups  Category = "device_backups"
	CategoryTimeMachine    Category = "time_machine"
	// Additional categories from Mole
	CategoryMavenCache     Category = "maven_cache"
	CategoryIPFS           Category = "ipfs_node"
	CategorySystemCaches   Category = "system_caches"
	CategoryTrash          Category = "trash"
	// New comprehensive categories
	CategoryFontCache      Category = "font_cache"
	CategoryPrintSpooler   Category = "print_spooler"
	CategoryXcode          Category = "xcode"
	CategoryUnity          Category = "unity"
	CategoryUnreal         Category = "unreal"
	CategoryAndroid        Category = "android"
	CategoryRust           Category = "rust"
	CategoryNode           Category = "node_modules"
	CategoryPython         Category = "python_cache"
	CategoryGo             Category = "go_cache"
	CategoryFonts          Category = "fonts"
	CategoryDiagnostics    Category = "diagnostics"
	CategoryMedia          Category = "media_cache"
)

type Risk string

const (
	RiskSafe   Risk = "safe"
	RiskReview Risk = "review"
	RiskHigh   Risk = "high"
)

type Action string

const (
	ActionTrash     Action = "trash"
	ActionPermanent Action = "permanent_delete"
	ActionAdvisory  Action = "advisory"
	ActionNative    Action = "native_uninstall"
	ActionCommand   Action = "managed_command"
	ActionSkip      Action = "skip"
)

type FindingStatus string

const (
	StatusPlanned   FindingStatus = "planned"
	StatusSkipped   FindingStatus = "skipped"
	StatusDeleted   FindingStatus = "deleted"
	StatusCompleted FindingStatus = "completed"
	StatusAdvisory  FindingStatus = "advisory"
	StatusFailed    FindingStatus = "failed"
	StatusProtected FindingStatus = "protected"
)

type RecoveryHint struct {
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

type ProtectionReason string

const (
	ProtectionNone                ProtectionReason = ""
	ProtectionAdvisoryOnly        ProtectionReason = "advisory_only"
	ProtectionCommandExcluded     ProtectionReason = "command_excluded"
	ProtectionEmptyPath           ProtectionReason = "empty_path"
	ProtectionRelativePath        ProtectionReason = "relative_path"
	ProtectionControlChars        ProtectionReason = "control_chars"
	ProtectionTraversal           ProtectionReason = "path_traversal"
	ProtectionCriticalRoot        ProtectionReason = "critical_root"
	ProtectionMissingPath         ProtectionReason = "missing_path"
	ProtectionSymlink             ProtectionReason = "symlink_blocked"
	ProtectionProtectedPath       ProtectionReason = "protected_path"
	ProtectionOutsideAllowedRoots ProtectionReason = "outside_allowed_roots"
	ProtectionAdminRequired       ProtectionReason = "admin_required"
	ProtectionUnsafeCommand       ProtectionReason = "unsafe_command"
	ProtectionRunningApp          ProtectionReason = "running_app"
)

type ProtectionState string

const (
	ProtectionStateUnprotected      ProtectionState = "unprotected"
	ProtectionStateCommandProtected ProtectionState = "command_protected"
	ProtectionStateUserProtected    ProtectionState = "user_protected"
	ProtectionStateSystemProtected  ProtectionState = "system_protected"
	ProtectionStateSafeException    ProtectionState = "safe_exception"
)

type PolicyDecision struct {
	Allowed       bool             `json:"allowed"`
	Reason        ProtectionReason `json:"reason,omitempty"`
	Message       string           `json:"message,omitempty"`
	RequiresTrash bool             `json:"requires_trash,omitempty"`
}

type ProtectionExplanation struct {
	Path             string          `json:"path"`
	Command          string          `json:"command,omitempty"`
	State            ProtectionState `json:"state"`
	Message          string          `json:"message"`
	CommandMatches   []string        `json:"command_matches,omitempty"`
	UserMatches      []string        `json:"user_matches,omitempty"`
	SystemMatches    []string        `json:"system_matches,omitempty"`
	FamilyMatches    []string        `json:"family_matches,omitempty"`
	ExceptionMatches []string        `json:"exception_matches,omitempty"`
}

type Fingerprint struct {
	Mode         uint32    `json:"mode"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	ContentHash  string    `json:"content_hash,omitempty"`
}

type DirectoryPreviewFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type DirectoryPreview struct {
	Path        string                 `json:"path"`
	Unavailable bool                   `json:"unavailable,omitempty"`
	Total       int                    `json:"total,omitempty"`
	Dirs        int                    `json:"dirs,omitempty"`
	Files       int                    `json:"files,omitempty"`
	Names       []string               `json:"names,omitempty"`
	DirNames    []string               `json:"dir_names,omitempty"`
	FileSamples []DirectoryPreviewFile `json:"file_samples,omitempty"`
}

type Finding struct {
	ID             string         `json:"id"`
	RuleID         string         `json:"rule_id"`
	Name           string         `json:"name"`
	Category       Category       `json:"category"`
	Path           string         `json:"path"`
	DisplayPath    string         `json:"display_path"`
	Risk           Risk           `json:"risk"`
	Bytes          int64          `json:"bytes"`
	RequiresAdmin  bool           `json:"requires_admin"`
	Action         Action         `json:"action"`
	Recovery       RecoveryHint   `json:"recovery"`
	Status         FindingStatus  `json:"status"`
	LastModified   time.Time      `json:"last_modified,omitempty"`
	Fingerprint    Fingerprint    `json:"fingerprint"`
	Source         string         `json:"source,omitempty"`
	NativeCommand  string         `json:"native_command,omitempty"`
	CommandPath    string         `json:"command_path,omitempty"`
	CommandArgs    []string       `json:"command_args,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
	Capability     string         `json:"capability,omitempty"`
	TaskPhase      string         `json:"task_phase,omitempty"`
	TaskImpact     string         `json:"task_impact,omitempty"`
	TaskVerify     []string       `json:"task_verify,omitempty"`
	SuggestedBy    []string       `json:"suggested_by,omitempty"`
	Policy         PolicyDecision `json:"policy"`
}

type Totals struct {
	ItemCount   int   `json:"item_count"`
	Bytes       int64 `json:"bytes"`
	SafeBytes   int64 `json:"safe_bytes"`
	ReviewBytes int64 `json:"review_bytes"`
	HighBytes   int64 `json:"high_bytes"`
}

type ExecutionPlan struct {
	ScanID               string           `json:"scan_id"`
	Command              string           `json:"command"`
	Profile              string           `json:"profile,omitempty"`
	Platform             string           `json:"platform"`
	CreatedAt            time.Time        `json:"created_at"`
	PlanState            string           `json:"plan_state"`
	DryRun               bool             `json:"dry_run"`
	RequiresConfirmation bool             `json:"requires_confirmation"`
	Warnings             []string         `json:"warnings"`
	Items                []Finding        `json:"items"`
	Totals               Totals           `json:"totals"`
	Targets              []string         `json:"targets,omitempty"`
	Policy               ProtectionPolicy `json:"policy"`
}

type OperationResult struct {
	FindingID string           `json:"finding_id"`
	Path      string           `json:"path"`
	Status    FindingStatus    `json:"status"`
	Reason    ProtectionReason `json:"reason,omitempty"`
	Message   string           `json:"message,omitempty"`
	Bytes     int64            `json:"bytes,omitempty"`
}

type ExecutionProgress struct {
	ScanID       string          `json:"scan_id"`
	StartedAt    time.Time       `json:"started_at"`
	Current      int             `json:"current"`
	Completed    int             `json:"completed"`
	Total        int             `json:"total"`
	Event        ProgressEvent   `json:"event,omitempty"`
	Phase        ProgressPhase   `json:"phase"`
	Step         string          `json:"step,omitempty"`
	Detail       string          `json:"detail,omitempty"`
	SectionKey   string          `json:"section_key,omitempty"`
	SectionLabel string          `json:"section_label,omitempty"`
	SectionIndex int             `json:"section_index,omitempty"`
	SectionTotal int             `json:"section_total,omitempty"`
	SectionDone  int             `json:"section_done,omitempty"`
	SectionItems int             `json:"section_items,omitempty"`
	SectionBytes int64           `json:"section_bytes,omitempty"`
	Item         Finding         `json:"item"`
	Result       OperationResult `json:"result"`
}

type ProgressEvent string

const (
	ProgressEventItem    ProgressEvent = "item"
	ProgressEventSection ProgressEvent = "section"
)

type ProgressPhase string

const (
	ProgressPhaseStarting  ProgressPhase = "starting"
	ProgressPhasePreparing ProgressPhase = "preparing"
	ProgressPhaseRunning   ProgressPhase = "running"
	ProgressPhaseVerifying ProgressPhase = "verifying"
	ProgressPhaseFinished  ProgressPhase = "finished"
)

type ExecutionResult struct {
	ID               string            `json:"id"`
	ScanID           string            `json:"scan_id"`
	StartedAt        time.Time         `json:"started_at"`
	FinishedAt       time.Time         `json:"finished_at"`
	Items            []OperationResult `json:"items"`
	Warnings         []string          `json:"warnings"`
	FollowUpCommands []string          `json:"follow_up_commands,omitempty"`
}

type ProtectionPolicy struct {
	ProtectedPaths          []string `json:"protected_paths"`
	Command                 string   `json:"command,omitempty"`
	CommandProtectedPaths   []string `json:"command_protected_paths,omitempty"`
	UserProtectedPaths      []string `json:"user_protected_paths,omitempty"`
	SystemProtectedPaths    []string `json:"system_protected_paths,omitempty"`
	FamilyProtectedPaths    []string `json:"family_protected_paths,omitempty"`
	ProtectedFamilies       []string `json:"protected_families,omitempty"`
	ProtectedPathExceptions []string `json:"protected_path_exceptions,omitempty"`
	AllowedRoots            []string `json:"allowed_roots,omitempty"`
	TrashOnly               bool     `json:"trash_only"`
	AllowAdmin              bool     `json:"allow_admin"`
	BlockSymlinks           bool     `json:"block_symlinks"`
	// WhitelistPaths are user-defined paths to exclude from cleaning (Mole-style)
	WhitelistPaths []string `json:"whitelist_paths,omitempty"`
}

type MaintenanceTask struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Risk              Risk     `json:"risk"`
	Capability        string   `json:"capability,omitempty"`
	Phase             string   `json:"phase,omitempty"`
	EstimatedImpact   string   `json:"estimated_impact,omitempty"`
	RequiresApp       []string `json:"requires_app,omitempty"`
	Steps             []string `json:"steps"`
	Verification      []string `json:"verification,omitempty"`
	SuggestedByChecks []string `json:"suggested_by_checks,omitempty"`
	Action            Action   `json:"action,omitempty"`
	Paths             []string `json:"paths,omitempty"`
	PathGlobs         []string `json:"path_globs,omitempty"`
	CommandPath       string   `json:"command_path,omitempty"`
	CommandArgs       []string `json:"command_args,omitempty"`
	TimeoutSeconds    int      `json:"timeout_seconds,omitempty"`
	RequiresAdmin     bool     `json:"requires_admin,omitempty"`
	RecoveryCommands  []string `json:"recovery_commands,omitempty"`
}

type ProtectionScope struct {
	Command string   `json:"command"`
	Paths   []string `json:"paths"`
}

type ProtectedFamily struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CheckGroup string

const (
	CheckGroupSecurity CheckGroup = "security"
	CheckGroupUpdates  CheckGroup = "updates"
	CheckGroupConfig   CheckGroup = "config"
	CheckGroupHealth   CheckGroup = "health"
)

type CheckItem struct {
	ID               string     `json:"id"`
	Group            CheckGroup `json:"group"`
	Name             string     `json:"name"`
	Status           string     `json:"status"`
	Message          string     `json:"message"`
	AutofixAvailable bool       `json:"autofix_available"`
	Commands         []string   `json:"commands,omitempty"`
}

type CheckSummary struct {
	Total       int `json:"total"`
	OK          int `json:"ok"`
	Warn        int `json:"warn"`
	Autofixable int `json:"autofixable"`
}

type CheckReport struct {
	CreatedAt time.Time    `json:"created_at"`
	Platform  string       `json:"platform"`
	Items     []CheckItem  `json:"items"`
	Summary   CheckSummary `json:"summary"`
}

type AppEntry struct {
	Name                  string    `json:"name"`
	DisplayName           string    `json:"display_name"`
	BundlePath            string    `json:"bundle_path,omitempty"`
	SupportPaths          []string  `json:"support_paths,omitempty"`
	Origin                string    `json:"origin,omitempty"`
	RequiresAdmin         bool      `json:"requires_admin"`
	UninstallHint         string    `json:"uninstall_hint,omitempty"`
	UninstallCommand      string    `json:"uninstall_command,omitempty"`
	QuietUninstallCommand string    `json:"quiet_uninstall_command,omitempty"`
	LastModified          time.Time `json:"last_modified,omitempty"`
	ApproxBytes           int64     `json:"approx_bytes,omitempty"`
	Sensitive             bool      `json:"sensitive,omitempty"`
	FamilyMatches         []string  `json:"family_matches,omitempty"`
}
